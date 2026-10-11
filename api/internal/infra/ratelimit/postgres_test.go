package ratelimit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// fakeBuckets is the bucketStore test double: it either delegates to scripted results or fails.
type fakeBuckets struct {
	mu        sync.Mutex
	takeErr   error
	blockTake bool
	hits      map[string]int64
	windowEnd time.Time
	resetErr  error
	resets    int
	pruned    int64
	pruneErr  error
	prunes    atomic.Int64
}

func (f *fakeBuckets) take(ctx context.Context, keyHash []byte, maxPlusOne int64, _ time.Duration) (int64, time.Time, error) {
	if f.blockTake {
		<-ctx.Done()
		return 0, time.Time{}, ctx.Err()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.takeErr != nil {
		return 0, time.Time{}, f.takeErr
	}
	if f.hits == nil {
		f.hits = map[string]int64{}
	}
	f.hits[string(keyHash)] = min(f.hits[string(keyHash)]+1, maxPlusOne)
	return f.hits[string(keyHash)], f.windowEnd, nil
}

func (f *fakeBuckets) reset(context.Context, []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resets++
	return f.resetErr
}

func (f *fakeBuckets) pruneExpired(context.Context, int) (int64, error) {
	f.prunes.Add(1)
	return f.pruned, f.pruneErr
}

func newFakeStore(t *testing.T, buckets *fakeBuckets, options PostgresOptions, rule Rule) *PostgresStore {
	t.Helper()
	stores, err := newSharedStores(buckets, options)
	if err != nil {
		t.Fatalf("newSharedStores: %v", err)
	}
	// Keep opportunistic cleanup out of decision tests unless a test opts in.
	stores.shared.nextCleanup.Store(time.Now().Add(time.Hour).UnixNano())
	store, ok := stores.New(rule).(*PostgresStore)
	if !ok {
		t.Fatalf("shared factory returned %T, want *PostgresStore", stores.New(rule))
	}
	return store
}

func failureCount(t *testing.T, rule, reason, decision string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "rate_limit_store_failures_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["rule"] == rule && labels["reason"] == reason && labels["decision"] == decision {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func TestStoresWithoutSharedBackendReturnProcessLocalStores(t *testing.T) {
	var nilStores *Stores
	for _, stores := range []*Stores{nilStores, NewMemoryStores()} {
		if stores.Shared() {
			t.Fatal("memory factory must not report a shared backend")
		}
		if _, ok := stores.New(Rule{Name: "global", Max: 1, Window: time.Minute}).(*MemoryStore); !ok {
			t.Fatal("memory factory must return a *MemoryStore")
		}
	}
}

func TestNewSharedStoresRejectsUnknownFailurePolicy(t *testing.T) {
	if _, err := newSharedStores(&fakeBuckets{}, PostgresOptions{Failure: "retry"}); err == nil {
		t.Fatal("expected unknown failure policy to be rejected")
	}
	if _, err := NewPostgresStores(nil, PostgresOptions{}); err == nil {
		t.Fatal("expected a nil database to be rejected")
	}
	stores, err := newSharedStores(&fakeBuckets{}, PostgresOptions{})
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	if stores.shared.failure != FailureLocal || stores.shared.timeout != DefaultStoreTimeout {
		t.Fatalf("defaults = (%s, %s), want (local, %s)", stores.shared.failure, stores.shared.timeout, DefaultStoreTimeout)
	}
}

func TestPostgresStoreMapsSaturatedHitsToFixedWindowDecisions(t *testing.T) {
	windowEnd := time.Unix(1_700_000_060, 0)
	store := newFakeStore(t, &fakeBuckets{windowEnd: windowEnd}, PostgresOptions{}, Rule{Name: "test", Max: 2, Window: time.Minute})
	ctx := context.Background()

	want := []struct {
		allowed   bool
		remaining int
	}{{true, 1}, {true, 0}, {false, 0}, {false, 0}}
	for index, expected := range want {
		allowed, remaining, resetAt := store.Take(ctx, "client")
		if allowed != expected.allowed || remaining != expected.remaining || !resetAt.Equal(windowEnd) {
			t.Fatalf("take %d = (%v, %d, %s), want (%v, %d, %s)",
				index+1, allowed, remaining, resetAt, expected.allowed, expected.remaining, windowEnd)
		}
	}
}

func TestPostgresStoreHashesNamespacedKeys(t *testing.T) {
	buckets := &fakeBuckets{}
	login := newFakeStore(t, buckets, PostgresOptions{}, Rule{Name: "auth_login_ip", Max: 1, Window: time.Minute})
	register := newFakeStore(t, buckets, PostgresOptions{}, Rule{Name: "auth_register_ip", Max: 1, Window: time.Minute})
	rewindowed := newFakeStore(t, buckets, PostgresOptions{}, Rule{Name: "auth_login_ip", Max: 1, Window: time.Hour})

	key := "auth:login:ip:203.0.113.7"
	digest := login.keyHash(key)
	expected := sha256.Sum256([]byte("auth_login_ip\x0060000000000\x00" + key))
	if !bytes.Equal(digest, expected[:]) {
		t.Fatalf("key hash = %x, want %x", digest, expected)
	}
	if len(digest) != sha256.Size || bytes.Contains(digest, []byte("203.0.113.7")) {
		t.Fatalf("key hash must be a 32-byte digest without the raw key, got %x", digest)
	}
	if bytes.Equal(digest, register.keyHash(key)) || bytes.Equal(digest, rewindowed.keyHash(key)) {
		t.Fatal("rules and windows must not share buckets")
	}
}

func TestPostgresStoreFailurePolicies(t *testing.T) {
	failure := errors.New("connection refused")
	rule := Rule{Name: "policy_test", Max: 2, Window: time.Minute}

	t.Run("open allows", func(t *testing.T) {
		before := failureCount(t, rule.Name, failureReasonError, "allow")
		store := newFakeStore(t, &fakeBuckets{takeErr: failure}, PostgresOptions{Failure: FailureOpen}, rule)
		for range 5 {
			allowed, remaining, resetAt := store.Take(context.Background(), "client")
			if !allowed || remaining != rule.Max-1 || time.Until(resetAt) <= 0 {
				t.Fatalf("fail-open take = (%v, %d, %s), want allowed with a future reset", allowed, remaining, resetAt)
			}
		}
		if got := failureCount(t, rule.Name, failureReasonError, "allow") - before; got != 5 {
			t.Fatalf("fail-open failures recorded = %v, want 5", got)
		}
	})

	t.Run("closed denies with a short retry", func(t *testing.T) {
		before := failureCount(t, rule.Name, failureReasonError, "deny")
		store := newFakeStore(t, &fakeBuckets{takeErr: failure}, PostgresOptions{Failure: FailureClosed}, rule)
		allowed, remaining, resetAt := store.Take(context.Background(), "client")
		if allowed || remaining != 0 || time.Until(resetAt) <= 0 || time.Until(resetAt) > closedRetryAfter {
			t.Fatalf("fail-closed take = (%v, %d, %s), want denied within %s", allowed, remaining, resetAt, closedRetryAfter)
		}
		if got := failureCount(t, rule.Name, failureReasonError, "deny") - before; got != 1 {
			t.Fatalf("fail-closed failures recorded = %v, want 1", got)
		}
	})

	t.Run("local enforces a process-local budget", func(t *testing.T) {
		before := failureCount(t, rule.Name, failureReasonError, "local")
		buckets := &fakeBuckets{takeErr: failure}
		store := newFakeStore(t, buckets, PostgresOptions{Failure: FailureLocal}, rule)
		results := make([]bool, 0, 3)
		for range 3 {
			allowed, _, _ := store.Take(context.Background(), "client")
			results = append(results, allowed)
		}
		if !results[0] || !results[1] || results[2] {
			t.Fatalf("local fallback decisions = %v, want [true true false]", results)
		}
		if got := failureCount(t, rule.Name, failureReasonError, "local") - before; got != 3 {
			t.Fatalf("local fallback failures recorded = %v, want 3", got)
		}

		// Reset clears the outage-time local bucket as well as the shared one.
		store.Reset(context.Background(), "client")
		if allowed, _, _ := store.Take(context.Background(), "client"); !allowed {
			t.Fatal("reset must clear the local fallback bucket")
		}
		if buckets.resets != 1 {
			t.Fatalf("shared resets = %d, want 1", buckets.resets)
		}
	})
}

func TestPostgresStoreTimeoutFallsBackWithinBudget(t *testing.T) {
	rule := Rule{Name: "timeout_test", Max: 1, Window: time.Minute}
	before := failureCount(t, rule.Name, failureReasonTimeout, "deny")
	store := newFakeStore(t, &fakeBuckets{blockTake: true}, PostgresOptions{
		Timeout: 20 * time.Millisecond,
		Failure: FailureClosed,
	}, rule)

	started := time.Now()
	allowed, _, _ := store.Take(context.Background(), "client")
	elapsed := time.Since(started)
	if allowed {
		t.Fatal("fail-closed timeout must deny")
	}
	if elapsed < 20*time.Millisecond || elapsed > time.Second {
		t.Fatalf("timed-out decision took %s, want about 20ms", elapsed)
	}
	if got := failureCount(t, rule.Name, failureReasonTimeout, "deny") - before; got != 1 {
		t.Fatalf("timeout failures recorded = %v, want 1", got)
	}
}

func TestPostgresStoreRateLimitsOutageWarnings(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	store := newFakeStore(t, &fakeBuckets{takeErr: errors.New("down")}, PostgresOptions{
		Failure: FailureOpen,
		Logger:  logger,
	}, Rule{Name: "warn_test", Max: 1, Window: time.Minute})

	for range 50 {
		store.Take(context.Background(), "client")
	}
	if count := strings.Count(logs.String(), "ratelimit.store_unavailable"); count != 1 {
		t.Fatalf("outage warnings = %d, want 1 per interval:\n%s", count, logs.String())
	}

	clock := time.Now().Add(failureWarnInterval + time.Second)
	store.backend.now = func() time.Time { return clock }
	store.Take(context.Background(), "client")
	if !strings.Contains(logs.String(), "suppressed=49") {
		t.Fatalf("next warning must report suppressed count:\n%s", logs.String())
	}
}

func TestPostgresStoreResetFailureIsRecorded(t *testing.T) {
	rule := Rule{Name: "reset_test", Max: 1, Window: time.Minute}
	before := failureCount(t, rule.Name, failureReasonError, "reset")
	store := newFakeStore(t, &fakeBuckets{resetErr: errors.New("down")}, PostgresOptions{Failure: FailureOpen}, rule)
	store.Reset(context.Background(), "client")
	if got := failureCount(t, rule.Name, failureReasonError, "reset") - before; got != 1 {
		t.Fatalf("reset failures recorded = %v, want 1", got)
	}
}

func TestSharedBackendCleanupIsBoundedAndDrainsBacklog(t *testing.T) {
	buckets := &fakeBuckets{pruned: 10}
	stores, err := newSharedStores(buckets, PostgresOptions{CleanupEvery: time.Minute, CleanupBatch: 10})
	if err != nil {
		t.Fatal(err)
	}
	backend := stores.shared

	backend.cleanupOnce()
	if next := time.Unix(0, backend.nextCleanup.Load()); time.Until(next) > time.Second {
		t.Fatalf("a full batch must schedule the next cleanup immediately, got %s", time.Until(next))
	}

	buckets.pruned = 3
	backend.cleanupOnce()
	if next := time.Unix(0, backend.nextCleanup.Load()); time.Until(next) < 50*time.Second {
		t.Fatalf("a partial batch must wait for the cleanup interval, got %s", time.Until(next))
	}

	// Inside the interval no cleanup starts, however many decisions run.
	store := stores.New(Rule{Name: "cleanup_test", Max: 100, Window: time.Minute})
	prunesBefore := buckets.prunes.Load()
	for range 20 {
		store.Take(context.Background(), "client")
	}
	if got := buckets.prunes.Load() - prunesBefore; got != 0 {
		t.Fatalf("cleanups inside the interval = %d, want 0", got)
	}

	// Once due, concurrent decisions start exactly one cleanup.
	backend.nextCleanup.Store(0)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { store.Take(context.Background(), "client") })
	}
	wg.Wait()
	deadline := time.Now().Add(time.Second)
	for backend.cleaning.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := buckets.prunes.Load() - prunesBefore; got != 1 {
		t.Fatalf("due cleanups = %d, want exactly 1", got)
	}
}

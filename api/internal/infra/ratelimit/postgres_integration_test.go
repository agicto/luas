package ratelimit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fastagi/fastbuild/api/database/migrations"
	testplatform "github.com/fastagi/fastbuild/api/internal/infra/testing"
)

// openBucketTable opens an isolated PostgreSQL schema holding the real migrated bucket table.
func openBucketTable(tb testing.TB) *sql.DB {
	tb.Helper()
	db := testplatform.OpenPostgres(tb, nil)
	if err := migrations.All()[migrations.RateLimitBucketsMigration].Up(db); err != nil {
		tb.Fatalf("migrate rate_limit_buckets: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Fatalf("sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	sqlDB.SetMaxIdleConns(16)
	return sqlDB
}

// newTestPostgresStores builds a shared factory that denies on any store failure, so an
// unexpected fallback shows up as a wrong decision instead of passing silently.
func newTestPostgresStores(tb testing.TB, db *sql.DB) *Stores {
	tb.Helper()
	stores, err := NewPostgresStores(db, PostgresOptions{Timeout: 5 * time.Second, Failure: FailureClosed})
	if err != nil {
		tb.Fatalf("NewPostgresStores: %v", err)
	}
	return stores
}

func TestPostgresStoreIsAtomicAcrossReplicas(t *testing.T) {
	db := openBucketTable(t)
	rule := Rule{Name: "global", Max: 10, Window: time.Minute}
	// Two factories model two API replicas sharing one database.
	replicas := []Store{newTestPostgresStores(t, db).New(rule), newTestPostgresStores(t, db).New(rule)}

	var allowed atomic.Int64
	remainingSeen := make([]atomic.Int64, rule.Max)
	resetAts := make(chan time.Time, 50)
	var wg sync.WaitGroup
	for index := range 50 {
		wg.Go(func() {
			ok, remaining, resetAt := replicas[index%2].Take(context.Background(), "203.0.113.7")
			resetAts <- resetAt
			if ok {
				allowed.Add(1)
				remainingSeen[remaining].Add(1)
			}
		})
	}
	wg.Wait()
	close(resetAts)

	if got := allowed.Load(); got != int64(rule.Max) {
		t.Fatalf("allowed = %d across replicas, want exactly %d", got, rule.Max)
	}
	for remaining := range remainingSeen {
		if count := remainingSeen[remaining].Load(); count != 1 {
			t.Fatalf("remaining=%d observed %d times, want once", remaining, count)
		}
	}
	var first time.Time
	for resetAt := range resetAts {
		if first.IsZero() {
			first = resetAt
		}
		if !resetAt.Equal(first) {
			t.Fatalf("decisions in one window reported different resets: %s and %s", first, resetAt)
		}
	}

	var hits int64
	if err := db.QueryRow(`SELECT hits FROM rate_limit_buckets`).Scan(&hits); err != nil {
		t.Fatal(err)
	}
	if hits != int64(rule.Max)+1 {
		t.Fatalf("denied decisions must saturate at max+1, got hits=%d", hits)
	}
}

func TestPostgresStoreWindowResetsOnDatabaseClock(t *testing.T) {
	db := openBucketTable(t)
	store := newTestPostgresStores(t, db).New(Rule{Name: "window", Max: 2, Window: 250 * time.Millisecond})
	ctx := context.Background()

	_, _, firstReset := store.Take(ctx, "client")
	store.Take(ctx, "client")
	if allowed, remaining, resetAt := store.Take(ctx, "client"); allowed || remaining != 0 || !resetAt.Equal(firstReset) {
		t.Fatalf("over-limit take = (%v, %d, %s), want denied until %s", allowed, remaining, resetAt, firstReset)
	}

	time.Sleep(time.Until(firstReset) + 50*time.Millisecond)
	allowed, remaining, nextReset := store.Take(ctx, "client")
	if !allowed || remaining != 1 || !nextReset.After(firstReset) {
		t.Fatalf("take after window = (%v, %d, %s), want a fresh window after %s", allowed, remaining, nextReset, firstReset)
	}

	// The boundary is exact: a window ending now has expired.
	if _, err := db.Exec(`UPDATE rate_limit_buckets SET hits = 3, window_end = statement_timestamp()`); err != nil {
		t.Fatal(err)
	}
	if allowed, remaining, _ := store.Take(ctx, "client"); !allowed || remaining != 1 {
		t.Fatalf("take at the window boundary = (%v, %d), want a fresh window", allowed, remaining)
	}
}

func TestPostgresStorePersistsOnlyHashedKeys(t *testing.T) {
	db := openBucketTable(t)
	store := newTestPostgresStores(t, db).New(Rule{Name: "auth_login_subject", Max: 3, Window: time.Minute})
	key := "auth:login:subject:person@example.com"
	store.Take(context.Background(), key)

	var stored []byte
	if err := db.QueryRow(`SELECT key_hash FROM rate_limit_buckets`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte("auth_login_subject\x00" + strconv.FormatInt(time.Minute.Nanoseconds(), 10) + "\x00" + key))
	if string(stored) != string(want[:]) {
		t.Fatalf("stored key = %x, want SHA-256 %x", stored, want)
	}

	var leaked bool
	if err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM rate_limit_buckets WHERE position(convert_to('person@example.com', 'UTF8') IN key_hash) > 0)`,
	).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked {
		t.Fatal("raw subject must never be persisted")
	}

	store.Reset(context.Background(), key)
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM rate_limit_buckets`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("reset left %d bucket(s)", rows)
	}
}

func TestPruneExpiredBucketsDeletesOnlyExpiredRowsInBatches(t *testing.T) {
	db := openBucketTable(t)
	stores := newTestPostgresStores(t, db)
	// Keep opportunistic cleanup out of this test so only the explicit prune deletes rows.
	stores.shared.nextCleanup.Store(time.Now().Add(time.Hour).UnixNano())
	store := stores.New(Rule{Name: "prune", Max: 5, Window: time.Hour})
	for index := range 7 {
		store.Take(context.Background(), "client-"+strconv.Itoa(index))
	}
	if _, err := db.Exec(`
		UPDATE rate_limit_buckets SET window_end = statement_timestamp() - interval '1 second'
		WHERE key_hash IN (SELECT key_hash FROM rate_limit_buckets ORDER BY key_hash LIMIT 5)
	`); err != nil {
		t.Fatal(err)
	}

	deleted, err := PruneExpiredBuckets(context.Background(), db, 2, 1)
	if err != nil || deleted != 2 {
		t.Fatalf("one bounded batch = (%d, %v), want (2, nil)", deleted, err)
	}
	deleted, err = PruneExpiredBuckets(context.Background(), db, 2, 10)
	if err != nil || deleted != 3 {
		t.Fatalf("draining prune = (%d, %v), want (3, nil)", deleted, err)
	}

	var live int
	if err := db.QueryRow(`SELECT count(*) FROM rate_limit_buckets WHERE window_end > statement_timestamp()`).Scan(&live); err != nil {
		t.Fatal(err)
	}
	if live != 2 {
		t.Fatalf("live buckets after prune = %d, want 2", live)
	}

	for _, args := range [][2]int{{0, 1}, {maxPruneBatch + 1, 1}, {1, 0}} {
		if _, err := PruneExpiredBuckets(context.Background(), db, args[0], args[1]); err == nil {
			t.Fatalf("PruneExpiredBuckets(batch=%d, maxBatches=%d) must be rejected", args[0], args[1])
		}
	}
}

func TestPostgresStoreOpportunisticCleanupRemovesExpiredRows(t *testing.T) {
	db := openBucketTable(t)
	stores := newTestPostgresStores(t, db)
	stores.shared.nextCleanup.Store(time.Now().Add(time.Hour).UnixNano())
	store := stores.New(Rule{Name: "cleanup", Max: 5, Window: time.Hour})
	for index := range 3 {
		store.Take(context.Background(), "client-"+strconv.Itoa(index))
	}
	if _, err := db.Exec(`UPDATE rate_limit_buckets SET window_end = statement_timestamp() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}

	stores.shared.nextCleanup.Store(0)
	store.Take(context.Background(), "fresh")
	deadline := time.Now().Add(5 * time.Second)
	for {
		var rows int
		if err := db.QueryRow(`SELECT count(*) FROM rate_limit_buckets`).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if rows == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("opportunistic cleanup left %d bucket(s), want only the live one", rows)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPostgresStoreFallsBackWhenTheDecisionExceedsItsTimeout(t *testing.T) {
	db := openBucketTable(t)
	rule := Rule{Name: "lock_timeout", Max: 5, Window: time.Minute}
	stores, err := NewPostgresStores(db, PostgresOptions{Timeout: 50 * time.Millisecond, Failure: FailureOpen})
	if err != nil {
		t.Fatal(err)
	}
	store := stores.New(rule)
	store.Take(context.Background(), "client")

	// Another session holds the bucket's row lock, so the next upsert cannot finish in time.
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`SELECT 1 FROM rate_limit_buckets FOR UPDATE`); err != nil {
		t.Fatal(err)
	}

	before := failureCount(t, rule.Name, failureReasonTimeout, "allow")
	started := time.Now()
	allowed, _, _ := store.Take(context.Background(), "client")
	elapsed := time.Since(started)
	if !allowed {
		t.Fatal("fail-open must allow a timed-out decision")
	}
	if elapsed > time.Second {
		t.Fatalf("timed-out decision took %s, want about 50ms", elapsed)
	}
	if got := failureCount(t, rule.Name, failureReasonTimeout, "allow") - before; got != 1 {
		t.Fatalf("timeout failures recorded = %v, want 1", got)
	}
}

func TestRateLimitBucketTableIsUnlogged(t *testing.T) {
	db := openBucketTable(t)
	var persistence string
	if err := db.QueryRow(
		`SELECT relpersistence::text FROM pg_class WHERE oid = 'rate_limit_buckets'::regclass`,
	).Scan(&persistence); err != nil {
		t.Fatal(err)
	}
	if persistence != "u" {
		t.Fatalf("rate_limit_buckets persistence = %q, want UNLOGGED (u)", persistence)
	}
}

func BenchmarkPostgresStoreTake(b *testing.B) {
	db := openBucketTable(b)
	store := newTestPostgresStores(b, db).New(Rule{Name: "bench", Max: 1 << 30, Window: time.Hour})
	ctx := context.Background()

	b.Run("distinct_keys", func(b *testing.B) {
		var sequence atomic.Int64
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				store.Take(ctx, "client-"+strconv.FormatInt(sequence.Add(1)%10_000, 10))
			}
		})
	})
	b.Run("hot_key", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				store.Take(ctx, "hot")
			}
		})
	})
}

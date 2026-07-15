package ratelimit

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

func TestMemoryStoreTakeEnforcesQuotaAndExactWindowBoundary(t *testing.T) {
	clock := &manualClock{now: time.Unix(1_700_000_000, 0)}
	store := NewMemoryStore(2, time.Minute, withClock(clock.Now))
	ctx := context.Background()

	allowed, remaining, firstReset := store.Take(ctx, "client")
	if !allowed || remaining != 1 {
		t.Fatalf("first take = (%v, %d), want (true, 1)", allowed, remaining)
	}

	allowed, remaining, secondReset := store.Take(ctx, "client")
	if !allowed || remaining != 0 {
		t.Fatalf("second take = (%v, %d), want (true, 0)", allowed, remaining)
	}
	if !secondReset.Equal(firstReset) {
		t.Fatalf("second reset = %s, want %s", secondReset, firstReset)
	}

	allowed, remaining, deniedReset := store.Take(ctx, "client")
	if allowed || remaining != 0 || !deniedReset.Equal(firstReset) {
		t.Fatalf("denied take = (%v, %d, %s), want (false, 0, %s)", allowed, remaining, deniedReset, firstReset)
	}

	clock.Advance(time.Minute)
	allowed, remaining, nextReset := store.Take(ctx, "client")
	if !allowed || remaining != 1 {
		t.Fatalf("boundary take = (%v, %d), want (true, 1)", allowed, remaining)
	}
	if !nextReset.Equal(firstReset.Add(time.Minute)) {
		t.Fatalf("next reset = %s, want %s", nextReset, firstReset.Add(time.Minute))
	}
}

func TestMemoryStoreNormalizesInvalidDirectPolicyToDefaults(t *testing.T) {
	clock := &manualClock{now: time.Unix(1_700_000_000, 0)}
	store := NewMemoryStore(0, 0, withClock(clock.Now))

	allowed, remaining, resetAt := store.Take(context.Background(), "client")
	if !allowed || remaining != DefaultConfig().Max-1 {
		t.Fatalf("first take = (%v, %d), want (true, %d)", allowed, remaining, DefaultConfig().Max-1)
	}
	if want := clock.Now().Add(DefaultConfig().Duration); !resetAt.Equal(want) {
		t.Fatalf("reset = %s, want %s", resetAt, want)
	}
}

func TestMemoryStoreTakeIsAtomicUnderConcurrency(t *testing.T) {
	const (
		quota    = 64
		requests = 256
	)
	store := NewMemoryStore(quota, time.Minute)
	ctx := context.Background()

	var allowed atomic.Int64
	var group sync.WaitGroup
	group.Add(requests)
	for range requests {
		go func() {
			defer group.Done()
			if ok, _, _ := store.Take(ctx, "shared-client"); ok {
				allowed.Add(1)
			}
		}()
	}
	group.Wait()

	if got := allowed.Load(); got != quota {
		t.Fatalf("allowed requests = %d, want %d", got, quota)
	}
}

func TestMemoryStoreBoundsBucketsAndEvictsLeastRecentlyUsed(t *testing.T) {
	clock := &manualClock{now: time.Unix(1_700_000_000, 0)}
	store := NewMemoryStore(1, time.Hour, WithMaxBuckets(2), withClock(clock.Now))
	ctx := context.Background()

	store.Take(ctx, "alpha")
	store.Take(ctx, "beta")
	store.Take(ctx, "alpha") // A denied request still makes alpha the hottest bucket.
	store.Take(ctx, "gamma")

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.entries) != 2 {
		t.Fatalf("bucket count = %d, want 2", len(store.entries))
	}
	if _, exists := store.entries["alpha"]; !exists {
		t.Fatal("recently used alpha bucket was evicted")
	}
	if _, exists := store.entries["beta"]; exists {
		t.Fatal("least recently used beta bucket was retained")
	}
	if _, exists := store.entries["gamma"]; !exists {
		t.Fatal("new gamma bucket is missing")
	}
}

func TestMemoryStoreOpportunisticallyRemovesExpiredBuckets(t *testing.T) {
	clock := &manualClock{now: time.Unix(1_700_000_000, 0)}
	store := NewMemoryStore(1, 30*time.Second, WithMaxBuckets(3), withClock(clock.Now))
	ctx := context.Background()

	store.Take(ctx, "expired")
	clock.Advance(30 * time.Second)
	store.Take(ctx, "current")

	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.entries["expired"]; exists {
		t.Fatal("expired bucket was not removed during request-path cleanup")
	}
	if len(store.entries) != 1 {
		t.Fatalf("bucket count = %d, want 1", len(store.entries))
	}
}

func BenchmarkMemoryStoreTake(b *testing.B) {
	ctx := context.Background()

	b.Run("hot_allowed", func(b *testing.B) {
		store := NewMemoryStore(b.N+1, time.Hour)
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			store.Take(ctx, "client")
		}
	})

	b.Run("hot_denied", func(b *testing.B) {
		store := NewMemoryStore(1, time.Hour)
		store.Take(ctx, "client")
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			store.Take(ctx, "client")
		}
	})

	b.Run("bounded_identity_churn", func(b *testing.B) {
		const bucketCapacity = 1_024
		keys := make([]string, bucketCapacity*2)
		for index := range keys {
			keys[index] = "client-" + strconv.Itoa(index)
		}
		store := NewMemoryStore(1, time.Hour, WithMaxBuckets(bucketCapacity))
		b.ReportAllocs()
		b.ResetTimer()
		for index := range b.N {
			store.Take(ctx, keys[index%len(keys)])
		}
	})
}

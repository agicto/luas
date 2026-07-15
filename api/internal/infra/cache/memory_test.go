package cache

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type cacheTestClock struct {
	now time.Time
}

func (c *cacheTestClock) Now() time.Time {
	return c.now
}

func (c *cacheTestClock) Advance(duration time.Duration) {
	c.now = c.now.Add(duration)
}

func TestMemoryStoreExpiresAtExactTTLBoundary(t *testing.T) {
	clock := &cacheTestClock{now: time.Unix(1_700_000_000, 0)}
	store, constructionErr := newMemoryStore(MemoryConfig{}, clock.Now)
	if constructionErr != nil {
		t.Fatal(constructionErr)
	}
	ctx := context.Background()
	if err := store.Set(ctx, "key", []byte("value"), time.Minute); err != nil {
		t.Fatal(err)
	}

	clock.Advance(time.Minute)
	if _, err := store.Get(ctx, "key"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected exact-boundary expiry, got %v", err)
	}
	added, err := store.Add(ctx, "key", []byte("replacement"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected Add to replace an expired value")
	}
}

func TestMemoryStoreBoundsPayloadBytesAndReportsStats(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{
		MaxEntries:   4,
		MaxBytes:     12,
		MaxItemBytes: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Set(ctx, "a", []byte("12345"), time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "b", []byte("12345"), time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "c", []byte("12345"), time.Hour); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(ctx, "a"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected byte-pressure eviction, got %v", err)
	}
	stats := store.Stats()
	if stats.Entries != 2 || stats.PayloadBytes != 12 || stats.MaxBytes != 12 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if err := store.Set(ctx, "oversized", []byte("123456789"), time.Hour); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected item-size rejection, got %v", err)
	}
}

func TestMemoryStoreValidationDoesNotReplaceLiveValue(t *testing.T) {
	store, constructionErr := NewMemoryStore(MemoryConfig{})
	if constructionErr != nil {
		t.Fatal(constructionErr)
	}
	ctx := context.Background()
	if err := store.SetForever(ctx, "key", []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "key", []byte("replacement"), 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected invalid TTL, got %v", err)
	}
	value, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != "original" {
		t.Fatalf("invalid write replaced live value: %q", value)
	}
}

func TestMemoryStoreRejectsInvalidCapacityAndOperations(t *testing.T) {
	for name, config := range map[string]MemoryConfig{
		"entries": {MaxEntries: -1},
		"bytes":   {MaxBytes: -1},
		"item":    {MaxItemBytes: -1},
		"item over total": {
			MaxBytes:     10,
			MaxItemBytes: 11,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewMemoryStore(config); !errors.Is(err, ErrInvalidCapacity) {
				t.Fatalf("expected invalid capacity, got %v", err)
			}
		})
	}

	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Set(canceled, "key", []byte("value"), time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if _, err := store.Get(context.Background(), "key"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("canceled write mutated cache: %v", err)
	}
	if err := store.Set(context.Background(), "", []byte("value"), time.Hour); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected invalid key, got %v", err)
	}
	if err := store.Set(
		context.Background(),
		strings.Repeat("x", MaxKeyBytes+1),
		[]byte("value"),
		time.Hour,
	); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected oversized key rejection, got %v", err)
	}
}

func TestMemoryStoreClearResetsPayloadAccounting(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.SetForever(ctx, "key", []byte("value")); err != nil {
		t.Fatal(err)
	}
	if err := store.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if stats := store.Stats(); stats.Entries != 0 || stats.PayloadBytes != 0 {
		t.Fatalf("clear retained state: %+v", stats)
	}
}

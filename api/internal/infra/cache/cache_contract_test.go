package cache

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryStoreIsBoundedAndUsesLRUEviction(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{MaxEntries: 2})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	for key, value := range map[string]string{"a": "one", "b": "two"} {
		if err := store.Set(ctx, key, []byte(value), time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "c", []byte("three"), time.Hour); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(ctx, "b"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected least-recently-used key to be evicted, got %v", err)
	}
	if got := store.Len(); got != 2 {
		t.Fatalf("expected capacity 2, got %d entries", got)
	}
}

func TestMemoryStoreAddAndTakeAreAtomic(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	const workers = 128
	var added atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := store.Add(ctx, "winner", []byte("value"), time.Hour)
			if err != nil {
				t.Errorf("Add failed: %v", err)
				return
			}
			if ok {
				added.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := added.Load(); got != 1 {
		t.Fatalf("expected exactly one successful Add, got %d", got)
	}

	var taken atomic.Int32
	start = make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.Take(ctx, "winner")
			switch {
			case err == nil:
				taken.Add(1)
			case errors.Is(err, ErrCacheMiss):
			default:
				t.Errorf("Take failed: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := taken.Load(); got != 1 {
		t.Fatalf("expected exactly one successful Take, got %d", got)
	}
}

func TestMemoryStoreOwnsStoredAndReturnedBytes(t *testing.T) {
	store, constructionErr := NewMemoryStore(MemoryConfig{})
	if constructionErr != nil {
		t.Fatal(constructionErr)
	}
	ctx := context.Background()
	input := []byte("value")
	if err := store.Set(ctx, "key", input, time.Hour); err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'

	first, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	first[1] = 'X'
	second, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second, []byte("value")) {
		t.Fatalf("cache bytes alias caller memory: %q", second)
	}
}

func TestLoaderWaiterCanCancelWithoutWaitingForSharedLoad(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	loader, err := NewLoader(store)
	if err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	leaderDone := make(chan error, 1)
	go func() {
		_, loadErr := loader.Remember(
			context.Background(),
			"shared",
			time.Hour,
			func(context.Context) ([]byte, error) {
				close(started)
				<-release
				return []byte("loaded"), nil
			},
		)
		leaderDone <- loadErr
	}()
	<-started

	waiterCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	before := time.Now()
	var waiterLoadCalled atomic.Bool
	_, err = loader.Remember(waiterCtx, "shared", time.Hour, func(context.Context) ([]byte, error) {
		waiterLoadCalled.Store(true)
		return nil, errors.New("waiter unexpectedly started a second load")
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected waiter deadline, got %v", err)
	}
	if elapsed := time.Since(before); elapsed > 250*time.Millisecond {
		t.Fatalf("waiter ignored cancellation for %s", elapsed)
	}
	if waiterLoadCalled.Load() {
		t.Fatal("waiter did not join the existing shared load")
	}

	close(release)
	if err := <-leaderDone; err != nil {
		t.Fatal(err)
	}
}

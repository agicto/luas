package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoaderCollapsesConcurrentMissesAndCachesResult(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	loader, err := NewLoader(store)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 64
	var loadCount atomic.Int32
	start := make(chan struct{})
	release := make(chan struct{})
	results := make([][]byte, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errs[index] = loader.Remember(
				context.Background(),
				"shared",
				time.Hour,
				func(context.Context) ([]byte, error) {
					loadCount.Add(1)
					<-release
					return []byte("value"), nil
				},
			)
		}(i)
	}
	close(start)
	deadline := time.Now().Add(time.Second)
	for loadCount.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()

	if got := loadCount.Load(); got != 1 {
		t.Fatalf("expected one shared load, got %d", got)
	}
	for index := range results {
		if errs[index] != nil || string(results[index]) != "value" {
			t.Fatalf("worker %d = (%q, %v)", index, results[index], errs[index])
		}
	}
	results[0][0] = 'X'
	if string(results[1]) != "value" {
		t.Fatal("shared callers received aliased byte slices")
	}

	var cachedLoadCalled atomic.Bool
	_, err = loader.Remember(context.Background(), "shared", time.Hour, func(context.Context) ([]byte, error) {
		cachedLoadCalled.Store(true)
		return nil, errors.New("cached load callback ran")
	})
	if err != nil {
		t.Fatal(err)
	}
	if cachedLoadCalled.Load() {
		t.Fatal("cached load callback must not run")
	}
}

func TestLoaderDoesNotCacheFailures(t *testing.T) {
	store, constructionErr := NewMemoryStore(MemoryConfig{})
	if constructionErr != nil {
		t.Fatal(constructionErr)
	}
	loader, loaderErr := NewLoader(store)
	if loaderErr != nil {
		t.Fatal(loaderErr)
	}
	wantErr := errors.New("load failed")
	if _, err := loader.Remember(
		context.Background(),
		"key",
		time.Hour,
		func(context.Context) ([]byte, error) { return nil, wantErr },
	); !errors.Is(err, wantErr) {
		t.Fatalf("expected load error, got %v", err)
	}

	value, err := loader.Remember(
		context.Background(),
		"key",
		time.Hour,
		func(context.Context) ([]byte, error) { return []byte("recovered"), nil },
	)
	if err != nil || string(value) != "recovered" {
		t.Fatalf("retry = (%q, %v)", value, err)
	}
}

func TestRememberJSONLoadsTypedValueOnce(t *testing.T) {
	type record struct {
		Count int `json:"count"`
	}
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	loader, err := NewLoader(store)
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	load := func(context.Context) (record, error) {
		calls++
		return record{Count: 42}, nil
	}
	for i := 0; i < 2; i++ {
		value, err := RememberJSON(context.Background(), loader, "record", time.Hour, load)
		if err != nil {
			t.Fatal(err)
		}
		if value.Count != 42 {
			t.Fatalf("unexpected typed value: %+v", value)
		}
	}
	if calls != 1 {
		t.Fatalf("expected one typed load, got %d", calls)
	}
}

type failingCacheStore struct {
	err error
}

func (s *failingCacheStore) Get(context.Context, string) ([]byte, error) {
	return nil, s.err
}

func (s *failingCacheStore) Set(context.Context, string, []byte, time.Duration) error {
	return s.err
}

func (s *failingCacheStore) SetForever(context.Context, string, []byte) error {
	return s.err
}

func (s *failingCacheStore) Add(context.Context, string, []byte, time.Duration) (bool, error) {
	return false, s.err
}

func (s *failingCacheStore) Take(context.Context, string) ([]byte, error) {
	return nil, s.err
}

func (s *failingCacheStore) Delete(context.Context, string) error {
	return s.err
}

func TestLoaderPropagatesStoreFailureWithoutCallingSource(t *testing.T) {
	wantErr := errors.New("cache backend unavailable")
	loader, err := NewLoader(&failingCacheStore{err: wantErr})
	if err != nil {
		t.Fatal(err)
	}
	var sourceCalled atomic.Bool
	_, err = loader.Remember(context.Background(), "key", time.Hour, func(context.Context) ([]byte, error) {
		sourceCalled.Store(true)
		return []byte("value"), nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected backend error, got %v", err)
	}
	if sourceCalled.Load() {
		t.Fatal("source ran after a cache backend failure")
	}

	var nilLoader *Loader
	if _, err := nilLoader.Remember(context.Background(), "key", time.Hour, nil); !errors.Is(err, ErrNilDependency) {
		t.Fatalf("expected nil-loader error, got %v", err)
	}
}

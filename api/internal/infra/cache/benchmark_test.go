package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkMemoryGet(b *testing.B) {
	ctx := context.Background()
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		b.Fatal(err)
	}
	if err := store.Set(ctx, "key", []byte("value"), time.Hour); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Get(ctx, "key"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryGetParallel(b *testing.B) {
	ctx := context.Background()
	store, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		b.Fatal(err)
	}
	if err := store.Set(ctx, "key", []byte("value"), time.Hour); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := store.Get(ctx, "key"); err != nil {
				b.Error(err)
			}
		}
	})
}

func BenchmarkMemoryBoundedChurn(b *testing.B) {
	ctx := context.Background()
	store, err := NewMemoryStore(MemoryConfig{MaxEntries: 10_000})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		if err := store.Set(ctx, key, []byte("value"), time.Hour); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(store.Len()), "retained-entries")
}

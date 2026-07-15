package cache

import (
	"context"
	"errors"
	"reflect"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/zgiai/luas/api/internal/infra/metrics"
)

// Loader collapses concurrent misses for one explicitly owned Store. It has no
// global registry and creates goroutines only while a shared load is in flight.
type Loader struct {
	store   Store
	flights singleflight.Group
	label   string
}

// NewLoader creates a cache-aside loader scoped to one Store instance.
func NewLoader(store Store) (*Loader, error) {
	if isNilStore(store) {
		return nil, ErrNilDependency
	}
	return &Loader{store: store, label: metricLabelForStore(store)}, nil
}

// Remember returns a cached value or shares one load among concurrent callers.
// A canceled caller stops waiting; the in-flight load follows the context of
// the caller that started it.
func (l *Loader) Remember(
	ctx context.Context,
	key string,
	ttl time.Duration,
	load func(context.Context) ([]byte, error),
) ([]byte, error) {
	if l == nil || isNilStore(l.store) {
		return nil, ErrNilDependency
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := validateTTL(ttl); err != nil {
		return nil, err
	}
	if load == nil {
		return nil, ErrNilDependency
	}

	if value, err := l.store.Get(ctx, key); err == nil {
		metrics.RecordCacheHit(l.label)
		return value, nil
	} else if !errors.Is(err, ErrCacheMiss) {
		return nil, err
	}
	metrics.RecordCacheMiss(l.label)

	result := l.flights.DoChan(key, func() (interface{}, error) {
		if value, err := l.store.Get(ctx, key); err == nil {
			metrics.RecordCacheHit(l.label)
			return value, nil
		} else if !errors.Is(err, ErrCacheMiss) {
			return nil, err
		}

		value, err := load(ctx)
		if err != nil {
			return nil, err
		}
		if err := l.store.Set(ctx, key, value, ttl); err != nil {
			return nil, err
		}
		return cloneBytes(value), nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case completed := <-result:
		if completed.Err != nil {
			return nil, completed.Err
		}
		value, ok := completed.Val.([]byte)
		if !ok {
			return nil, errors.New("cache: shared load returned an invalid value")
		}
		return cloneBytes(value), nil
	}
}

func metricLabelForStore(store Store) string {
	switch store.(type) {
	case *MemoryStore:
		return "memory"
	case *RedisStore:
		return "redis"
	default:
		return "custom"
	}
}

func isNilStore(store Store) bool {
	if store == nil {
		return true
	}
	value := reflect.ValueOf(store)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

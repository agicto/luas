package cache

import (
	"bytes"
	"context"
	"errors"
	"time"
)

const (
	// MaxKeyBytes bounds caller-controlled cache-key growth across adapters.
	MaxKeyBytes = 1_024
	// DefaultMaxItemBytes bounds one cached payload unless an adapter is configured otherwise.
	DefaultMaxItemBytes = 1 << 20
)

var (
	// ErrCacheMiss reports that a key has no live value.
	ErrCacheMiss = errors.New("cache: key not found")
	// ErrInvalidKey reports an empty or excessively large cache key.
	ErrInvalidKey = errors.New("cache: key must contain 1 to 1024 bytes")
	// ErrInvalidTTL reports a non-positive TTL on an expiring operation.
	ErrInvalidTTL = errors.New("cache: TTL must be greater than zero")
	// ErrValueTooLarge reports a payload that exceeds the adapter's configured item limit.
	ErrValueTooLarge = errors.New("cache: value exceeds configured item limit")
	// ErrInvalidCapacity reports an invalid bounded-memory configuration.
	ErrInvalidCapacity = errors.New("cache: capacity must be greater than zero")
	// ErrInvalidNamespace reports an unsafe or ambiguous Redis namespace.
	ErrInvalidNamespace = errors.New("cache: Redis namespace must be non-empty and end with ':'")
	// ErrNilDependency reports a missing context, store, client, or load function.
	ErrNilDependency = errors.New("cache: required dependency is nil")
)

// Store is a driver-neutral cache seam. Values are bytes so switching adapters
// cannot silently change Go types. Set and Add require an explicit positive TTL;
// callers must opt into non-expiring data through SetForever.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	SetForever(ctx context.Context, key string, value []byte) error
	Add(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	Take(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrNilDependency
	}
	return ctx.Err()
}

func validateKey(key string) error {
	if len(key) == 0 || len(key) > MaxKeyBytes {
		return ErrInvalidKey
	}
	return nil
}

func validateTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	return nil
}

func validateItemSize(value []byte, maxItemBytes int) error {
	if len(value) > maxItemBytes {
		return ErrValueTooLarge
	}
	return nil
}

func cloneBytes(value []byte) []byte {
	return bytes.Clone(value)
}

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetJSON decodes one driver-neutral byte value into T.
func GetJSON[T any](ctx context.Context, store Store, key string) (T, error) {
	var result T
	if isNilStore(store) {
		return result, ErrNilDependency
	}
	value, err := store.Get(ctx, key)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(value, &result); err != nil {
		return result, fmt.Errorf("cache: decode JSON: %w", err)
	}
	return result, nil
}

// SetJSON encodes value once and stores it with a positive TTL.
func SetJSON[T any](ctx context.Context, store Store, key string, value T, ttl time.Duration) error {
	if isNilStore(store) {
		return ErrNilDependency
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: encode JSON: %w", err)
	}
	return store.Set(ctx, key, encoded, ttl)
}

// SetJSONForever explicitly stores a non-expiring encoded value.
func SetJSONForever[T any](ctx context.Context, store Store, key string, value T) error {
	if isNilStore(store) {
		return ErrNilDependency
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: encode JSON: %w", err)
	}
	return store.SetForever(ctx, key, encoded)
}

// RememberJSON collapses concurrent typed loads while keeping serialization
// identical across memory and Redis adapters.
func RememberJSON[T any](
	ctx context.Context,
	loader *Loader,
	key string,
	ttl time.Duration,
	load func(context.Context) (T, error),
) (T, error) {
	var result T
	if loader == nil || load == nil {
		return result, ErrNilDependency
	}
	encoded, err := loader.Remember(ctx, key, ttl, func(loadCtx context.Context) ([]byte, error) {
		value, loadErr := load(loadCtx)
		if loadErr != nil {
			return nil, loadErr
		}
		payload, encodeErr := json.Marshal(value)
		if encodeErr != nil {
			return nil, fmt.Errorf("cache: encode JSON: %w", encodeErr)
		}
		return payload, nil
	})
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(encoded, &result); err != nil {
		return result, fmt.Errorf("cache: decode JSON: %w", err)
	}
	return result, nil
}

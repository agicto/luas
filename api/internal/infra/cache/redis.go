package cache

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const maxRedisNamespaceBytes = 256

// RedisClient is the narrow borrowed-client seam required by RedisStore.
// The composition root owns connection configuration, readiness, and Close.
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	GetDel(ctx context.Context, key string) *redis.StringCmd
	Unlink(ctx context.Context, keys ...string) *redis.IntCmd
}

// RedisConfig controls namespacing and one-item payload limits. Namespace is
// required so a shared deployment cannot accidentally collide with other apps.
type RedisConfig struct {
	Namespace    string
	MaxItemBytes int
}

// RedisStore is a shared cache adapter for Redis 6.2 or later. It borrows its
// client and therefore deliberately exposes no Close method.
type RedisStore struct {
	client       RedisClient
	namespace    string
	maxItemBytes int
}

// NewRedisStore creates a namespaced shared cache adapter.
func NewRedisStore(client RedisClient, config RedisConfig) (*RedisStore, error) {
	if isNilRedisClient(client) {
		return nil, ErrNilDependency
	}
	if !validRedisNamespace(config.Namespace) {
		return nil, ErrInvalidNamespace
	}
	if config.MaxItemBytes == 0 {
		config.MaxItemBytes = DefaultMaxItemBytes
	}
	if config.MaxItemBytes < 0 {
		return nil, ErrInvalidCapacity
	}
	return &RedisStore{
		client:       client,
		namespace:    config.Namespace,
		maxItemBytes: config.MaxItemBytes,
	}, nil
}

func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := s.validateOperation(ctx, key); err != nil {
		return nil, err
	}
	return redisBytes(s.client.Get(ctx, s.namespace+key))
}

func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateTTL(ttl); err != nil {
		return err
	}
	return s.set(ctx, key, value, ttl)
}

func (s *RedisStore) SetForever(ctx context.Context, key string, value []byte) error {
	return s.set(ctx, key, value, 0)
}

func (s *RedisStore) set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := s.validateOperation(ctx, key); err != nil {
		return err
	}
	if err := validateItemSize(value, s.maxItemBytes); err != nil {
		return err
	}
	return s.client.Set(ctx, s.namespace+key, cloneBytes(value), ttl).Err()
}

func (s *RedisStore) Add(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) (bool, error) {
	if err := s.validateOperation(ctx, key); err != nil {
		return false, err
	}
	if err := validateTTL(ttl); err != nil {
		return false, err
	}
	if err := validateItemSize(value, s.maxItemBytes); err != nil {
		return false, err
	}
	return s.client.SetNX(ctx, s.namespace+key, cloneBytes(value), ttl).Result()
}

func (s *RedisStore) Take(ctx context.Context, key string) ([]byte, error) {
	if err := s.validateOperation(ctx, key); err != nil {
		return nil, err
	}
	return redisBytes(s.client.GetDel(ctx, s.namespace+key))
}

func (s *RedisStore) Delete(ctx context.Context, key string) error {
	if err := s.validateOperation(ctx, key); err != nil {
		return err
	}
	return s.client.Unlink(ctx, s.namespace+key).Err()
}

func (s *RedisStore) validateOperation(ctx context.Context, key string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	return validateKey(key)
}

func redisBytes(command *redis.StringCmd) ([]byte, error) {
	value, err := command.Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	return cloneBytes(value), nil
}

func validRedisNamespace(namespace string) bool {
	if namespace == "" || len(namespace) > maxRedisNamespaceBytes ||
		strings.TrimSpace(namespace) != namespace || !strings.HasSuffix(namespace, ":") {
		return false
	}
	for _, character := range namespace {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func isNilRedisClient(client RedisClient) bool {
	if client == nil {
		return true
	}
	value := reflect.ValueOf(client)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

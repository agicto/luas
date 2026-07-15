package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var _ RedisClient = (*redis.Client)(nil)

type fakeRedisEntry struct {
	value     string
	expiresAt time.Time
}

type fakeRedisClient struct {
	mu sync.Mutex

	entries     map[string]fakeRedisEntry
	now         func() time.Time
	setNXCalls  int
	getDelCalls int
	unlinkCalls int
	failure     error
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{entries: make(map[string]fakeRedisEntry), now: time.Now}
}

func (c *fakeRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.operationError(ctx); err != nil {
		return redis.NewStringResult("", err)
	}
	entry, ok := c.liveEntryLocked(key)
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(entry.value, nil)
}

func (c *fakeRedisClient) Set(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) *redis.StatusCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.operationError(ctx); err != nil {
		return redis.NewStatusResult("", err)
	}
	encoded, err := fakeRedisValue(value)
	if err != nil {
		return redis.NewStatusResult("", err)
	}
	c.entries[key] = fakeRedisEntry{value: encoded, expiresAt: fakeExpiry(c.now(), expiration)}
	return redis.NewStatusResult("OK", nil)
}

func (c *fakeRedisClient) SetNX(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) *redis.BoolCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setNXCalls++
	if err := c.operationError(ctx); err != nil {
		return redis.NewBoolResult(false, err)
	}
	if _, ok := c.liveEntryLocked(key); ok {
		return redis.NewBoolResult(false, nil)
	}
	encoded, err := fakeRedisValue(value)
	if err != nil {
		return redis.NewBoolResult(false, err)
	}
	c.entries[key] = fakeRedisEntry{value: encoded, expiresAt: fakeExpiry(c.now(), expiration)}
	return redis.NewBoolResult(true, nil)
}

func (c *fakeRedisClient) GetDel(ctx context.Context, key string) *redis.StringCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getDelCalls++
	if err := c.operationError(ctx); err != nil {
		return redis.NewStringResult("", err)
	}
	entry, ok := c.liveEntryLocked(key)
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	delete(c.entries, key)
	return redis.NewStringResult(entry.value, nil)
}

func (c *fakeRedisClient) Unlink(ctx context.Context, keys ...string) *redis.IntCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unlinkCalls++
	if err := c.operationError(ctx); err != nil {
		return redis.NewIntResult(0, err)
	}
	var removed int64
	for _, key := range keys {
		if _, ok := c.entries[key]; ok {
			delete(c.entries, key)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

func (c *fakeRedisClient) operationError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.failure
}

func (c *fakeRedisClient) liveEntryLocked(key string) (fakeRedisEntry, bool) {
	entry, ok := c.entries[key]
	if !ok {
		return fakeRedisEntry{}, false
	}
	if !entry.expiresAt.IsZero() && !c.now().Before(entry.expiresAt) {
		delete(c.entries, key)
		return fakeRedisEntry{}, false
	}
	return entry, true
}

func fakeRedisValue(value interface{}) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	default:
		return "", fmt.Errorf("unsupported fake Redis value %T", value)
	}
}

func fakeExpiry(now time.Time, ttl time.Duration) time.Time {
	if ttl == 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}

func TestStoreContractMatchesMemoryAndRedis(t *testing.T) {
	factories := map[string]func(t *testing.T) Store{
		"memory": func(t *testing.T) Store {
			store, err := NewMemoryStore(MemoryConfig{})
			if err != nil {
				t.Fatal(err)
			}
			return store
		},
		"redis": func(t *testing.T) Store {
			store, err := NewRedisStore(newFakeRedisClient(), RedisConfig{Namespace: "luas:test:cache:"})
			if err != nil {
				t.Fatal(err)
			}
			return store
		},
	}

	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			assertStoreContract(t, factory(t))
		})
	}
}

func assertStoreContract(t *testing.T, store Store) {
	t.Helper()
	ctx := context.Background()
	input := []byte(`{"count":42}`)
	if err := store.Set(ctx, "record", input, time.Hour); err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'
	first, err := store.Get(ctx, "record")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != `{"count":42}` {
		t.Fatalf("adapter changed byte value: %q", first)
	}
	first[0] = 'X'
	second, err := store.Get(ctx, "record")
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != `{"count":42}` {
		t.Fatalf("adapter returned aliased bytes: %q", second)
	}

	added, err := store.Add(ctx, "record", []byte("replacement"), time.Hour)
	if err != nil || added {
		t.Fatalf("existing Add = (%v, %v), want (false, nil)", added, err)
	}
	taken, err := store.Take(ctx, "record")
	if err != nil || string(taken) != `{"count":42}` {
		t.Fatalf("Take = (%q, %v)", taken, err)
	}
	if _, missErr := store.Take(ctx, "record"); !errors.Is(missErr, ErrCacheMiss) {
		t.Fatalf("second Take must miss, got %v", missErr)
	}
	added, err = store.Add(ctx, "record", []byte("replacement"), time.Hour)
	if err != nil || !added {
		t.Fatalf("missing Add = (%v, %v), want (true, nil)", added, err)
	}
	if err := store.Delete(ctx, "record"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "record"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Delete left value: %v", err)
	}
	if err := store.SetForever(ctx, "forever", []byte("value")); err != nil {
		t.Fatal(err)
	}
	if value, err := store.Get(ctx, "forever"); err != nil || string(value) != "value" {
		t.Fatalf("SetForever/Get = (%q, %v)", value, err)
	}
}

func TestRedisStoreUsesNamespacedAtomicCommandsAndBorrowsClient(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewRedisStore(client, RedisConfig{Namespace: "luas:production:cache:"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	added, err := store.Add(ctx, "key", []byte("value"), time.Hour)
	if err != nil || !added {
		t.Fatalf("Add = (%v, %v)", added, err)
	}
	if _, err := store.Take(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "delete", []byte("value"), time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "delete"); err != nil {
		t.Fatal(err)
	}

	client.mu.Lock()
	setNXCalls, getDelCalls, unlinkCalls := client.setNXCalls, client.getDelCalls, client.unlinkCalls
	_, leakedUnprefixedKey := client.entries["key"]
	client.mu.Unlock()
	if setNXCalls != 1 || getDelCalls != 1 || unlinkCalls != 1 {
		t.Fatalf("unexpected Redis command counts: SET NX=%d GETDEL=%d UNLINK=%d", setNXCalls, getDelCalls, unlinkCalls)
	}
	if leakedUnprefixedKey {
		t.Fatal("Redis adapter wrote an unnamespaced key")
	}
	if _, ownsClose := interface{}(store).(interface{ Close() error }); ownsClose {
		t.Fatal("RedisStore must not close its borrowed client")
	}
}

func TestRedisStoreValidationAndBackendErrors(t *testing.T) {
	var typedNil *fakeRedisClient
	for name, testCase := range map[string]struct {
		client   RedisClient
		config   RedisConfig
		expected error
	}{
		"nil client":       {client: nil, config: RedisConfig{Namespace: "luas:test:"}, expected: ErrNilDependency},
		"typed nil client": {client: typedNil, config: RedisConfig{Namespace: "luas:test:"}, expected: ErrNilDependency},
		"empty namespace":  {client: newFakeRedisClient(), config: RedisConfig{}, expected: ErrInvalidNamespace},
		"missing separator": {
			client: newFakeRedisClient(), config: RedisConfig{Namespace: "luas:test"}, expected: ErrInvalidNamespace,
		},
		"negative item limit": {
			client: newFakeRedisClient(), config: RedisConfig{Namespace: "luas:test:", MaxItemBytes: -1}, expected: ErrInvalidCapacity,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewRedisStore(testCase.client, testCase.config)
			if !errors.Is(err, testCase.expected) {
				t.Fatalf("expected %v, got %v", testCase.expected, err)
			}
		})
	}

	backendErr := errors.New("Redis unavailable")
	client := newFakeRedisClient()
	client.failure = backendErr
	store, err := NewRedisStore(client, RedisConfig{Namespace: "luas:test:"})
	if err != nil {
		t.Fatal(err)
	}
	if _, backendGetErr := store.Get(context.Background(), "key"); !errors.Is(backendGetErr, backendErr) {
		t.Fatalf("expected backend error, got %v", backendGetErr)
	}

	limited, err := NewRedisStore(newFakeRedisClient(), RedisConfig{
		Namespace:    "luas:test:",
		MaxItemBytes: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := limited.Set(context.Background(), "key", []byte("12345"), time.Hour); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected item-size rejection, got %v", err)
	}
}

func TestRedisStoreRealServer(t *testing.T) {
	address := os.Getenv("LUAS_TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("LUAS_TEST_REDIS_ADDR is not set")
	}
	client := redis.NewClient(&redis.Options{Addr: address})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping test Redis: %v", err)
	}

	namespace := fmt.Sprintf("luas:cache-integration:%d:", time.Now().UnixNano())
	store, err := NewRedisStore(client, RedisConfig{Namespace: namespace})
	if err != nil {
		t.Fatal(err)
	}
	assertStoreContract(t, store)
	for _, key := range []string{"record", "forever"} {
		if err := client.Unlink(ctx, namespace+key).Err(); err != nil {
			t.Fatalf("cleanup Redis key: %v", err)
		}
	}
}

func TestRedisStoreAddIsAtomic(t *testing.T) {
	client := newFakeRedisClient()
	store, err := NewRedisStore(client, RedisConfig{Namespace: "luas:test:"})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 128
	var successes atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			added, addErr := store.Add(context.Background(), "key", []byte("value"), time.Hour)
			if addErr != nil {
				t.Errorf("Add: %v", addErr)
			} else if added {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("expected one Add winner, got %d", successes.Load())
	}
}

func TestJSONHelpersPreserveConcreteTypesAcrossStores(t *testing.T) {
	type record struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	want := record{Name: "cache", Count: 42}
	stores := map[string]Store{}
	memory, err := NewMemoryStore(MemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}
	stores["memory"] = memory
	redisStore, err := NewRedisStore(newFakeRedisClient(), RedisConfig{Namespace: "luas:test:"})
	if err != nil {
		t.Fatal(err)
	}
	stores["redis"] = redisStore

	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			if err := SetJSON(context.Background(), store, "record", want, time.Hour); err != nil {
				t.Fatal(err)
			}
			got, err := GetJSON[record](context.Background(), store, "record")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
		})
	}
}

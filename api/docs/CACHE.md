# Cache Capability

Luas ships an optional cache capability under `internal/infra/cache`. It is not assembled into the
default runtime, no starter depends on it, and there are no `REDIS_*` environment switches. A
downstream composition root must choose an adapter and inject the resulting `cache.Store` at the
business seam that owns keys, TTLs, invalidation, and outage behavior.

Cache data is a disposable copy of authoritative state. Do not use this capability as the authority
for authentication sessions, permissions, idempotency receipts, usage quotas, rate limits, durable
work, or business records. `Add` is not a distributed-lock API: it has no ownership token, renewal,
compare-and-delete release, or fencing semantics.

## Store Contract

Every adapter implements one byte-oriented contract:

| Operation | Semantics |
|---|---|
| `Get` | Return an owned byte slice or `ErrCacheMiss`. Backend failures remain errors. |
| `Set` | Replace a value with an explicit positive TTL. |
| `SetForever` | Deliberately opt into a non-expiring value; adapter capacity or server eviction still applies. |
| `Add` | Atomically set only when no live value exists; requires a positive TTL. |
| `Take` | Atomically return and remove one live value. |
| `Delete` | Remove one key; a missing key is not an error. |

Keys contain 1 to 1,024 bytes. Both built-in adapters copy values at their ownership boundary and
default to a 1 MiB item limit. `Set`, `Add`, and cache-aside loading reject non-positive TTLs instead
of interpreting zero as forever. `SetForever` makes that lifecycle decision visible in review.

The shared contract deliberately omits `Has`, counters, key enumeration, and store-wide `Flush`:

- `Has` hides backend errors and encourages time-of-check/time-of-use races;
- counter expiry and overflow belong to a purpose-built quota or limiter seam;
- enumeration can expose key material;
- flushing a shared Redis keyspace is too broad for application request paths.

Use `GetJSON`, `SetJSON`, `SetJSONForever`, and `RememberJSON` when JSON is the selected codec. The
codec stays above the byte contract, so memory and Redis return the same concrete Go type instead of
silently changing structs into `map[string]any` and integers into `float64`.

## Bounded Memory Adapter

`NewMemoryStore(cache.MemoryConfig{})` selects these defaults:

| Limit | Default |
|---|---:|
| Active entries | 10,000 |
| Retained key/value payload | 64 MiB |
| One value | 1 MiB |

The adapter is process-local, volatile, and least-recently-used. It owns no cleanup goroutine.
Reads remove an expired hit; writes evict least-recently-used entries in O(1) when either capacity
limit is reached. `Stats` performs the deliberate O(n) expiry sweep and reports only counts and
payload bytes, never keys. `MaxBytes` measures key and value payload bytes; Go map and entry metadata
remain additional bounded-per-entry overhead.

Each API replica has an independent memory cache. Do not claim cross-replica coherence, atomicity,
or invalidation when this adapter is selected.

## Explicit Redis Adapter

The Redis adapter requires Redis 6.2 or later because atomic `Take` uses `GETDEL`. `Add` uses
`SET ... NX`, and `Delete` uses `UNLINK` so value memory can be reclaimed asynchronously. Every
adapter requires an explicit namespace ending in `:` such as `acme:production:profile:`.

```go
client := redis.NewClient(&redis.Options{
    Addr: "redis.internal:6379",
})

store, err := cache.NewRedisStore(client, cache.RedisConfig{
    Namespace: "acme:production:profile:",
})
```

The adapter borrows the client. The downstream composition root owns TLS/authentication, command
timeouts, pool sizing, startup `PING`, readiness, telemetry, and exactly one `Close`. The adapter
does not create a client from strings, read process configuration, close a shared client, enumerate
keys, or fall back to process-local memory during an outage. Such fallback changes consistency and
must be an explicit product policy outside this capability.

Redis `maxmemory` and an eviction policy remain deployment responsibilities. Include app,
environment, and owning capability in the namespace. Use opaque IDs or keyed hashes rather than
emails, tokens, URLs, or other sensitive values in keys.

## Cache-Aside Loading

`NewLoader(store)` creates a loader scoped to one explicit store. `Remember` rechecks the cache
inside `golang.org/x/sync/singleflight`, shares one in-flight load per key, and lets each waiter stop
waiting when its context is canceled. The load itself follows the context of the caller that started
it. Loader and backend errors are returned; only successful values are cached.

The loader prevents a same-process stampede. It does not coordinate loaders in other replicas. Use
request coalescing in a shared system only after the product has defined stale-data policy, outage
behavior, and observability.

## Ownership Checklist

Before caching a business read, its owning starter or downstream feature must define:

1. authoritative source and acceptable staleness;
2. finite, privacy-safe key shape and tenant scope;
3. TTL and explicit invalidation events;
4. behavior when the cache is unavailable;
5. multi-replica coherence expectations;
6. metrics that distinguish hit, miss, load failure, and source latency.

## Verification

```bash
go test ./internal/infra/cache
go test -race ./internal/infra/cache -count=10
make benchmark-cache
LUAS_TEST_REDIS_ADDR=127.0.0.1:6379 go test ./internal/infra/cache -run '^TestRedisStoreRealServer$' -v
```

Architecture rationale is recorded in
[`adr/0012-cache-capability-boundary.md`](adr/0012-cache-capability-boundary.md).

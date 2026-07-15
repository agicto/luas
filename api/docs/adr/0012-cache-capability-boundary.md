# ADR 0012: Cache Capability Boundary

- Status: Accepted
- Date: 2026-07-16

## Context

The legacy cache surface exposed a global mutable manager, an implicitly created memory store, broad
package functions, and a second unused interface under `internal/infra/contracts`. Its `any` value
contract was not substitutable: memory returned the original Go value while Redis JSON-decoded the
same payload into different runtime types. `Add` and `Pull` were multi-command check/write and
get/delete sequences despite names that implied atomicity.

Every memory store also started a permanent cleanup goroutine, retained unbounded key cardinality,
and could panic when closed twice. The Redis constructor created a client without startup health or
lifecycle ownership, while `Close` could also close a client borrowed from another subsystem.
Prefix scanning exposed a broad flush operation. The custom singleflight implementation blocked on
an existing call before observing waiter cancellation.

No assembled starter used these surfaces, so preserving them would give downstream applications a
misleading production contract without retaining runtime compatibility value.

## Decision

1. Cache remains an optional technical capability, not a starter and not default runtime state.
2. `Store` carries bytes. Serialization is caller-owned, with typed JSON helpers supplied above the
   byte seam.
3. Expiring writes require positive TTLs. Non-expiring writes use the explicit `SetForever` name.
4. `Add` and `Take` are atomic adapter operations. `Has`, generic counters, enumeration, and shared
   `Flush` are absent from the common contract.
5. Memory storage is bounded by entry count, retained key/value payload bytes, and item bytes. It
   uses intrusive LRU eviction, copies values, removes expiry opportunistically, and owns no
   background goroutine or close lifecycle.
6. Redis requires a namespaced, borrowed client and Redis 6.2 or later. It maps atomic operations to
   `SET NX` and `GETDEL`, deletes with `UNLINK`, propagates backend errors, and has no local fallback.
7. Cache-aside request coalescing is scoped to one `Loader` and uses the maintained
   `golang.org/x/sync/singleflight` implementation. Waiters can honor their own cancellation.
8. The duplicate cache contract, global manager, client-from-string constructor, and custom
   singleflight package are removed.
9. `Add` is not presented as distributed locking. A future lock capability must separately define
   ownership tokens, bounded leases, compare-and-delete release, renewal, fencing, and outage rules.

## Consequences

- Switching between memory and Redis no longer changes decoded Go types.
- Concurrent `Add` and `Take` have one winner rather than a check/use race.
- Memory use is bounded by explicit payload and cardinality ceilings and creates zero persistent
  goroutines per store.
- Reads pay for an ownership copy. This intentionally trades one small allocation for driver
  substitution and race-safe payload ownership.
- Redis connection security, pool settings, readiness, and shutdown stay visible at the composition
  root rather than hiding behind cache configuration strings.
- Downstream callers must inject a store and own cache policy. Existing code built against the old
  internal global helpers must migrate deliberately; Luas itself had no production callers.

## References

- [Redis `SET` command](https://redis.io/docs/latest/commands/set/)
- [Redis `GETDEL` command](https://redis.io/docs/latest/commands/getdel/)
- [Redis `UNLINK` command](https://redis.io/docs/latest/commands/unlink/)
- [Redis key eviction](https://redis.io/docs/latest/develop/reference/eviction/)
- [Redis Go client connection guidance](https://redis.io/docs/latest/develop/clients/go/connect/)
- [Go `singleflight` package](https://pkg.go.dev/golang.org/x/sync/singleflight)

# ADR 0015: Shared PostgreSQL Rate-Limit Store

- Status: Accepted
- Date: 2026-10-11
- Amends: [ADR 0011](0011-rate-limit-runtime-boundary.md) decisions 1 and 6

## Context

ADR 0011 kept every built-in limiter process-local, so N API replicas enforce each global and
authentication budget N times. Products that scale horizontally had to add a gateway/WAF limit or
write their own shared adapter before the authentication quotas meant what their configuration says.
Every FastBuild deployment already runs PostgreSQL; Redis is not part of the runtime.

ADR 0011 also listed what a shared adapter must specify: atomic algorithm, outage behavior,
readiness signal, key privacy, time source, and lifecycle. It prohibited silent fallback to
per-process buckets.

## Decision

1. `RATE_LIMIT_STORE=postgres` is an opt-in shared store for the global limiter and every
   authentication quota, including the login failure budget and its reset on success. `memory`
   remains the default.
2. **Algorithm.** One atomic `INSERT ... ON CONFLICT (key_hash) DO UPDATE ... RETURNING` per decision
   implements the same fixed window as the memory store. Hits saturate at `max + 1` so denied
   requests never grow the counter.
3. **Time source.** Window starts, ends, and expiry use the database clock (`statement_timestamp()`),
   so replica clock skew cannot split a window.
4. **Key privacy.** Rows store only `SHA-256(rule name, window, key)`; raw IPs, subjects, and tokens
   are never persisted.
5. **Table lifecycle.** A core migration creates the `UNLOGGED` `rate_limit_buckets` table. Losing
   counters after a crash or failover only resets current windows, and skipping WAL keeps the
   per-request write cheap. Expired rows are deleted in bounded, single-flight, short-lived cleanup
   passes and by `ratelimit:prune`; no long-lived goroutine exists.
6. **Outage behavior.** `RATE_LIMIT_STORE_TIMEOUT` (default 50 ms, at most 1 s) bounds each decision.
   On error or timeout, `RATE_LIMIT_STORE_FAILURE` applies: `local` (default) enforces the same rule
   against a bounded process-local bucket, `open` allows, and `closed` denies with the existing
   `429` + `COMMON.RATE_LIMITED` envelope and a one-second `Retry-After`.
7. The `local` policy is an explicit, configured, observable degradation, not the silent fallback
   ADR 0011 prohibits: every fallback increments `rate_limit_store_failures_total` with rule, reason,
   and decision labels, and a `ratelimit.store_unavailable` warning is logged at most once per
   10 seconds per process with a suppressed count. It is the default because it is never weaker than
   the memory store FastBuild already ships as its production default, and it never rejects healthy
   traffic merely because PostgreSQL was slow for one decision.
8. **Readiness.** The store adds no readiness check of its own; the existing database readiness check
   already covers its only dependency, and a degraded decision path must not remove replicas.

## Consequences

- Shared enforcement costs one PostgreSQL round trip and one pooled connection per rate-limited
  request. Hot keys serialize on one row lock.
- Unlike the memory store, the shared table has no cardinality ceiling: many source IPs add one small
  row each per window until cleanup removes it.
- The shared store does not absorb floods: every evaluated request has already reached the API and
  costs a database write. A gateway/WAF stays the volumetric and bot-protection layer.
- `closed` returns `429` rather than `503` during a store outage so clients keep one backoff path; the
  metric and warning distinguish an outage from real abuse.

## References

- [ADR 0011: Rate-Limit Runtime Boundary](0011-rate-limit-runtime-boundary.md)
- [PostgreSQL `INSERT ... ON CONFLICT`](https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT)
- [PostgreSQL unlogged tables](https://www.postgresql.org/docs/current/sql-createtable.html#SQL-CREATETABLE-UNLOGGED)

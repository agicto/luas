# ADR 0011: Rate-Limit Runtime Boundary

- Status: Accepted
- Date: 2026-07-15

## Context

Luas enables a global client-IP quota and endpoint-specific authentication quotas in production.
The authentication guard correctly uses independent source and normalized/hashed subject buckets,
but all assembled stores are process-local.

The repository also contained an unassembled Redis token-bucket implementation and inert
`REDIS_*` configuration. That implementation rounded every refill rate below one request per second
up to one request per second, returned approximate quota state, and silently fell back to independent
local buckets when Redis failed. For a policy such as three requests per hour, this could materially
weaken the configured limit. Configuration that no runtime consumes also violates the typed startup
snapshot's promise that documented settings have observable behavior.

## Decision

1. The built-in limiter remains a single-process fixed-window baseline.
2. Every store enforces decisions atomically through one `Take` operation.
3. Every store has a configurable active-bucket ceiling and evicts the least recently used bucket
   at capacity. Expired buckets are removed opportunistically without background goroutines.
4. Global and authentication cardinality use separate typed settings because authentication owns
   several independent endpoint/dimension stores.
5. Luas does not claim a built-in Redis rate-limit driver. The generic Redis cache adapter is a
   separate infrastructure surface and requires explicit downstream composition.
6. Multi-replica products must enforce shared policy at a gateway/WAF or deliberately assemble a
   shared limiter adapter. Such an adapter must specify its atomic algorithm, outage behavior,
   readiness signal, key privacy, time source, and client shutdown lifecycle. Silent fallback to
   per-process buckets is prohibited because it changes the security policy during dependency failure.

## Consequences

- Attacker-controlled IP or subject churn cannot grow one store without bound.
- Production no longer starts eight persistent cleanup goroutines for the default global and
  authentication rules.
- LRU eviction and fixed-window boundary bursts remain documented limitations of the local baseline.
- Removing inert Redis settings is a configuration cleanup, not removal of an active runtime feature.
- A future shared driver is a deliberate architecture change with integration and failure-mode tests,
  rather than a hidden switch behind generic Redis environment variables.

## References

- [Redis rate-limiter patterns](https://redis.io/docs/latest/develop/use-cases/rate-limiter/)
- [OWASP Bot Management and Anti-Automation](https://cheatsheetseries.owasp.org/cheatsheets/Bot_Management_and_Anti-Automation_Cheat_Sheet.html)
- [OWASP Credential Stuffing Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Credential_Stuffing_Prevention_Cheat_Sheet.html)

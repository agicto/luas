# ADR 0008: Usage Metering And Quota Boundary

## Status

Accepted

## Context

Luas already has user-owned API keys, AI/workflow capabilities, private assets, organizations, and
typed settings. Downstream API and AI products still repeat the same accounting work: deduplicating
usage facts, aggregating periods, deciding limits, exposing current utilization, and later adapting
those facts to billing providers.

HTTP rate limiting cannot own this behavior because it protects transport, commonly uses ephemeral
buckets, and does not represent billable or user-visible business usage. Settings cannot own it
because a usage counter has an event and concurrency lifecycle. Starting with Stripe, OpenMeter, or
another provider would leak payment/vendor vocabulary into core domain ownership and make local
development dependent on an external service.

## Decision

Add an optional organization-dependent `usage` starter with four explicit concepts:

1. code-owned finite usage metrics;
2. immutable idempotent receipts keyed by producer source and event ID;
3. non-negative UTC day/month counters;
4. optional subject hard quotas resolved from code defaults plus CAS operator overrides.

Trusted Go callers use separate record and consume seams. Record preserves facts and bounded signed
corrections even when usage is already over quota. Consume uses the server clock and performs the
receipt reservation, owner/counter lock, quota check, counter update, and receipt finalization in one
database transaction. There is no public or browser write endpoint.

Metric dimensions are exact code-owned finite enums. Quantities use the JavaScript safe integer
range so PostgreSQL, Go, JSON, and TypeScript agree exactly. Receipts retain only minimized business
metadata and can be pruned after 90 days while counters remain. Individual usage events do not fan
out into the audit starter; operator reconciliation, quota, and prune commands are audited.

The browser receives only bounded current summaries for the current user and owner/admin active
organization. Billing, plans, prices, entitlements, provider events, arbitrary tags, historical
analytics, and stream-processing infrastructure are deferred.

## Consequences

- New projects can enforce and display usage without choosing a billing provider.
- Retry safety and concurrent quota decisions have one local transactional authority.
- Safe defaults remain unlimited; product policy is introduced through reviewed catalog defaults or
  explicit subject overrides rather than hidden scaffold limits.
- Relational receipts are appropriate for starter-scale workloads, but high-volume products should
  replace the domain seams with a dedicated metering pipeline.
- Receipt pruning makes the idempotency horizon explicit; producers must not replay beyond it.
- Organization activation is required so one starter can model both individual and tenant usage with
  referential integrity and consistent browser context.

## Verification Impact

- catalog validation tests cover key/unit/period/dimension/limit bounds;
- repository and service tests cover duplicate replay, payload conflicts, correction underflow,
  late events, quota CAS, hard denial, and account cleanup;
- PostgreSQL Compose exercises concurrent idempotency and hard-quota consumption;
- browser tests cover strict read schemas, private adapters, role behavior, responsive rendering, and
  mock/production parity;
- root governance keeps API, Web, contracts, starter selection, migration, retention, privacy, and
  no-public-ingest rules aligned.

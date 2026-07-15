# Usage Contract

The optional `usage` starter owns durable business usage events, period counters, subject quota
overrides, and atomic limit decisions. It depends on the default `user` and `audit` starters and on
the optional `organization` starter.

Enable both deployable halves explicitly:

```env
OPTIONAL_STARTERS=organization,usage
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,usage
```

Usage is not request telemetry, an HTTP rate limiter, a feature entitlement, a plan, a price, an
invoice, or a payment-provider event. The starter has no public or browser ingestion endpoint.
Application services report trusted events through the Go domain seams; operators can use the Luas
CLI for reconciliation and quota administration.

## Vocabulary And Catalog

A **usage metric** is an immutable code-owned definition with:

- a dotted lowercase key, for example `api.requests`;
- a lowercase unit, for example `request` or `token`;
- one UTC calendar period: `day` or `month`;
- an optional non-negative default hard limit;
- zero or more code-owned dimensions, each with a finite enum allowlist.

The catalog is capped at 64 scoped definitions. A dimension key is code-owned, a definition has at
most 8 dimensions, and each dimension has at most 32 allowed values. Events must provide exactly
the dimensions declared by their metric. This prevents accidental PII and unbounded cardinality.
Runtime metric creation, arbitrary JSON, free-form tags, and customer identifiers in dimensions
are forbidden.

The shipped catalog defines monthly user and organization metrics for:

- `api.requests` (`request`)
- `ai.input_tokens` (`token`)
- `ai.output_tokens` (`token`)
- `asset.transfer_bytes` (`byte`)
- `workflow.runs` (`run`)

Shipped default limits are unlimited (`null`). A downstream app may replace the catalog or set a
subject quota override through the internal writer/CLI without changing metric identity.

## Events And Idempotency

A trusted producer supplies:

| Field | Rule |
|---|---|
| `source` | Lowercase dotted producer name, 1-64 bytes |
| `event_id` | Stable retry identifier, 1-128 safe ASCII bytes |
| `scope` / `subject_id` | Existing `user` or `organization` owner |
| `metric` | Exact catalog key available for that scope |
| `quantity` | Non-zero safe integer; see operation rules below |
| `dimensions` | Exact finite schema declared by the metric |
| `occurred_at` | Required for record; normalized to UTC |

`source + event_id` is globally unique within retained receipts. Repeating an identical request
returns the original receipt and does not update a counter or audit log again. Reusing the pair with
different semantic input returns `409 USAGE.IDEMPOTENCY_CONFLICT`. Fingerprints never contain
arbitrary payloads or secrets.

Every quantity, counter, limit, remaining value, and overage must stay within JavaScript's safe
integer range (`0..9007199254740991`, with negative record corrections down to
`-9007199254740991`). This keeps Go, PostgreSQL, JSON, and Web values exact.

### Record

`RecordUsage` stores a trusted fact after work occurred. Positive quantities add usage; negative
quantities correct prior usage. A correction that would make the period counter negative fails with
`USAGE.INVALID_EVENT`. Record never rejects a valid fact merely because it crosses a hard quota;
the resulting summary reports overage.

Record timestamps may be at most 24 hours old and no more than 5 minutes in the future. This bounded
late-event policy keeps period counters deterministic and prevents callers from escaping a current
quota by choosing an old period.

### Consume

`ConsumeUsage` uses the server clock, accepts only positive quantities, and atomically:

1. reserves the idempotency receipt;
2. locks the subject and current counter;
3. resolves the effective quota;
4. writes the counter only when the result remains within the hard limit;
5. finalizes an accepted or denied receipt in the same database transaction.

A denied decision is durable and returns `USAGE.QUOTA_EXCEEDED`; an identical retry returns the same
denial. Concurrent consumers cannot collectively pass the hard limit.

## Counters And Quotas

A **usage counter** is identified by scope, subject, metric, and UTC period start. It is always
non-negative and carries a monotonic storage version. `day` means `[00:00, next 00:00)` UTC;
`month` means `[first day 00:00, first day next month 00:00)` UTC.

The effective hard limit is a subject quota override when present, otherwise the metric's code
default. An unlimited quota is `null`. Operator quota set/reset uses explicit expected versions:

- no durable quota history has version `0`;
- set creates version `1` or increments the current version;
- reset keeps a tombstone and increments the version;
- a no-op does not increment the version;
- stale writes fail with `USAGE.QUOTA_VERSION_CONFLICT`.

Quota changes do not rewrite historical receipts or counters. Lowering a quota below current usage
is allowed and produces overage; future consumption is denied until the period changes or the limit
is raised/reset.

## Browser Read Contract

The browser contract is read-only and returns the finite current-period catalog. It is deliberately
unpaginated because the code-owned catalog is capped at 64 definitions.

| Method | Path | Auth | Behavior |
|---|---|---|---|
| `GET` | `/v1/usage/user` | Authentication session | Current user's effective summaries |
| `GET` | `/v1/organization-usage` | Authentication session + verified `Organization-Id` | Owner/admin organization summaries |

Organization members receive `403 COMMON.PERMISSION_DENIED`. Every response, including errors, is
`Cache-Control: private, no-store`, `Pragma: no-cache`, and varies on authorization; organization
responses also vary on `Organization-Id`.

Example `data`:

```json
[
  {
    "scope": "user",
    "metric": "api.requests",
    "unit": "request",
    "period": "month",
    "period_start": "2026-07-01T00:00:00Z",
    "period_end": "2026-08-01T00:00:00Z",
    "used": 42,
    "limit": 1000,
    "remaining": 958,
    "overage": 0,
    "over_limit": false,
    "quota_source": "override",
    "quota_version": 3,
    "updated_at": "2026-07-15T12:00:00Z"
  }
]
```

For unlimited metrics, `limit` and `remaining` are `null`, `overage` is `0`, and
`quota_source` is `default`. A missing counter returns `used: 0`, version-independent quota
metadata, and `updated_at: null`. No event IDs, sources, fingerprints, dimensions, or receipt rows
enter the browser response.

All successes use the shared envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": []
}
```

## Errors

| HTTP | `error_code` | Meaning |
|---:|---|---|
| 404 | `USAGE.METRIC_NOT_FOUND` | Metric is not in the selected scope catalog |
| 409 | `USAGE.IDEMPOTENCY_CONFLICT` | `source + event_id` was reused with different input |
| 412 | `USAGE.QUOTA_VERSION_CONFLICT` | Expected quota version is stale |
| 422 | `USAGE.INVALID_EVENT` | Quantity, dimensions, owner, or counter transition is invalid |
| 422 | `USAGE.EVENT_OUTSIDE_WINDOW` | Record timestamp violates late/future bounds |
| 428 | `USAGE.PRECONDITION_REQUIRED` | Quota mutation omitted an expected version |
| 429 | `USAGE.QUOTA_EXCEEDED` | Atomic consumption would cross the hard limit |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Starter or persistence dependency is unavailable |

HTTP consumers mapping a quota denial should include `Retry-After` when the period reset is known.
Messages remain human-readable only; callers branch on `error_code` or typed Go errors.

## Retention, Privacy, And Audit

Receipts contain only owner identity, metric, integer quantity, finite enum dimensions, timing,
counter transition, quota snapshot, and an SHA-256 fingerprint. They never contain request bodies,
prompts, filenames, URLs, provider payloads, prices, or secrets.

Receipts have a minimum 90-day idempotency horizon. `usage:prune` refuses a newer cutoff and deletes
only finalized receipts; counters and quota history remain. Producers must not replay an event after
its documented idempotency horizon. User account deletion removes user receipts, counters, and quota
overrides in the same account transaction after all deletion guards pass.

High-volume usage events are the immutable business ledger and do not create one audit row per
event. Operator quota changes, manual record/consume commands, and pruning write minimized
system-actor audit metadata without dimensions or event payloads. Audit failure remains best-effort
after the usage transaction and produces an operator warning.

## Operator Commands

The production CLI provides:

```text
usage:list
usage:record
usage:consume
usage:quota:set
usage:quota:reset
usage:prune
```

Commands require the `usage` starter, an available database, explicit scope/subject/metric input,
and stable source/event IDs for mutations. Quota set/reset requires an expected version. Command
output may show metric, quantity, counter, decision, limit, period, version, and prune count; it must
not show dimensions, fingerprints, or unrelated subject data.

## Deliberate Deferrals

- prices, currency, plans, subscriptions, invoices, taxes, and payment providers;
- boolean/static entitlements, grants, prepaid balances, rollover, and credit wallets;
- public/third-party event ingestion, CloudEvents compatibility, webhooks, and API credentials;
- arbitrary metrics, arbitrary JSON payloads, free-form dimensions, and high-cardinality analytics;
- per-dimension quota aggregation, sliding windows, billing-cycle anchors, and timezone-local periods;
- Kafka/stream processing, ClickHouse/warehouse export, and exactly-once external delivery;
- browser event history, quota administration, cost projections, and billing UI.

Downstream apps with high event volume can replace the recorder/reader seams with a dedicated
metering provider while preserving metric identity, subject ownership, idempotency, and quota
decision semantics.

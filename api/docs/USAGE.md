# Usage Starter

The optional `usage` starter provides trusted business metering, durable period counters, subject
quota overrides, and atomic consume decisions. It depends on `user`, `audit`, and `organization`:

```dotenv
OPTIONAL_STARTERS=organization,usage
```

The browser feature is activated independently. See [`../../contracts/USAGE.md`](../../contracts/USAGE.md)
for the canonical HTTP and domain contract.

## Ownership Boundary

Usage owns five framework-neutral seams in `internal/domain`:

- `UsageRecorder` records a fact after work has happened and permits bounded negative corrections.
- `UsageConsumer` atomically decides whether quota-controlled work may proceed.
- `UsageReader` returns the finite current-period summary catalog.
- `UsageQuotaWriter` changes one subject override with an expected version.
- `UsageMaintainer` prunes finalized receipts outside the idempotency horizon.

Application services call these seams directly. Do not add a public ingestion route, pass browser
payloads into the recorder, or make another starter import `internal/modules/usage`. HTTP request
telemetry and middleware rate limiting remain infrastructure concerns. Prices, plans, invoices,
entitlements, and payment providers remain outside this starter.

## Extending The Catalog

Edit `internal/modules/usage/catalog.go` when a downstream application needs another metric. A
definition must retain all of these properties:

- immutable, code-owned dotted metric identity;
- `user` or `organization` ownership;
- one `day` or `month` UTC period;
- a safe-integer default limit or `nil` for unlimited;
- at most eight dimensions, each with a finite code-owned allowlist;
- no free-form tags, request data, customer identifiers, or runtime definitions.

The complete scoped catalog is capped at 64 definitions. Update the strict Web schema and contract
in the same change. A new producer should use a stable dotted `source`, persist or derive one stable
`event_id`, and document its retry horizon. The retained `source + event_id` pair is the global
idempotency key.

## Record And Consume

Use `RecordUsage` only after the measured work is authoritative. Its timestamp must be within the
24-hour late window and five-minute future tolerance. Record can expose overage because rejecting a
completed fact would make the ledger inaccurate.

Use `ConsumeUsage` before hard-limited work. It uses the server clock and one database transaction
to reserve the receipt, lock the owner and current counter, resolve quota, and persist the accepted
or denied decision. Identical retries replay the original decision. Do not split check and increment
into separate calls.

Repository state is validated before use. Corrupt counters, quota rows, or incomplete receipts fail
closed as `COMMON.SERVICE_UNAVAILABLE`; they are not silently normalized.

## Quotas, Retention, And Deletion

Quota set/reset is compare-and-swap using an explicit expected version. Reset writes a tombstone so
old writers cannot recreate a stale override. Lowering a limit below current use is valid and makes
subsequent consumption fail until usage or policy permits it.

Receipts retain at least 90 days of idempotency history. Schedule `usage:prune` with a cutoff no
newer than that boundary; pruning removes only finalized receipts. Counters and quota history stay
durable. User account deletion removes user-owned usage state in the shared deletion transaction.

## Operator CLI

Run commands from `api/` with the same `OPTIONAL_STARTERS` selection and database configuration as
the deployed API:

```bash
go run ./cmd/luas usage:list --scope=user --subject-id=1
go run ./cmd/luas usage:record --source=ops.reconcile --event-id=evt-1 --scope=user --subject-id=1 --metric=api.requests --quantity=1 --occurred-at=2026-07-15T12:00:00Z
go run ./cmd/luas usage:consume --source=worker.dispatch --event-id=evt-2 --scope=user --subject-id=1 --metric=workflow.runs --quantity=1
go run ./cmd/luas usage:quota:set --scope=user --subject-id=1 --metric=api.requests --limit=1000 --expected-version=0
go run ./cmd/luas usage:quota:reset --scope=user --subject-id=1 --metric=api.requests --expected-version=1
go run ./cmd/luas usage:prune --before=2026-04-01T00:00:00Z
```

Manual mutations and pruning emit minimized system-actor audit metadata. Event IDs, fingerprints,
and dimensions must not enter audit output.

## Replacement

High-volume downstream systems may replace the domain implementations with a dedicated metering
adapter. Preserve metric identity, owner checks, exact idempotency, safe-integer transport, atomic
consume decisions, retention behavior, and the read contract. Keep provider SDKs outside product
modules and do not make external exactly-once delivery claims without an explicit outbox/stream
design.

## Verification

```bash
go test ./internal/modules/usage ./internal/bootstrap/operatorcommands ./internal/starter
go test ./...
go vet ./...
bash .agents/skills/sql-migration-review/scripts/check-migration.sh database/migrations/2026_07_15_050000_create_usage_tables.go
cd .. && python3 .agents/skills/luas-framework-review/scripts/check-usage-boundary.py
```

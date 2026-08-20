---
name: sql-migration-review
description: Review a Luas PostgreSQL migration for deploy compatibility, lock duration, backfill safety, index strategy, and recovery before it ships.
---

# PostgreSQL Migration Review

Review the moment a steady-state schema changes in production. Schema and query
design belong to `database-design`; this explicit workflow owns rollout risk.

## Scope

Use for a versioned migration under `api/database/migrations/` that changes a
table, column, type, default, nullability, constraint, index, or persisted data.
Read the migration, matching PO, query call sites, and available production row
count/traffic evidence. Do not load `database-design` unless the desired final
schema itself remains unresolved.

## Review

### Deploy compatibility

- Running old and new application instances must both tolerate the schema
  during the deployment window.
- Remove reads/writes in an earlier deploy before dropping a column or table.
- Rename through add-new, compatible reads/dual writes, backfill, cutover, then
  drop-old. Do not use a one-step rename on a live contract.
- Add new required data in phases so existing rows and old writers remain
  valid until the backfill and constraint are complete.

### PostgreSQL lock and transaction behavior

- Identify every statement's lock level and whether it scans or rewrites the
  table. Table size and peak write traffic are evidence, not guesses.
- Use `CREATE INDEX CONCURRENTLY` for a hot table when its longer, retryable
  build is preferable to blocking writes. It cannot run inside a transaction;
  the migration must declare the repository's non-transactional mode.
- A constant column default on supported PostgreSQL versions may avoid a table
  rewrite, but DDL still takes a lock. Volatile defaults, type changes,
  validation, and backfills need separate analysis.
- Add expensive constraints in a staged form when appropriate, validate them
  separately, and only then tighten nullability or remove compatibility code.

### Index and backfill behavior

- New query paths have indexes aligned with equality, range, and ordering
  predicates; removed indexes have usage evidence from PostgreSQL statistics.
- Backfills are bounded, resumable, idempotent, observable, and safe to retry.
  Batch by a stable key/range instead of issuing one unbounded update.
- Keep schema expansion, data backfill, application cutover, and destructive
  contraction in separate migrations or deploy stages.

### Recovery

- Every Luas migration implements its required `Down` path and is tested in the
  repository's rollback/reapply cycle.
- A `Down` implementation is not automatically the production recovery plan.
  Prefer rolling the application back against the compatible expanded schema;
  use a forward repair when reversing data or DDL would be destructive.
- Destructive contraction requires backup/export and restore evidence before
  deployment.

### Repository hygiene

- Use `api/database/migrations/YYYY_MM_DD_HHMMSS_<name>.go`.
- Keep one rollout concern per migration and register it with its owning
  starter/core assembly.
- Align the PO only at the deploy stage where new code can safely depend on the
  new shape.

## Static First Pass

```bash
bash api/.agents/skills/sql-migration-review/scripts/check-migration.sh \
  api/database/migrations/2026_07_25_000000_add_audit_retention_index.go
```

The script checks repository naming, required `Down`, unsupported dialect
markers, destructive or rename patterns, unbounded data changes, index mode,
and mixed concerns. It cannot prove production lock duration or compatibility.

## Verification

- Run the owning migration/package tests against disposable PostgreSQL.
- Run the relevant rollback/reapply test path, not only an empty-schema apply.
- For a starter migration, run its starter catalog/boundary guard.
- Record the application deploy order, migration transaction mode, recovery
  plan, and any production evidence used in the review.

## Output

Return numbered findings with migration and line evidence:

- **block**: data loss, incompatible deploy, unsupported dialect, or unsafe
  production lock.
- **fix-before-deploy**: missing recovery proof, index, bounded backfill, or
  rollout stage.
- **followup**: non-blocking clarity or maintainability improvement.

If clean, state how many migrations were reviewed and that no findings remain.

## Related Skills

Navigation only; do not load automatically:

- `database-design` when the target schema or query path is unresolved.
- `pr-description-writer` for explicitly requested release communication.

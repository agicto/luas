---
name: database-design
description: Design Luas persistence models, indexes, bounded queries, and table lifecycle. Use for schema or query-shape decisions; use sql-migration-review for rollout safety.
---

# Database Design

Design the steady-state PostgreSQL schema and repository query shape. Migration
ordering, deploy compatibility, lock duration, and backfill rollout belong to
`sql-migration-review`, not this skill.

## Authority

Read `api/AGENTS.md` for naming, layering, and the PostgreSQL-only boundary.
Read `api/docs/DATABASE.md` only when runtime configuration, pool behavior,
supported versions, or database performance evidence is active.

## Design Decisions

### Persistence boundary

- Keep domain values free of GORM. Persistence objects use the `PO` suffix and
  convert explicitly at the repository boundary.
- Define `TableName()` explicitly and use plural `snake_case` table names.
- Mark sensitive persistence fields with `json:"-"` even when the PO is not
  intended for transport.
- Choose lifecycle columns from record semantics. Mutable records commonly use
  `created_at` and `updated_at`; append-only records and hard-delete tables do
  not gain `deleted_at` by reflex.
- Represent SQL nullability deliberately. Use pointers or a reviewed nullable
  type only when absence differs from the zero value.

### PostgreSQL types and constraints

- PostgreSQL is the only compatibility target. Do not add SQLite or MySQL
  branches, fixtures, migrations, drivers, or tests.
- Prefer ordinary relational columns and constraints. Use `jsonb` only for data
  whose shape is intentionally flexible and whose query/index strategy is
  understood.
- Enforce identity and relationship invariants with database uniqueness,
  foreign keys, and check constraints where they remain valid independently of
  application code.
- Keep evolving business-state validation in the owning service rather than a
  database enum or trigger that is difficult to deploy compatibly.

### Indexes and query shape

- Derive indexes from real `WHERE`, join, ordering, and uniqueness paths. Do
  not index every field or choose composite order from cardinality alone.
- For a composite index, align leading columns with equality predicates, then
  range/order predicates used by the actual query.
- Keep list ordering deterministic. Unbounded lists require pagination; finite
  code-owned catalogs use the reviewed bounded-list annotation.
- Avoid query-per-row loops. Use a bounded join, preload, batch, or aggregate
  that preserves the repository contract.
- Select only response-owned columns when excluding sensitive or large fields
  materially changes correctness or cost.

## Performance Evidence

Inspect the generated SQL and run `EXPLAIN (ANALYZE, BUFFERS)` against
representative PostgreSQL data before claiming an index or query improvement.
Record exact application statement counts first; treat local latency and
allocations as comparison evidence.

Use the repository profile only when the changed seam matches it:

```bash
LUAS_TEST_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/luas_profile?sslmode=disable' \
  make benchmark-database
```

Do not enable `SkipDefaultTransaction`, `PrepareStmt`, or implicit prepared
statements globally without a transaction audit and deployment-pooler evidence.

## Verification

- Run `scripts/validate-db.sh <module-or-model-path>` for cheap PO checks.
- Run the owning repository/module tests.
- Verify constraints, transactions, locks, indexes, migrations, and query shape
  against disposable PostgreSQL through `LUAS_TEST_POSTGRES_DSN`.
- Use [examples/module_model_example.go](examples/module_model_example.go) only
  when a concrete PO/index example is needed; it is not a universal template.

## Completion Criteria

- The PO and table lifecycle match domain semantics.
- Constraints and indexes correspond to observable invariants and query paths.
- Lists are bounded and deterministically ordered.
- No unsupported dialect or database-free test is used as SQL evidence.
- Performance claims include PostgreSQL query-plan and statement-count proof.

## Related Skills

Navigation only; do not load automatically:

- `module-creation` for a new route-owning starter.
- `sql-migration-review` for deploying the schema change safely.

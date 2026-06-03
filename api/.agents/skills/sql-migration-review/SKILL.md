---
name: sql-migration-review
description: Review a GORM/SQL migration for backward compatibility, lock duration, index strategy, and rollback safety before it ships.
---

# SQL Migration Review

## Purpose

Migrations run once in production and rarely get a second chance. `database-design` covers the steady-state schema; this skill covers the moment of change.

## When to Use

Before merging or applying any migration that:

- adds, drops, or renames a column or table
- changes a column type, default, or nullability
- adds, drops, or changes an index
- backfills data
- alters constraints or foreign keys
- changes enum values

Skip for schema-less changes (seed data only, view-only adjustments to read replicas).

## Review Checklist

### 1. Backward Compatibility

The old code must keep working during the deploy window.

- [ ] Are running app instances compatible with the new schema for the duration of the deploy?
- [ ] If a column is being dropped: has all code been updated to stop referencing it in a *prior* deploy?
- [ ] If a column is being renamed: are you doing add-new + dual-write + cutover + drop-old (multi-step), not a single rename?
- [ ] If a NOT NULL column is being added: is there a default, or is the table small enough that a backfill is safe?
- [ ] If an enum value is being removed: confirm no rows still reference it.

### 2. Lock Duration

A long-held write lock blocks production traffic.

- [ ] For tables over ~1M rows: is the change index-only, or does it rewrite the whole table?
- [ ] MySQL: prefer `ALGORITHM=INPLACE, LOCK=NONE` where supported.
- [ ] Postgres: prefer `CREATE INDEX CONCURRENTLY` and `ADD COLUMN ... NULL` (no default) for hot tables.
- [ ] Avoid `ALTER TABLE ... ADD COLUMN ... DEFAULT <value> NOT NULL` on large tables — this rewrites every row on older MySQL and pre-11 Postgres.

### 3. Index Strategy

- [ ] Does every new query path have an index that covers it?
- [ ] Are composite indexes ordered with the highest-selectivity column first?
- [ ] Are you removing an index that is still in use? (check the slow query log / `pg_stat_user_indexes`)
- [ ] Are there duplicate or unused indexes you can drop in the same migration?
- [ ] Does the new index name follow project convention (`idx_<table>_<cols>`)?

### 4. Backfill Safety

- [ ] Is the backfill batched (e.g., `LIMIT 1000` in a loop), not a single statement on a large table?
- [ ] Is the backfill idempotent — safe to re-run if it dies halfway?
- [ ] Does the backfill use the new column / index, or is it scanning the whole table?
- [ ] If the backfill runs in app code (not SQL), is there a way to resume from the last processed ID?

### 5. Rollback

- [ ] Does the migration have a `Down` method that actually inverts the change?
- [ ] If data is destroyed by the migration, is there a backup or export step before destruction?
- [ ] Can you safely roll back the *app* deploy without rolling back the migration? (forward-compatible)
- [ ] Have you tested the rollback against a copy of prod data, not an empty dev DB?

### 6. Naming and File Hygiene

- [ ] Migration filename follows project convention (`api/internal/modules/<x>/migrations/` or wherever `database-design` documents it).
- [ ] Filename timestamp does not collide with another developer's pending migration.
- [ ] One concern per migration — don't bundle "add column X" with "rename table Y".
- [ ] GORM model definitions (`*_model.go`) updated to match the new schema.

## Source Material

Before reviewing, read:

1. The migration file itself.
2. The model definition (`*_model.go`) — does it match the new schema?
3. Existing query call sites (`grep` for the changed column) — will they still work?
4. The `database-design` skill for schema-level expectations.
5. Production stats if available: row count, peak QPS, write/read ratio.

## Output

Produce a numbered list of findings with severity:

- **block**: must fix before merge (data loss, prod lock, forward-incompat).
- **fix-before-deploy**: must fix before applying in prod (rollback gap, missing index, unsafe backfill).
- **followup**: nice-to-fix in a later PR (style, naming, redundant indexes).

If no findings, say so explicitly: "Reviewed N migrations, no findings."

## Anti-patterns

- "It worked locally" — local has 100 rows; prod has 100M.
- A single migration that adds + backfills + drops in one shot. Split it.
- Renaming a column directly — always add-new / drop-old across two deploys.
- Migrations that depend on app code running concurrently. Migrations must be self-contained.
- Using ORM auto-migration in production. Migrations should be explicit, reviewed, and versioned.
- "It's a small table" — confirm with `SELECT COUNT(*)`, don't guess.

## Helper Script

`scripts/check-migration.sh <path/to/migration.{sql,go}>` runs the cheap
static checks: filename pattern, Down path exists, high-risk patterns
(`ADD COLUMN ... DEFAULT ... NOT NULL`, `DROP COLUMN`, bare `RENAME`,
unbounded `DELETE`, non-`CONCURRENTLY` index creation on Postgres,
unbatched `UPDATE`), and migrations bundling too many concerns.

The script is a first pass; the human review (backward compat, lock
duration, rollback safety against real prod data) still applies.

```bash
bash api/.agents/skills/sql-migration-review/scripts/check-migration.sh \
    api/internal/modules/user/migrations/202604_add_email_verified.sql
```

## Pair With

- `database-design` for steady-state schema standards.
- `verification-before-completion` for the up-and-down execution check.
- `pr-description-writer` — the PR should include a Migration Notes section.

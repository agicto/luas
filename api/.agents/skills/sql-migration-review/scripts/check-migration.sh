#!/usr/bin/env bash

# SQL Migration Review — Static Checks
#
# Usage:
#   check-migration.sh <path/to/migration.sql>
#   check-migration.sh <path/to/migration.go>
#
# Runs the cheap, mechanical checks documented in ../SKILL.md so the
# expensive human review focuses on judgment calls. Does NOT replace
# the review checklist — it catches the most common drift only.

set -u

FILE=${1:-}
if [ -z "$FILE" ]; then
    echo "Usage: check-migration.sh <migration-file>" >&2
    exit 2
fi
if [ ! -f "$FILE" ]; then
    echo "❌ File not found: $FILE" >&2
    exit 2
fi

echo "🔍 Checking migration: $FILE"
echo "================================================"

ERRORS=0
WARNINGS=0
report_err() { echo "❌ $1"; ERRORS=$((ERRORS + 1)); }
report_warn() { echo "⚠️  $1"; WARNINGS=$((WARNINGS + 1)); }
report_ok() { echo "✅ $1"; }

# -----------------------------------------------------------------------------
# 1. Filename hygiene
# -----------------------------------------------------------------------------
BASE=$(basename "$FILE")
if [[ "$BASE" =~ ^[0-9]{14}_[a-z0-9_]+\.(sql|go)$ ]] || \
   [[ "$BASE" =~ ^[0-9]{10}_[a-z0-9_]+\.(sql|go)$ ]] || \
   [[ "$BASE" =~ ^[0-9]{4}_[0-9]{2}_[0-9]{2}_[0-9]{6}_[a-z0-9_]+\.(sql|go)$ ]]; then
    report_ok "Filename follows timestamp_snake_case pattern"
else
    report_warn "Filename '$BASE' does not match the standard timestamp_<name>.{sql,go} pattern"
fi

# -----------------------------------------------------------------------------
# 2. Down / rollback present
# -----------------------------------------------------------------------------
if grep -qE "Down\(|DOWN|-- down|down:" "$FILE"; then
    report_ok "Down / rollback path detected"
else
    report_err "No Down / rollback path detected — every migration MUST have one"
fi

# -----------------------------------------------------------------------------
# 3. High-risk patterns
# -----------------------------------------------------------------------------
if grep -qiE "ALTER TABLE[^;]+ADD COLUMN[^;]+DEFAULT[^;]+NOT NULL" "$FILE"; then
    report_warn "Found ADD COLUMN ... DEFAULT ... NOT NULL — rewrites every row on large tables; verify table size"
fi

if grep -qiE "DROP COLUMN|DROP TABLE" "$FILE"; then
    report_warn "Found DROP COLUMN / DROP TABLE — confirm no live code still references it (prior deploy removed reads/writes)"
fi

if grep -qiE "RENAME COLUMN|RENAME TO" "$FILE"; then
    report_err "Found RENAME — split into add-new + dual-write + cutover + drop-old across migrations"
fi

if grep -qiE "DELETE FROM[^;]*;" "$FILE" && ! grep -qiE "DELETE FROM[^;]*WHERE" "$FILE"; then
    report_err "Found unbounded DELETE without WHERE — likely catastrophic"
fi

# -----------------------------------------------------------------------------
# 4. Index strategy hints
# -----------------------------------------------------------------------------
if grep -qiE "CREATE INDEX" "$FILE" && ! grep -qiE "CREATE INDEX CONCURRENTLY|--postgres-only" "$FILE" && grep -qi "postgres" "$FILE"; then
    report_warn "Postgres CREATE INDEX without CONCURRENTLY locks the table; prefer CONCURRENTLY on hot tables"
fi

# -----------------------------------------------------------------------------
# 5. Backfill patterns
# -----------------------------------------------------------------------------
if grep -qiE "UPDATE[^;]+SET" "$FILE" && ! grep -qiE "LIMIT|WHERE id (>|BETWEEN)|batch" "$FILE"; then
    report_warn "Found UPDATE without LIMIT / batching — risky on large tables"
fi

# -----------------------------------------------------------------------------
# 6. Multiple concerns
# -----------------------------------------------------------------------------
CONCERNS=0
grep -qiE "ADD COLUMN|CREATE TABLE" "$FILE" && CONCERNS=$((CONCERNS + 1))
grep -qiE "DROP COLUMN|DROP TABLE" "$FILE" && CONCERNS=$((CONCERNS + 1))
grep -qiE "RENAME COLUMN|RENAME TO" "$FILE" && CONCERNS=$((CONCERNS + 1))
grep -qiE "CREATE INDEX|DROP INDEX" "$FILE" && CONCERNS=$((CONCERNS + 1))
if [ $CONCERNS -gt 2 ]; then
    report_warn "Migration touches $CONCERNS distinct kinds of change — consider splitting"
fi

# -----------------------------------------------------------------------------
echo ""
echo "================================================"
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✅ Static checks clean. Human review still required for backward compat, lock duration, rollback safety."
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠️  $WARNINGS warning(s). Address before merge or document why ignored."
    exit 0
else
    echo "❌ $ERRORS error(s) and $WARNINGS warning(s). Fix before merge."
    exit 1
fi

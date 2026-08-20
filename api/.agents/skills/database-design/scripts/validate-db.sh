#!/bin/bash

# Database Standards Validation Script
# Usage: ./validate-db.sh <module_name | path_to_model_file>
#
# Accepts either a module name (resolved to internal/modules/<name>/model.go)
# or an explicit file path, so this validator can be used in the same CI
# loop as the other skill validators.

set -euo pipefail

ARG=${1:-}

if [ -z "$ARG" ]; then
    echo "Usage: ./validate-db.sh <module_name | path_to_model_file>"
    exit 1
fi

# Resolve: if the arg is a file, use it directly; otherwise treat as module name.
if [ -f "$ARG" ]; then
    MODEL_FILE="$ARG"
else
    MODEL_FILE="internal/modules/${ARG}/model.go"
fi

if [ ! -f "$MODEL_FILE" ]; then
    echo "❌ Model file not found: $MODEL_FILE"
    exit 1
fi

echo "🔍 Validating Database standards for '$(basename "$MODEL_FILE")'..."
echo "=============================================="

ERRORS=0
WARNINGS=0

# 1. Check for PO suffix on struct names
if grep -q "type.*struct" "$MODEL_FILE" && ! grep -q "PO struct" "$MODEL_FILE"; then
    echo "❌ Struct definitions in model.go should have 'PO' suffix (e.g., UserPO)."
    ERRORS=$((ERRORS + 1))
else
    echo "✅ Struct naming conventions passed."
fi

# 2. Check for TableName() method
if ! grep -q "func.*TableName().*string" "$MODEL_FILE"; then
    echo "❌ Missing TableName() method. Luas persistence objects define table ownership explicitly."
    ERRORS=$((ERRORS + 1))
else
    echo "✅ TableName() method detected."
fi

# 3. Report lifecycle fields without imposing one table shape.
LIFECYCLE_FIELDS=()
for field in "ID" "CreatedAt" "UpdatedAt" "DeletedAt"; do
    if grep -q "$field" "$MODEL_FILE"; then
        LIFECYCLE_FIELDS+=("$field")
    fi
done

if [ ${#LIFECYCLE_FIELDS[@]} -gt 0 ]; then
    echo "ℹ️  Lifecycle fields detected: ${LIFECYCLE_FIELDS[*]}. Confirm each matches record semantics."
else
    echo "ℹ️  No conventional lifecycle fields detected. Confirm the table uses an intentional key and lifecycle."
fi

# 4. Check only explicit column names. GORM directives such as CASCADE are case-sensitive.
EXPLICIT_COLUMNS=$(
    grep -oE 'gorm:"[^"]*"' "$MODEL_FILE" |
        grep -oE 'column:[^;"]+' |
        cut -d: -f2- ||
        true
)
NON_SNAKE_COLUMNS=""
if [ -n "$EXPLICIT_COLUMNS" ]; then
    NON_SNAKE_COLUMNS=$(printf '%s\n' "$EXPLICIT_COLUMNS" | grep -Ev '^[a-z][a-z0-9_]*$' || true)
fi

if [ -n "$NON_SNAKE_COLUMNS" ]; then
    echo "⚠️  Explicit GORM column names must use snake_case:"
    printf '   %s\n' "$NON_SNAKE_COLUMNS"
    WARNINGS=$((WARNINGS + 1))
fi

echo "=============================================="
if [ $ERRORS -eq 0 ]; then
    echo "SUCCESS: Database standards mostly met ($WARNINGS warnings)."
    exit 0
else
    echo "FAILURE: Found $ERRORS errors. Please fix before proceeding."
    exit 1
fi

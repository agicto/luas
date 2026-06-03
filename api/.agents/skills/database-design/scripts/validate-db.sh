#!/bin/bash

# Database Standards Validation Script
# Usage: ./validate-db.sh <module_name | path_to_model_file>
#
# Accepts either a module name (resolved to internal/modules/<name>/model.go)
# or an explicit file path, so this validator can be used in the same CI
# loop as the other skill validators.

set -e

ARG=$1

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

echo "🔍 Validating Database standards for '$(basename $MODEL_FILE)'..."
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
    echo "⚠️  Missing TableName() method. Explicit table names are recommended."
    WARNINGS=$((WARNINGS + 1))
else
    echo "✅ TableName() method detected."
fi

# 3. Check for baseline lifecycle fields
for field in "ID" "CreatedAt" "UpdatedAt"; do
    if ! grep -q "$field" "$MODEL_FILE"; then
        echo "❌ Missing mandatory field: $field"
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -eq 0 ]; then
    echo "✅ Baseline lifecycle fields present."
fi

if grep -q "DeletedAt" "$MODEL_FILE"; then
    echo "✅ Soft delete field detected."
else
    echo "ℹ️  Soft delete field not detected. Confirm the table does not require soft delete."
fi

# 4. Check for snake_case in gorm labels
if grep "gorm:\"" "$MODEL_FILE" | grep -v "primaryKey" | grep -q "[A-Z]"; then
    # This is a very simple check that might have false positives, but it targets mixedCase in tags
    echo "⚠️  Detected possible non-snake_case naming in GORM tags. Check column names."
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

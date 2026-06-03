#!/bin/bash

# Logging Standards Validation Script (slog)
# Usage: ./validate-logging.sh <module_name>
#
# Checks that a module under api/internal/modules/<name>/ follows the
# log/slog conventions documented in ../SKILL.md.

set -e

MODULE=$1
MODULE_DIR="internal/modules/${MODULE}"

if [ -z "$MODULE" ]; then
    echo "Usage: ./validate-logging.sh <module_name>"
    exit 1
fi

if [ ! -d "${MODULE_DIR}" ]; then
    echo "❌ Module directory not found: ${MODULE_DIR}"
    exit 1
fi

echo "🔍 Validating logging standards for '${MODULE}' module..."
echo "=============================================="

ERRORS=0
WARNINGS=0

# =============================================================================
# 1. Forbidden loggers — no logrus / zap / zerolog allowed in new code
# =============================================================================

echo ""
echo "🚫 Level 1: Checking for forbidden logger imports..."

FORBIDDEN=$(grep -rn "github.com/sirupsen/logrus\|go.uber.org/zap\|github.com/rs/zerolog" "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" || true)
if [ -n "$FORBIDDEN" ]; then
    echo "❌ Found forbidden logger imports (use log/slog instead):"
    echo "$FORBIDDEN"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ No forbidden logger imports"
fi

# =============================================================================
# 2. Legacy log.Printf — flag but don't fail
# =============================================================================

echo ""
echo "📊 Level 2: Checking for legacy log.Printf usage..."

LEGACY=$(grep -rn 'log\.Printf\|log\.Println\|log\.Fatal\|log\.Fatalf' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" | grep -v 'log/slog' || true)
if [ -n "$LEGACY" ]; then
    echo "⚠️  Found stdlib log.Printf usage (prefer log/slog):"
    echo "$LEGACY"
    WARNINGS=$((WARNINGS + 1))
else
    echo "✅ No legacy log.Printf usage"
fi

# =============================================================================
# 3. String interpolation in slog calls (anti-pattern)
# =============================================================================

echo ""
echo "🔍 Level 3: Checking for fmt.Sprintf inside slog calls..."

INTERPOLATED=$(grep -rn 'slog\.\(Debug\|Info\|Warn\|Error\)\(Context\)\?.*fmt\.Sprintf' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" || true)
if [ -n "$INTERPOLATED" ]; then
    echo "⚠️  Found fmt.Sprintf inside slog calls (use structured args instead):"
    echo "$INTERPOLATED"
    WARNINGS=$((WARNINGS + 1))
else
    echo "✅ No string interpolation anti-patterns"
fi

# =============================================================================
# 4. Sensitive data heuristic
# =============================================================================

echo ""
echo "🔐 Level 4: Checking for sensitive data in logs..."

SENSITIVE=$(grep -rin 'slog\.\(Debug\|Info\|Warn\|Error\).*"\(password\|credit_card\|ssn\|secret\)"' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" || true)
if [ -n "$SENSITIVE" ]; then
    echo "❌ Possible sensitive data in logs:"
    echo "$SENSITIVE"
    echo "   Never log passwords / credit cards / SSNs / secrets."
    ERRORS=$((ERRORS + 1))
else
    echo "✅ No obvious sensitive data in logs"
fi

# =============================================================================
# 5. Context-aware logging — prefer InfoContext / ErrorContext
# =============================================================================

echo ""
echo "🔗 Level 5: Checking context-aware logging..."

NON_CTX=$(grep -rn 'slog\.\(Debug\|Info\|Warn\|Error\)(' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v 'Context(' | grep -v "//" || true)
if [ -n "$NON_CTX" ]; then
    NON_CTX_COUNT=$(echo "$NON_CTX" | wc -l | tr -d ' ')
    echo "⚠️  Found ${NON_CTX_COUNT} slog calls without Context variant (prefer slog.InfoContext / ErrorContext etc.):"
    echo "$NON_CTX" | head -5
    WARNINGS=$((WARNINGS + 1))
else
    echo "✅ Slog calls use Context variants"
fi

# =============================================================================
# 6. Error logs include err field
# =============================================================================

echo ""
echo "❌ Level 6: Checking error logging shape..."

ERR_LOGS=$(grep -rn 'slog\.ErrorContext\|slog\.Error(' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" || true)
if [ -n "$ERR_LOGS" ]; then
    ERR_COUNT=$(echo "$ERR_LOGS" | wc -l | tr -d ' ')
    echo "✅ Found ${ERR_COUNT} error log site(s)"
    NO_ERR_FIELD=$(echo "$ERR_LOGS" | grep -v '"err"' || true)
    if [ -n "$NO_ERR_FIELD" ]; then
        echo "⚠️  Some error logs may not include an \"err\" field:"
        echo "$NO_ERR_FIELD" | head -3
        WARNINGS=$((WARNINGS + 1))
    fi
fi

# =============================================================================
# 7. Level counts
# =============================================================================

echo ""
echo "📊 Level 7: Slog level distribution..."

for level in "Debug" "Info" "Warn" "Error"; do
    COUNT=$(grep -rc "slog\.${level}\(Context\)\?(" "${MODULE_DIR}"/*.go 2>/dev/null | awk -F: '{sum+=$2} END {print sum+0}')
    if [ "$COUNT" -gt 0 ]; then
        echo "   ${level}: $COUNT call(s)"
    fi
done

# =============================================================================
# Summary
# =============================================================================

echo ""
echo "=============================================="
echo "📊 Validation Summary"
echo "=============================================="

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✅ All logging standards checks passed for '${MODULE}'."
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠️  Found $WARNINGS warning(s). Refer to .agents/skills/logging-standards/SKILL.md."
    exit 0
else
    echo "❌ Found $ERRORS error(s) and $WARNINGS warning(s). Fix the critical issues above."
    exit 1
fi

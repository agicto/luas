#!/bin/bash

# Logging Standards Validation Script
# Usage: ./validate-logging.sh <module_name>

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

# Track validation status
ERRORS=0
WARNINGS=0

# =============================================================================
# 1. Check for Structured Logging (logrus/zap)
# =============================================================================

echo ""
echo "📊 Level 1: Checking structured logging..."

# Check for logger import
if grep -r "github.com/sirupsen/logrus\|go.uber.org/zap" "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//"; then
    echo "✅ Structured logging library detected"
else
    echo "⚠️  No structured logging library found"
    echo "   Recommended: github.com/sirupsen/logrus or go.uber.org/zap"
    WARNINGS=$((WARNINGS + 1))
fi

# =============================================================================
# 2. Check for String Interpolation in Logs (Anti-pattern)
# =============================================================================

echo ""
echo "🚫 Level 2: Checking for anti-patterns..."

# Check for fmt.Printf style logging (anti-pattern)
STRING_LOGS=$(grep -rn 'logger\.Infof\|logger\.Debugf\|logger\.Warnf\|logger\.Errorf\|log\.Printf' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" || true)

if [ -n "$STRING_LOGS" ]; then
    echo "⚠️  Found string interpolation in logs (prefer structured fields):"
    echo "$STRING_LOGS"
    echo "   Use: logger.WithFields(logrus.Fields{...}).Info(...)"
    WARNINGS=$((WARNINGS + 1))
else
    echo "✅ No string interpolation anti-patterns found"
fi

# =============================================================================
# 3. Check for Structured Fields Usage
# =============================================================================

echo ""
echo "📋 Level 3: Checking structured fields..."

# Check for WithFields usage
if grep -rq 'WithFields\|WithField' "${MODULE_DIR}"/*.go 2>/dev/null; then
    echo "✅ Structured fields detected"
    
    # Count usage
    FIELD_COUNT=$(grep -rc 'WithFields\|WithField' "${MODULE_DIR}"/*.go 2>/dev/null | awk -F: '{sum+=$2} END {print sum}')
    echo "   Found $FIELD_COUNT structured field usages"
else
    echo "⚠️  No structured fields found"
    echo "   Use: logger.WithFields(logrus.Fields{...})"
    WARNINGS=$((WARNINGS + 1))
fi

# =============================================================================
# 4. Check for High-Cardinality Fields
# =============================================================================

echo ""
echo "🔍 Level 4: Checking high-cardinality fields..."

# Check for important identifiers
for field in "request_id" "user_id" "tenant_id"; do
    if grep -rq "\"${field}\"" "${MODULE_DIR}"/*.go 2>/dev/null; then
        echo "✅ Found '${field}' field"
    else
        echo "⚠️  No '${field}' field detected"
    fi
done

# =============================================================================
# 5. Check for Context Propagation
# =============================================================================

echo ""
echo "🔗 Level 5: Checking context propagation..."

# Check if logger is passed through context
if grep -rq 'FromContext\|WithLogger' "${MODULE_DIR}"/*.go 2>/dev/null; then
    echo "✅ Context propagation patterns detected"
else
    echo "⚠️  No context propagation detected"
    echo "   Recommended: Pass logger through context"
    WARNINGS=$((WARNINGS + 1))
fi

# =============================================================================
# 6. Check for Sensitive Data in Logs
# =============================================================================

echo ""
echo "🔐 Level 6: Checking for sensitive data..."

# Check for common sensitive field names
SENSITIVE_FIELDS=$(grep -rin '"password"\|"credit_card"\|"ssn"\|"token"\|"secret"' "${MODULE_DIR}"/*.go 2>/dev/null | grep -i 'log\|info\|debug\|warn\|error' || true)

if [ -n "$SENSITIVE_FIELDS" ]; then
    echo "⚠️  Possible sensitive data in logs:"
    echo "$SENSITIVE_FIELDS"
    echo "   NEVER log passwords, credit cards, SSNs, tokens, secrets"
    ERRORS=$((ERRORS + 1))
else
    echo "✅ No obvious sensitive data in logs"
fi

# =============================================================================
# 7. Check for Error Logging
# =============================================================================

echo ""
echo "❌ Level 7: Checking error logging..."

# Check if errors are logged with context
ERROR_LOGS=$(grep -rn '\.Error(' "${MODULE_DIR}"/*.go 2>/dev/null | grep -v "//" || true)

if [ -n "$ERROR_LOGS" ]; then
    ERROR_COUNT=$(echo "$ERROR_LOGS" | wc -l)
    echo "✅ Found $ERROR_COUNT error log statements"
    
    # Check if errors include context
    if grep -rq 'WithFields.*Error\|WithError' "${MODULE_DIR}"/*.go 2>/dev/null; then
        echo "✅ Error logs include context"
    else
        echo "⚠️  Error logs may lack context"
        echo "   Use: logger.WithFields(...).WithError(err).Error(...)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "⚠️  No error logging found (may not be needed)"
fi

# =============================================================================
# 8. Check Log Levels
# =============================================================================

echo ""
echo "📊 Level 8: Checking log level usage..."

for level in "Debug" "Info" "Warn" "Error"; do
    COUNT=$(grep -rc "\\.${level}(" "${MODULE_DIR}"/*.go 2>/dev/null | awk -F: '{sum+=$2} END {print sum}')
    if [ "$COUNT" -gt 0 ]; then
        echo "✅ ${level}: $COUNT occurrences"
    fi
done

# Check for Fatal (should be rare)
FATAL_COUNT=$(grep -rc "\\.Fatal(" "${MODULE_DIR}"/*.go 2>/dev/null | awk -F: '{sum+=$2} END {print sum}')
if [ "$FATAL_COUNT" -gt 0 ]; then
    echo "⚠️  Fatal: $FATAL_COUNT occurrences (use sparingly!)"
fi

# =============================================================================
# Generate Report
# =============================================================================

echo ""
echo "=============================================="
echo "📊 Validation Summary"
echo "=============================================="

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✅ All logging standards checks passed!"
    echo ""
    echo "Module '${MODULE}' follows Luas logging standards."
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠️  Found $WARNINGS warning(s)"
    echo ""
    echo "Module '${MODULE}' mostly follows standards, but has some improvements."
    echo "Refer to: .agent/skills/logging-standards/SKILL.md"
    exit 0
else
    echo "❌ Found $ERRORS error(s) and $WARNINGS warning(s)"
    echo ""
    echo "Please fix the critical issues above."
    echo "Refer to: .agent/skills/logging-standards/SKILL.md"
    exit 1
fi

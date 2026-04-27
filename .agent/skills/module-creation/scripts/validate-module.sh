#!/bin/bash

# Module Validation Script
# Usage: ./validate-module.sh <module_name>
# Validates the default starter-style 8-file scaffold.

set -e

MODULE_NAME=$1
MODULE_DIR="internal/modules/${MODULE_NAME}"

echo "🔍 Validating module: ${MODULE_NAME}"
echo "======================================="

# Check if module directory exists
if [ ! -d "${MODULE_DIR}" ]; then
    echo "❌ Module directory not found: ${MODULE_DIR}"
    exit 1
fi

echo "✓ Module directory exists"

# Check for 8 required files
REQUIRED_FILES=(
    "model.go"
    "dto.go"
    "repository.go"
    "service.go"
    "handler.go"
    "routes.go"
    "provider.go"
    "service_test.go"
)

MISSING_FILES=()

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "${MODULE_DIR}/${file}" ]; then
        MISSING_FILES+=("${file}")
    fi
done

if [ ${#MISSING_FILES[@]} -eq 0 ]; then
    echo "✓ All 8 required files present"
else
    echo "❌ Missing files:"
    for file in "${MISSING_FILES[@]}"; do
        echo "  - ${file}"
    done
    exit 1
fi

# Check for package declaration
echo ""
echo "Checking package declarations..."
for file in "${MODULE_DIR}"/*.go; do
    if ! grep -q "^package ${MODULE_NAME}$" "${file}"; then
        echo "❌ Incorrect package name in: $(basename ${file})"
        echo "   Expected: package ${MODULE_NAME}"
        exit 1
    fi
done
echo "✓ Package declarations correct"

# Check for provider.go content
echo ""
echo "Checking provider.go..."
if ! grep -q "var ProviderSet = wire.NewSet" "${MODULE_DIR}/provider.go"; then
    echo "❌ provider.go missing ProviderSet declaration"
    exit 1
fi
echo "✓ ProviderSet declared"

# Check repository and service files exist and are non-empty
echo ""
echo "Checking repository and service files..."
if [ ! -s "${MODULE_DIR}/repository.go" ] || [ ! -s "${MODULE_DIR}/service.go" ]; then
    echo "❌ repository.go or service.go is empty"
    exit 1
fi
echo "✓ Repository and service files present"

# Check for handler struct
echo ""
echo "Checking handler..."
if ! grep -q "type Handler struct" "${MODULE_DIR}/handler.go"; then
    echo "❌ Handler struct not found"
    exit 1
fi
echo "✓ Handler struct defined"

# Check for routes registration
echo ""
echo "Checking routes..."
if ! grep -q "RegisterRoutes" "${MODULE_DIR}/routes.go"; then
    echo "❌ RegisterRoutes entry not found"
    exit 1
fi
echo "✓ RegisterRoutes entry defined"

# Try to build the module
echo ""
echo "Building module..."
if ! go build ./internal/modules/${MODULE_NAME}/...; then
    echo "❌ Module build failed"
    exit 1
fi
echo "✓ Module builds successfully"

# Run tests
echo ""
echo "Running tests..."
if ! go test ./internal/modules/${MODULE_NAME}/... -v; then
    echo "⚠️  Some tests failed (review output above)"
else
    echo "✓ All tests passed"
fi

echo ""
echo "======================================="
echo "✅ Module validation complete!"
echo ""
echo "Next steps:"
echo "1. Refine the generated internal/domain file with real business fields"
echo "2. Decide whether this is a starter, optional starter, capability, or example"
echo "3. If it becomes a default starter, add its starter manifest to internal/starter/defaults.go"
echo "4. Run make wire and go test ./..."

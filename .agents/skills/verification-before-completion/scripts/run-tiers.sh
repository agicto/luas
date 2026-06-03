#!/usr/bin/env bash

# Verification Before Completion — Tier Runner
#
# Usage:
#   run-tiers.sh <tier>           # 0, 1, or 2
#   run-tiers.sh 0                # static checks only
#   run-tiers.sh 1                # 0 + targeted tests
#   run-tiers.sh 2                # 0 + 1 + full module suite
#
# Detects whether you're in api/ or web/ by walking up for go.mod or
# package.json and runs the appropriate toolchain.
#
# Exit code is 0 iff every chosen tier passes.

set -u

TIER=${1:-0}
if ! [[ "$TIER" =~ ^[012]$ ]]; then
    echo "Usage: run-tiers.sh <0|1|2>" >&2
    exit 2
fi

# Walk up looking for go.mod or package.json to figure out which half we're in.
DIR=$(pwd)
KIND=""
while [ "$DIR" != "/" ]; do
    if [ -f "$DIR/go.mod" ]; then KIND="go"; ROOT="$DIR"; break; fi
    if [ -f "$DIR/package.json" ]; then KIND="node"; ROOT="$DIR"; break; fi
    DIR=$(dirname "$DIR")
done

if [ -z "$KIND" ]; then
    echo "❌ Could not locate go.mod or package.json upward from $(pwd)" >&2
    exit 2
fi

cd "$ROOT"
echo "🔍 Verification at tier $TIER in $ROOT ($KIND)"
echo "================================================"

FAILED=0
record() {
    if [ "$1" -ne 0 ]; then
        echo "  ❌ $2"
        FAILED=1
    else
        echo "  ✅ $2"
    fi
}

# -----------------------------------------------------------------------------
# Tier 0 — Static
# -----------------------------------------------------------------------------
echo ""
echo "Tier 0 — Static"
if [ "$KIND" = "go" ]; then
    go build ./... > /tmp/run-tiers.log 2>&1; record $? "go build ./..."
    if command -v golangci-lint >/dev/null 2>&1; then
        golangci-lint run ./... --timeout=2m > /tmp/run-tiers.log 2>&1; record $? "golangci-lint"
    else
        echo "  ⚠️  golangci-lint not installed, skipped"
    fi
else
    pnpm type-check > /tmp/run-tiers.log 2>&1; record $? "pnpm type-check"
    pnpm lint > /tmp/run-tiers.log 2>&1; record $? "pnpm lint"
fi

if [ "$TIER" -eq 0 ]; then
    echo ""
    [ $FAILED -eq 0 ] && echo "✅ Tier 0 passed" || echo "❌ Tier 0 failed"
    exit $FAILED
fi

# -----------------------------------------------------------------------------
# Tier 1 — Local execution (tests)
# -----------------------------------------------------------------------------
echo ""
echo "Tier 1 — Local execution"
if [ "$KIND" = "go" ]; then
    go test ./... > /tmp/run-tiers.log 2>&1; record $? "go test ./..."
else
    pnpm test -- --run > /tmp/run-tiers.log 2>&1; record $? "pnpm test"
fi

if [ "$TIER" -eq 1 ]; then
    echo ""
    [ $FAILED -eq 0 ] && echo "✅ Tier 0 + 1 passed" || echo "❌ Failures above"
    exit $FAILED
fi

# -----------------------------------------------------------------------------
# Tier 2 — Build + race detector / full build
# -----------------------------------------------------------------------------
echo ""
echo "Tier 2 — End-to-end"
if [ "$KIND" = "go" ]; then
    go test -race ./... > /tmp/run-tiers.log 2>&1; record $? "go test -race"
else
    pnpm build > /tmp/run-tiers.log 2>&1; record $? "pnpm build"
fi

echo ""
[ $FAILED -eq 0 ] && echo "✅ All tiers (0+1+2) passed" || echo "❌ Failures above"
exit $FAILED

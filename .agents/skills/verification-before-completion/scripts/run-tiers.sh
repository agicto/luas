#!/usr/bin/env bash

# Verification Before Completion — Tier Runner
#
# Usage:
#   run-tiers.sh <tier> [scope...]
#
# Tier:
#   0  static checks only (build + lint)
#   1  0 + tests
#   2  0 + 1 + race / production build
#
# Scope (optional): one or more Go packages or test path patterns.
#   Defaults to ./... — i.e. the whole tree.
#   Web side: scope is ignored; the Node toolchain uses its own scripts.
#
# Detects whether you're in api/ or web/ by walking up for go.mod or
# package.json and runs the appropriate toolchain.
#
# Exit code is 0 iff every chosen tier passes within the chosen scope.

set -u

TIER=${1:-0}
if ! [[ "$TIER" =~ ^[012]$ ]]; then
    echo "Usage: run-tiers.sh <0|1|2> [scope...]" >&2
    exit 2
fi
shift || true
SCOPE=("$@")
if [ ${#SCOPE[@]} -eq 0 ]; then
    SCOPE=("./...")
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
if [ "$KIND" = "node" ] && [ -z "${CI:-}" ] && [ ! -t 0 ]; then
    export CI=true
fi
PNPM=(pnpm)
if [ "$KIND" = "node" ] && command -v corepack >/dev/null 2>&1; then
    PNPM=(corepack pnpm)
fi

echo "🔍 Verification at tier $TIER in $ROOT ($KIND)"
echo "================================================"

FAILED=0
LOG_FILE="${TMPDIR:-/tmp}/run-tiers.log"
record() {
    if [ "$1" -ne 0 ]; then
        echo "  ❌ $2"
        if [ -s "$LOG_FILE" ]; then
            echo "     Last output:"
            tail -n 40 "$LOG_FILE" | sed 's/^/       /'
        fi
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
    go build "${SCOPE[@]}" > "$LOG_FILE" 2>&1; record $? "go build ${SCOPE[*]}"
    if command -v golangci-lint >/dev/null 2>&1; then
        golangci-lint run "${SCOPE[@]}" --timeout=2m > "$LOG_FILE" 2>&1; record $? "golangci-lint run ${SCOPE[*]}"
    else
        echo "  ⚠️  golangci-lint not installed, skipped"
    fi
else
    "${PNPM[@]}" type-check > "$LOG_FILE" 2>&1; record $? "pnpm type-check"
    "${PNPM[@]}" lint > "$LOG_FILE" 2>&1; record $? "pnpm lint"
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
    go test "${SCOPE[@]}" > "$LOG_FILE" 2>&1; record $? "go test ${SCOPE[*]}"
else
    "${PNPM[@]}" test -- --run > "$LOG_FILE" 2>&1; record $? "pnpm test"
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
    go test -race "${SCOPE[@]}" > "$LOG_FILE" 2>&1; record $? "go test -race ${SCOPE[*]}"
else
    "${PNPM[@]}" build > "$LOG_FILE" 2>&1; record $? "pnpm build"
fi

echo ""
[ $FAILED -eq 0 ] && echo "✅ All tiers (0+1+2) passed" || echo "❌ Failures above"
exit $FAILED

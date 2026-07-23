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
if [ "$KIND" = "go" ] && [ -n "${CODEX_SANDBOX:-}" ] && [ -z "${GOCACHE:-}" ]; then
    export GOCACHE="${TMPDIR:-/tmp}/luas-go-build-cache"
    mkdir -p "$GOCACHE"
    if [ -z "${GOLANGCI_LINT_CACHE:-}" ]; then
        export GOLANGCI_LINT_CACHE="${TMPDIR:-/tmp}/luas-golangci-lint-cache"
        mkdir -p "$GOLANGCI_LINT_CACHE"
    fi
fi
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
COMMAND_INDEX=0
LOG_TAIL_LINES=${RUN_TIERS_LOG_TAIL_LINES:-80}
if ! [[ "$LOG_TAIL_LINES" =~ ^[0-9]+$ ]] || [ "$LOG_TAIL_LINES" -eq 0 ]; then
    LOG_TAIL_LINES=80
fi

if [ -n "${RUN_TIERS_LOG_DIR:-}" ]; then
    LOG_DIR=$RUN_TIERS_LOG_DIR
else
    TMP_ROOT=${TMPDIR:-/tmp}
    TMP_ROOT=${TMP_ROOT%/}
    LOG_DIR="$TMP_ROOT/luas-run-tiers.$(date +%Y%m%d%H%M%S).$$"
fi
mkdir -p "$LOG_DIR"

slugify() {
    printf '%s' "$1" | tr -cs 'A-Za-z0-9_.-' '-' | sed 's/^-//;s/-$//'
}

run_step() {
    local label=$1
    shift
    local slug log_file status

    COMMAND_INDEX=$((COMMAND_INDEX + 1))
    slug=$(slugify "$label")
    if [ -z "$slug" ]; then
        slug="command"
    fi
    log_file="$LOG_DIR/$(printf '%02d' "$COMMAND_INDEX")-${slug}.log"

    "$@" >"$log_file" 2>&1
    status=$?

    if [ "$status" -eq 0 ]; then
        echo "  ✅ $label"
        return
    fi

    echo "  ❌ $label"
    echo "     Exit code: $status"
    echo "     Log: $log_file"
    if [ -s "$log_file" ]; then
        echo "     Last $LOG_TAIL_LINES lines:"
        tail -n "$LOG_TAIL_LINES" "$log_file" | sed 's/^/       /'
    else
        echo "     Log is empty."
    fi
    FAILED=1
}

# -----------------------------------------------------------------------------
# Tier 0 — Static
# -----------------------------------------------------------------------------
echo ""
echo "Tier 0 — Static"
if [ "$KIND" = "go" ]; then
    run_step "go build ${SCOPE[*]}" go build "${SCOPE[@]}"
    if command -v golangci-lint >/dev/null 2>&1; then
        run_step "golangci-lint run ${SCOPE[*]}" golangci-lint run "${SCOPE[@]}" --timeout=2m
    else
        echo "  ⚠️  golangci-lint not installed, skipped"
    fi
else
    run_step "pnpm type-check" "${PNPM[@]}" type-check
    run_step "pnpm lint" "${PNPM[@]}" lint
fi

if [ "$TIER" -eq 0 ]; then
    echo ""
    [ $FAILED -eq 0 ] && echo "✅ Tier 0 passed" || echo "❌ Tier 0 failed"
    [ $FAILED -eq 0 ] || echo "Full command logs: $LOG_DIR"
    exit $FAILED
fi

# -----------------------------------------------------------------------------
# Tier 1 — Local execution (tests)
# -----------------------------------------------------------------------------
echo ""
echo "Tier 1 — Local execution"
if [ "$KIND" = "go" ]; then
    run_step "go test ${SCOPE[*]}" go test "${SCOPE[@]}"
else
    run_step "pnpm test" "${PNPM[@]}" test -- --run
fi

if [ "$TIER" -eq 1 ]; then
    echo ""
    [ $FAILED -eq 0 ] && echo "✅ Tier 0 + 1 passed" || echo "❌ Failures above"
    [ $FAILED -eq 0 ] || echo "Full command logs: $LOG_DIR"
    exit $FAILED
fi

# -----------------------------------------------------------------------------
# Tier 2 — Build + race detector / full build
# -----------------------------------------------------------------------------
echo ""
echo "Tier 2 — End-to-end"
if [ "$KIND" = "go" ]; then
    run_step "go test -race ${SCOPE[*]}" go test -race "${SCOPE[@]}"
else
    run_step "pnpm build" "${PNPM[@]}" build
fi

echo ""
[ $FAILED -eq 0 ] && echo "✅ All tiers (0+1+2) passed" || echo "❌ Failures above"
[ $FAILED -eq 0 ] || echo "Full command logs: $LOG_DIR"
exit $FAILED

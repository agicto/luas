#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'USAGE'
Usage:
  check-downstream-contamination.sh [--expected-origin URL] [--ignore-case] --pattern TEXT [--pattern TEXT...]

Scans the current git repository for task-specific downstream product leakage.
Patterns are fixed strings. The script intentionally ships with no baked-in product names.

Examples:
  check-downstream-contamination.sh --expected-origin git@github.com:agicto/luas.git --pattern "product-name"
  check-downstream-contamination.sh --ignore-case --pattern "legacy-brand" --pattern "deployment-job"
USAGE
}

EXPECTED_ORIGIN=""
IGNORE_CASE=0
PATTERNS=()

while [ "$#" -gt 0 ]; do
    case "$1" in
        --expected-origin)
            if [ "$#" -lt 2 ]; then
                echo "Missing value for --expected-origin" >&2
                exit 2
            fi
            EXPECTED_ORIGIN=$2
            shift 2
            ;;
        --pattern)
            if [ "$#" -lt 2 ]; then
                echo "Missing value for --pattern" >&2
                exit 2
            fi
            PATTERNS+=("$2")
            shift 2
            ;;
        --ignore-case|-i)
            IGNORE_CASE=1
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            usage >&2
            exit 2
            ;;
    esac
done

if [ -z "$EXPECTED_ORIGIN" ] && [ "${#PATTERNS[@]}" -eq 0 ]; then
    echo "Provide --expected-origin, at least one --pattern, or both." >&2
    usage >&2
    exit 2
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
cd "$REPO_ROOT"

if [ -n "$EXPECTED_ORIGIN" ]; then
    ACTUAL_ORIGIN=$(git remote get-url origin 2>/dev/null || true)
    if [ "$ACTUAL_ORIGIN" != "$EXPECTED_ORIGIN" ]; then
        echo "Origin mismatch:" >&2
        echo "  expected: $EXPECTED_ORIGIN" >&2
        echo "  actual:   ${ACTUAL_ORIGIN:-<none>}" >&2
        exit 1
    fi
fi

if [ "${#PATTERNS[@]}" -eq 0 ]; then
    echo "Downstream contamination check passed."
    exit 0
fi

TMP_OUTPUT=$(mktemp)
trap 'rm -f "$TMP_OUTPUT"' EXIT

RG_FLAGS=(
    --hidden
    --line-number
    --fixed-strings
    --glob '!.git'
    --glob '!node_modules'
    --glob '!web/node_modules'
    --glob '!web/.next'
    --glob '!admin/node_modules'
    --glob '!admin/dist'
    --glob '!api/tmp'
    --glob '!dist'
    --glob '!coverage'
)

if [ "$IGNORE_CASE" -eq 1 ]; then
    RG_FLAGS+=(--ignore-case)
fi

FOUND=0
for PATTERN in "${PATTERNS[@]}"; do
    if [ -z "$PATTERN" ]; then
        continue
    fi

    if rg "${RG_FLAGS[@]}" -- "$PATTERN" . >>"$TMP_OUTPUT"; then
        FOUND=1
    fi
done

if [ "$FOUND" -eq 1 ]; then
    echo "Potential downstream product leakage found:" >&2
    cat "$TMP_OUTPUT" >&2
    exit 1
fi

echo "Downstream contamination check passed."

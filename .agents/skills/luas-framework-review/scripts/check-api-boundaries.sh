#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
API_ROOT="$ROOT/api"

cd "$API_ROOT"

MODULE=$(go list -m)

KNOWN_VIOLATIONS=()

violations=()

append_violation() {
  violations+=("$1 imports $2 [$3]")
}

scan_imports() {
  local package_pattern=$1
  local rule=$2

  for pkg in $(go list "$package_pattern"); do
    while IFS= read -r imported; do
      if [ -z "$imported" ]; then
        continue
      fi

      case "$rule" in
        pkg)
          case "$imported" in
            "$MODULE/internal/"*) append_violation "$pkg" "$imported" "pkg must not import internal" ;;
          esac
          ;;
        capabilities)
          case "$imported" in
            "$MODULE/internal/infra/"*|"$MODULE/internal/modules/"*)
              append_violation "$pkg" "$imported" "capabilities must not import infra/modules"
              ;;
          esac
          ;;
        infra)
          case "$imported" in
            "$MODULE/internal/modules/"*) append_violation "$pkg" "$imported" "infra must not import modules" ;;
          esac
          ;;
        *)
          echo "unknown rule: $rule" >&2
          exit 2
          ;;
      esac
    done < <(go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' "$pkg")
  done
}

is_known_violation() {
  local candidate=$1

  if [ "${#KNOWN_VIOLATIONS[@]}" -gt 0 ]; then
    for known in "${KNOWN_VIOLATIONS[@]}"; do
      if [ "$candidate" = "$known" ]; then
        return 0
      fi
    done
  fi

  return 1
}

scan_imports ./pkg/... pkg
scan_imports ./internal/capabilities/... capabilities
scan_imports ./internal/infra/... infra

new_violations=()
known_count=0
new_violation_count=0

if [ "${#violations[@]}" -gt 0 ]; then
  for violation in "${violations[@]}"; do
    if is_known_violation "$violation"; then
      known_count=$((known_count + 1))
    else
      new_violations+=("$violation")
      new_violation_count=$((new_violation_count + 1))
    fi
  done
fi

if [ "$new_violation_count" -gt 0 ]; then
  printf 'New API package boundary violation(s):\n' >&2
  for violation in "${new_violations[@]}"; do
    printf '  - %s\n' "$violation" >&2
  done
  echo "See api/docs/PACKAGE_BOUNDARIES.md and api/docs/adr/0005-package-boundaries.md." >&2
  exit 1
fi

echo "API package boundary check passed (${known_count} known baseline exception(s), no new violations)."

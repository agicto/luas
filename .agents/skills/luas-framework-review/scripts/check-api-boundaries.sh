#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
API_ROOT="$ROOT/api"

cd "$API_ROOT"

if [ -n "${CODEX_SANDBOX:-}" ] && [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="${TMPDIR:-/tmp}/luas-go-build-cache"
  mkdir -p "$GOCACHE"
fi

if ! MODULE=$(go list -m); then
  echo "Unable to resolve the API module; package boundaries were not checked." >&2
  exit 1
fi

assert_absent_path() {
  local relative_path=$1
  local message=$2

  if [ -e "$API_ROOT/$relative_path" ]; then
    echo "api/$relative_path is reserved against reuse; $message" >&2
    exit 1
  fi
}

assert_absent_path "internal/contracts" "use internal/starter/assembly for starter registry seams and root contracts/ for HTTP contracts."
# Generic grab-bag packages hide ownership; helpers belong at the seam that owns the behavior.
for grab_bag in pkg/support pkg/utils pkg/common pkg/helpers internal/infra/utils internal/infra/common; do
  assert_absent_path "$grab_bag" "generic helper packages hide ownership; put the helper at the capability, runtime, or starter seam that owns it."
done
assert_absent_path "internal/infra/contracts" "the term contract is reserved for HTTP contracts under root contracts/; name Go interfaces for their seam."

KNOWN_VIOLATIONS=()

violations=()

append_violation() {
  violations+=("$1 imports $2 [$3]")
}

scan_imports() {
  local package_pattern=$1
  local rule=$2
  local listing

  if ! listing=$(go list -f '{{.ImportPath}}{{range .Imports}}{{"\t"}}{{.}}{{end}}' "$package_pattern"); then
    echo "Unable to list $package_pattern; package boundaries were not checked." >&2
    exit 1
  fi

  while IFS=$'\t' read -r -a fields; do
    local pkg=${fields[0]}
    local index
    for ((index = 1; index < ${#fields[@]}; index++)); do
      local imported=${fields[$index]}

      case "$rule" in
        pkg)
          case "$imported" in
            "$MODULE/internal/"*) append_violation "$pkg" "$imported" "pkg must not import internal" ;;
          esac
          ;;
        domain)
          case "$imported" in
            "$MODULE/pkg/"*|"$MODULE/internal/"*)
              append_violation "$pkg" "$imported" "domain must not import pkg/internal"
              ;;
          esac
          ;;
        capabilities)
          case "$imported" in
            "$MODULE/internal/domain"*|"$MODULE/internal/infra/"*|"$MODULE/internal/modules/"*)
              append_violation "$pkg" "$imported" "capabilities must not import domain/infra/modules"
              ;;
          esac
          ;;
        infra)
          case "$imported" in
            "$MODULE/internal/domain"*|"$MODULE/internal/modules/"*)
              append_violation "$pkg" "$imported" "infra must not import domain/modules"
              ;;
          esac
          ;;
        *)
          echo "unknown rule: $rule" >&2
          exit 2
          ;;
      esac
    done
  done <<<"$listing"
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
scan_imports ./internal/domain domain
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

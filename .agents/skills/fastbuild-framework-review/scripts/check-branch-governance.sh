#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
cd "$ROOT"

DOC="docs/BRANCHING_AND_RELEASES.md"
FAILED=0

fail() {
  printf 'Branch governance drift: %s\n' "$1" >&2
  FAILED=1
}

require_fixed() {
  local file="$1"
  local text="$2"

  if ! grep -Fq -- "$text" "$file"; then
    fail "$file must contain: $text"
  fi
}

if [ ! -f "$DOC" ]; then
  fail "missing required file: $DOC"
  exit 1
fi

for branch in main dev release hotfix feature; do
  require_fixed "$DOC" "\`$branch"
done

require_fixed "$DOC" "Never merge \`dev\` back to \`main\`."
require_fixed "$DOC" "Create \`release/*\` from \`main\`, not from \`dev\`."
require_fixed "$DOC" "Merging \`dev\` into \`main\`."

# Validation workflows never push; only release.yml elevates, and only to publish a release.
for workflow in .github/workflows/*.yml; do
  [ "$(basename "$workflow")" = release.yml ] && continue
  if grep -Eq 'git push|contents:[[:space:]]*write' "$workflow"; then
    fail "$workflow must not write to the repository"
  fi
done

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "Branch governance check passed."

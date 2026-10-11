#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
cd "$ROOT"

DOC="docs/BRANCHING_AND_RELEASES.md"
WORKFLOW=".github/workflows/sync-deploy-branches.yml"
FAILED=0

fail() {
  printf 'Branch governance drift: %s\n' "$1" >&2
  FAILED=1
}

require_file() {
  local file="$1"

  if [ ! -f "$file" ]; then
    fail "missing required file: $file"
  fi
}

require_fixed() {
  local file="$1"
  local text="$2"

  if ! grep -Fq -- "$text" "$file"; then
    fail "$file must contain: $text"
  fi
}

require_regex() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if ! grep -Eq -- "$pattern" "$file"; then
    fail "$file must match: $description"
  fi
}

require_file "$DOC"
require_file "$WORKFLOW"

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

for branch in main dev dev-c release deploy-dev deploy-dev-c; do
  require_fixed "$DOC" "\`$branch"
done

require_fixed "$DOC" "Never merge \`dev\` back to \`main\`."
require_fixed "$DOC" "Never merge \`dev-c\` back to \`main\`."
require_fixed "$DOC" "Create \`release/*\` from \`main\`, not from \`dev\`."
require_fixed "$DOC" "\`deploy-dev\` and \`deploy-dev-c\` are deployment triggers, not collaboration branches."
require_fixed "$DOC" "Merging \`dev\` into \`main\`."

require_regex "$WORKFLOW" '^[[:space:]]*-[[:space:]]dev$' "dev is an allowed sync source"
require_regex "$WORKFLOW" '^[[:space:]]*-[[:space:]]dev-c$' "dev-c is an allowed sync source"
require_fixed "$WORKFLOW" 'deploy_branch="deploy-dev"'
require_fixed "$WORKFLOW" 'deploy_branch="deploy-dev-c"'
require_fixed "$WORKFLOW" 'git checkout -B "$deploy_branch" "origin/$source_branch"'
require_fixed "$WORKFLOW" 'git push --force-with-lease='

if grep -Eq '^[[:space:]]*-[[:space:]](main|deploy-dev|deploy-dev-c)$' "$WORKFLOW"; then
  fail "$WORKFLOW must not sync from main or deployment trigger branches"
fi

if grep -Eq '^[[:space:]]*(main|deploy-dev|deploy-dev-c|release/[^)]*)\)' "$WORKFLOW"; then
  fail "$WORKFLOW mapping must keep only dev and dev-c as source branch cases"
fi

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "Branch governance check passed."

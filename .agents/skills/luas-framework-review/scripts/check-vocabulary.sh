#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
cd "$ROOT"

FILES=(
  AGENTS.md
  README.md
  .agents/skills/README.md
  docs/ARCHITECTURE.md
  docs/BRANCHING_AND_RELEASES.md
  docs/FRAMEWORK_QUALITY_ROADMAP.md
  docs/SCAFFOLD_SURFACES.md
  docs/STARTER_BUSINESS_ROADMAP.md
  docs/SKILL_GOVERNANCE_PLAN.md
  contracts/README.md
  api/AGENTS.md
  api/docs/ADDING_MODULE.md
  api/docs/MIDDLEWARE.md
  api/docs/PACKAGE_BOUNDARIES.md
  api/docs/adr/0001-layer-vocabulary.md
  api/docs/adr/0002-default-starters.md
  api/docs/adr/0005-package-boundaries.md
  api/.agents/skills/README.md
  api/internal/capabilities/README.md
  api/internal/domain/README.md
  api/internal/modules/README.md
  web/AGENTS.md
  web/README.md
  web/docs/ADDING_FEATURE.md
  web/docs/MOCK_BFF.md
  web-spa/AGENTS.md
  web-spa/README.md
  web-spa/docs/ADDING_FEATURE.md
  web-spa/docs/ARCHITECTURE.md
  web-spa/docs/DEPLOYMENT.md
  web-spa/docs/SECURITY.md
)

while IFS= read -r skill_file; do
  FILES+=("$skill_file")
done < <(
  find .agents api/.agents web/.agents \
    -path '*/.template/*' -prune -o \
    -type f -name 'SKILL.md' -print | sort
)

FORBIDDEN_PATTERNS=(
  "Mock API"
  "Mock Backend"
  "mock API"
  "mock backend"
  "API Route Handlers (Mock endpoints)"
  "Feature-first modules"
  "feature-first modules"
  "src/features/[module]"
  "Adding a New Data Module"
  "Luas framework"
  "business module"
  "business modules"
  "capability module"
  "capability modules"
  "internal/contracts"
  "product-specific modules"
  "product dashboard"
)

PUBLIC_README_FORBIDDEN_PATTERNS=(
  "This repo merges two previous projects"
  "zgiai/zgo"
  "zgiai/zweb"
  "LlamaFront"
  "Hypership"
  "inherited from both source repos"
)

FAILED=0
TMP_FILE=$(mktemp "${TMPDIR:-/tmp}/luas-vocabulary-grep.XXXXXX")
trap 'rm -f "$TMP_FILE"' EXIT

EXISTING_FILES=()
for file in "${FILES[@]}"; do
  [ ! -f "$file" ] || EXISTING_FILES+=("$file")
done

FORBIDDEN_ARGS=()
for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
  FORBIDDEN_ARGS+=(-e "$pattern")
done

if grep -nF "${FORBIDDEN_ARGS[@]}" "${EXISTING_FILES[@]}" >"$TMP_FILE"; then
  while IFS= read -r match; do
    printf 'Vocabulary drift: %s\n' "$match"
  done <"$TMP_FILE"
  FAILED=1
fi

PUBLIC_README_ARGS=()
for pattern in "${PUBLIC_README_FORBIDDEN_PATTERNS[@]}"; do
  PUBLIC_README_ARGS+=(-e "$pattern")
done

if grep -nF "${PUBLIC_README_ARGS[@]}" README.md >"$TMP_FILE"; then
  while IFS= read -r match; do
    printf 'Public README origin drift: README.md:%s\n' "$match"
  done <"$TMP_FILE"
  FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
  echo "Use CONTEXT.md vocabulary and present Luas as the independent open-source scaffold." >&2
  exit 1
fi

echo "Vocabulary check passed."

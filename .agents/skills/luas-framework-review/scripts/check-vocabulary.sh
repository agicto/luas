#!/usr/bin/env bash

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
cd "$ROOT"

FILES=(
  AGENTS.md
  README.md
  docs/ARCHITECTURE.md
  docs/BRANCHING_AND_RELEASES.md
  docs/FRAMEWORK_QUALITY_ROADMAP.md
  contracts/README.md
  api/AGENTS.md
  api/docs/ADDING_MODULE.md
  api/docs/adr/0001-layer-vocabulary.md
  api/docs/adr/0002-default-starters.md
  api/.agents/skills/README.md
  api/.agents/skills/architecture-principles/SKILL.md
  api/.agents/skills/coding-standards/SKILL.md
  api/.agents/skills/module-creation/SKILL.md
  web/AGENTS.md
  web/README.md
  web/docs/ADDING_FEATURE.md
  web/docs/MOCK_BFF.md
  .agents/skills/luas-framework-review/SKILL.md
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
  "product-specific modules"
  "product dashboard"
)

FAILED=0

for file in "${FILES[@]}"; do
  if [ ! -f "$file" ]; then
    continue
  fi

  for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
    if grep -nF "$pattern" "$file" >/tmp/luas-vocabulary-grep.txt; then
      while IFS= read -r match; do
        printf 'Vocabulary drift: %s:%s (%s)\n' "$file" "$match" "$pattern"
      done </tmp/luas-vocabulary-grep.txt
      FAILED=1
    fi
  done
done

if [ "$FAILED" -ne 0 ]; then
  echo "Use CONTEXT.md vocabulary: scaffold/starter/feature/module/mock BFF/API/console/error_code." >&2
  exit 1
fi

echo "Vocabulary check passed."

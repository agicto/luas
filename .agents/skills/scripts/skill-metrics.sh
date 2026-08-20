#!/usr/bin/env bash

# Print deterministic repository-skill context and invocation metrics.

set -eu
export LC_ALL=C

REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}
TOTAL=0
IMPLICIT=0
EXPLICIT=0
TOTAL_BYTES=0
MAX_BYTES=0
MAX_SKILL=""
DESCRIPTION_BYTES=0

while IFS= read -r skill_file; do
    TOTAL=$((TOTAL + 1))
    skill_dir=${skill_file%/SKILL.md}
    metadata="$skill_dir/agents/openai.yaml"
    name=$(sed -n 's/^name: //p' "$skill_file" | head -n 1)
    description=$(sed -n 's/^description: //p' "$skill_file" | head -n 1)
    bytes=$(wc -c < "$skill_file" | tr -d ' ')

    TOTAL_BYTES=$((TOTAL_BYTES + bytes))
    DESCRIPTION_BYTES=$((DESCRIPTION_BYTES + ${#description}))
    if [ "$bytes" -gt "$MAX_BYTES" ]; then
        MAX_BYTES=$bytes
        MAX_SKILL=$name
    fi

    if [ -f "$metadata" ] &&
       grep -q '^  allow_implicit_invocation: false$' "$metadata"; then
        EXPLICIT=$((EXPLICIT + 1))
    else
        IMPLICIT=$((IMPLICIT + 1))
    fi
done < <(
    find \
        "$REPO_ROOT/.agents/skills" \
        "$REPO_ROOT/api/.agents/skills" \
        "$REPO_ROOT/web/.agents/skills" \
        -name SKILL.md -not -path '*/.template/*' 2>/dev/null |
        sort
)

printf 'Repository skills: %s\n' "$TOTAL"
printf 'Implicitly invokable: %s\n' "$IMPLICIT"
printf 'Explicit-only: %s\n' "$EXPLICIT"
printf 'SKILL.md bytes: %s\n' "$TOTAL_BYTES"
printf 'Description bytes: %s\n' "$DESCRIPTION_BYTES"
printf 'Largest SKILL.md: %s (%s bytes)\n' "$MAX_SKILL" "$MAX_BYTES"

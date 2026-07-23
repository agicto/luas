#!/usr/bin/env bash

# Scaffold PR Body
#
# Generates a draft PR description from git log + git diff against the
# base branch. Output follows the structure documented in ../SKILL.md
# (Summary / What Changed / Test Plan / Risk & Rollback). Fill in the
# Motivation and Risk sections by hand — those require human judgment.
#
# Usage:
#   scaffold-pr-body.sh                      # base = main
#   scaffold-pr-body.sh dev                  # base = dev
#   scaffold-pr-body.sh main > /tmp/pr.md    # write to file

set -u

BASE=${1:-main}

if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Not inside a git repository" >&2
    exit 2
fi

if ! git rev-parse --verify "$BASE" > /dev/null 2>&1; then
    echo "❌ Base branch '$BASE' does not exist" >&2
    exit 2
fi

HEAD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT_COUNT=$(git rev-list --count "$BASE..HEAD")
if [ "$COMMIT_COUNT" -eq 0 ]; then
    echo "❌ No commits between $BASE and HEAD ($HEAD_BRANCH)" >&2
    exit 1
fi

# Files touched, grouped by top-level subtree
FILES_BY_AREA=$(git diff --name-only "$BASE...HEAD" | awk -F/ '{print $1}' | sort -u)

cat <<EOF
## Summary

<!-- 1–3 bullets. What changed, in plain language. -->
- TODO

## Motivation

<!-- Why this exists. Link issue/incident. State the problem before the solution. -->
TODO

## What Changed

<!-- Concrete list of files / modules / behaviors touched. Generated below; trim freely. -->

EOF

for area in $FILES_BY_AREA; do
    COUNT=$(git diff --name-only "$BASE...HEAD" -- "$area" | wc -l | tr -d ' ')
    echo "- **$area** ($COUNT file$([ "$COUNT" -ne 1 ] && echo s)):"
    git diff --name-only "$BASE...HEAD" -- "$area" | head -10 | sed 's/^/    - /'
    if [ "$COUNT" -gt 10 ]; then
        echo "    - …and $((COUNT - 10)) more"
    fi
done

cat <<EOF

## Test Plan

<!-- Use [x] for what you ran; [ ] for what the reviewer should run. -->
- [ ] Unit: \`go test ./<changed-pkg>/...\` (or \`pnpm test\`)
- [ ] API: \`cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./<changed-pkg>/...\`
- [ ] Lint: \`pnpm type-check && pnpm lint\` (web)
- [ ] Manual: TODO — what should the reviewer click through?

## Risk & Rollback

<!-- State the blast radius and the path back. -->
- **Blast radius**: TODO (single endpoint / single page / whole module / all users)
- **Rollback**: TODO (revert commit / feature flag off / DB rollback migration)

---

<details>
<summary>Commits ($COMMIT_COUNT)</summary>

EOF

git log --pretty=format:'- %h %s' "$BASE..HEAD"
echo ""
echo ""
echo "</details>"

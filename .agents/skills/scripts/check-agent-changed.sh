#!/usr/bin/env bash

# Fast agent-guidance checks for files changed from a feature-branch base plus
# staged, unstaged, and untracked work. Deletions, renames, and checker changes
# fall back to the complete agent check because local proof is insufficient.

set -euo pipefail
export LC_ALL=C

REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}
BASE_REF=${AGENT_CHECK_BASE_REF:-main}
cd "$REPO_ROOT"

VOCABULARY_CHECK=.agents/skills/luas-framework-review/scripts/check-vocabulary.sh
LINK_CHECK=.agents/skills/luas-framework-review/scripts/check-doc-links.py
ENGLISH_CHECK=.agents/skills/luas-framework-review/scripts/check-english-source.py
SKILL_CHECK=.agents/skills/scripts/validate-skill.sh

CHANGED_FILE=$(mktemp "${TMPDIR:-/tmp}/luas-agent-changed.XXXXXX")
UNTRACKED_FILE=$(mktemp "${TMPDIR:-/tmp}/luas-agent-untracked.XXXXXX")
trap 'rm -f "$CHANGED_FILE" "$UNTRACKED_FILE"' EXIT

FULL_REASON=""

collect_git_changes() {
    if ! git rev-parse --verify --quiet "$BASE_REF^{commit}" >/dev/null; then
        echo "Unknown AGENT_CHECK_BASE_REF: $BASE_REF" >&2
        exit 2
    fi

    git diff --name-only --diff-filter=ACMR "$BASE_REF"...HEAD
    git diff --cached --name-only --diff-filter=ACMR
    git diff --name-only --diff-filter=ACMR
    git ls-files --others --exclude-standard | tee "$UNTRACKED_FILE"

    if git diff --name-only --diff-filter=DR "$BASE_REF"...HEAD | grep -q . ||
       git diff --cached --name-only --diff-filter=DR | grep -q . ||
       git diff --name-only --diff-filter=DR | grep -q .; then
        FULL_REASON="a deletion or rename can break unchanged callers"
    fi
}

if [ "$#" -gt 0 ]; then
    printf '%s\n' "$@" >"$CHANGED_FILE"
    for path in "$@"; do
        if [ -f "$path" ] && ! git ls-files --error-unmatch -- "$path" >/dev/null 2>&1; then
            printf '%s\n' "$path" >>"$UNTRACKED_FILE"
        fi
    done
else
    collect_git_changes >"$CHANGED_FILE"
fi
sort -u "$CHANGED_FILE" -o "$CHANGED_FILE"

while IFS= read -r path; do
    case "$path" in
        Makefile|\
        "$VOCABULARY_CHECK"|\
        "$LINK_CHECK"|\
        "$ENGLISH_CHECK"|\
        "$SKILL_CHECK"|\
        .agents/skills/scripts/check-agent-changed.sh)
            FULL_REASON="an agent-check implementation file changed"
            ;;
    esac
done <"$CHANGED_FILE"

run_whitespace_checks() {
    git diff --check "$BASE_REF"...HEAD
    git diff --cached --check
    git diff --check

    while IFS= read -r path; do
        [ -f "$path" ] || continue
        local output=""
        output=$(git diff --no-index --check /dev/null "$path" 2>&1 || true)
        if [ -n "$output" ]; then
            printf '%s\n' "$output" >&2
            return 1
        fi
    done <"$UNTRACKED_FILE"
}

if [ -n "$FULL_REASON" ]; then
    echo "Changed agent check: using complete scan because $FULL_REASON."
    bash "$VOCABULARY_CHECK"
    PYTHONDONTWRITEBYTECODE=1 python3 "$LINK_CHECK"
    PYTHONDONTWRITEBYTECODE=1 python3 "$ENGLISH_CHECK"
    bash "$SKILL_CHECK" --all
    run_whitespace_checks
    exit 0
fi

if [ ! -s "$CHANGED_FILE" ]; then
    echo "Changed agent check passed (no changed files from $BASE_REF)."
    exit 0
fi

EXISTING_FILES=()
MARKDOWN_FILES=()
while IFS= read -r path; do
    [ -f "$path" ] || continue
    EXISTING_FILES+=("$path")
    case "$path" in
        *.md) MARKDOWN_FILES+=("$path") ;;
    esac
done <"$CHANGED_FILE"

bash "$VOCABULARY_CHECK"
if [ ${#MARKDOWN_FILES[@]} -gt 0 ]; then
    PYTHONDONTWRITEBYTECODE=1 python3 "$LINK_CHECK" "${MARKDOWN_FILES[@]}"
else
    echo "Markdown link check passed (0 changed files scanned)."
fi
if [ ${#EXISTING_FILES[@]} -gt 0 ]; then
    PYTHONDONTWRITEBYTECODE=1 python3 "$ENGLISH_CHECK" "${EXISTING_FILES[@]}"
else
    echo "English source check passed (0 changed files scanned)."
fi
bash "$SKILL_CHECK" --all
run_whitespace_checks

printf 'Changed agent check passed (%s files from %s).\n' \
    "$(wc -l < "$CHANGED_FILE" | tr -d ' ')" "$BASE_REF"

#!/usr/bin/env bash

# Validate SKILL.md metadata against codex CLI's loader requirements.
#
# Usage:
#   validate-skill.sh <path/to/SKILL.md>
#   validate-skill.sh --all          # validate every SKILL.md in the repo
#
# Checks (codex's core-skills/loader.rs constants):
#   - Frontmatter wrapped in --- ... --- (MissingFrontmatter is hard fail)
#   - `name:` field present, ≤ 64 chars
#   - `description:` field present, ≤ 1024 bytes
#   - Directory name matches `name:` (warn — not enforced by codex)
#   - File name is exactly SKILL.md (case sensitive — hard fail otherwise)
#
# Exit code: 0 on clean, 1 if any hard fail.

set -u

MAX_NAME=64
MAX_DESC=1024
MAX_DESC_PRACTICAL=200  # warn above this (codex injects all descs in one prompt)

ERRORS=0
WARNINGS=0
FILES=0

err()  { echo "  ❌ $1"; ERRORS=$((ERRORS + 1)); }
warn() { echo "  ⚠️  $1"; WARNINGS=$((WARNINGS + 1)); }
ok()   { :; }  # silent on per-check success to keep the report short

field() {
    awk -v k="$1" '
        /^---$/ { state++; next }
        state == 1 && index($0, k ":") == 1 {
            sub("^" k ":[[:space:]]*", "")
            print
            exit
        }
    ' "$2"
}

validate_one() {
    local file=$1
    FILES=$((FILES + 1))
    local pre=$ERRORS

    echo ""
    echo "📄 $file"

    # 1. Filename
    if [ "$(basename "$file")" != "SKILL.md" ]; then
        err "Filename must be exactly 'SKILL.md' (case sensitive)"
        return
    fi

    # 2. Frontmatter delimiters
    if ! head -1 "$file" | grep -q '^---$'; then
        err "First line must be '---' (no frontmatter)"
        return
    fi
    if ! awk 'NR > 1 && /^---$/ { found=1; exit } END { exit !found }' "$file"; then
        err "No closing '---' delimiter found for frontmatter"
        return
    fi

    # 3. name
    local name
    name=$(field name "$file")
    if [ -z "$name" ]; then
        err "Missing 'name:' field in frontmatter"
    elif [ ${#name} -gt $MAX_NAME ]; then
        err "name is ${#name} chars; codex max is $MAX_NAME"
    fi

    # 4. description
    local desc
    desc=$(field description "$file")
    local desc_bytes=${#desc}
    if [ -z "$desc" ]; then
        err "Missing 'description:' field in frontmatter"
    elif [ $desc_bytes -gt $MAX_DESC ]; then
        err "description is $desc_bytes bytes; codex max is $MAX_DESC"
    elif [ $desc_bytes -gt $MAX_DESC_PRACTICAL ]; then
        warn "description is $desc_bytes bytes; recommended ≤ $MAX_DESC_PRACTICAL to stay within the 2% context budget"
    fi

    # 5. Directory name matches name (informational only)
    local dir_name
    dir_name=$(basename "$(dirname "$file")")
    if [ -n "$name" ] && [ "$dir_name" != "$name" ]; then
        warn "directory '$dir_name' differs from frontmatter name '$name' (codex uses frontmatter)"
    fi

    if [ $ERRORS -eq $pre ]; then
        echo "  ✅ ok ($desc_bytes byte description)"
    fi
}

if [ "${1:-}" = "--all" ]; then
    REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}
    while IFS= read -r f; do
        validate_one "$f"
    done < <(find "$REPO_ROOT/.agents/skills" "$REPO_ROOT/api/.agents/skills" "$REPO_ROOT/web/.agents/skills" \
        -name SKILL.md -not -path '*/.template/*' 2>/dev/null | sort)
elif [ -n "${1:-}" ]; then
    validate_one "$1"
else
    echo "Usage: validate-skill.sh <SKILL.md | --all>" >&2
    exit 2
fi

echo ""
echo "================================================"
echo "Validated $FILES SKILL.md file(s)"
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✅ All clean"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠️  $WARNINGS warning(s)"
    exit 0
else
    echo "❌ $ERRORS error(s), $WARNINGS warning(s)"
    exit 1
fi

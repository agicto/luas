#!/usr/bin/env bash

# Validate repository skills against Codex discovery and context-budget rules.
#
# Usage:
#   validate-skill.sh <path/to/SKILL.md>
#   validate-skill.sh --all
#
# Set SKILL_VALIDATION_VERBOSE=1 to print every passing file.

set -u
export LC_ALL=C

MAX_NAME=64
MAX_DESC=1024
MAX_DESC_PRACTICAL=200
MAX_LINES=200
MAX_IMPLICIT_BYTES=6000
MAX_EXPLICIT_BYTES=7000
MIN_SHORT_DESC=25
MAX_SHORT_DESC=64

ERRORS=0
WARNINGS=0
FILES=0
SEEN_NAMES="|"
RESERVED_NAMES="|imagegen|openai-docs|plugin-creator|skill-creator|skill-installer|"
CURRENT_FILE=""
VERBOSE=${SKILL_VALIDATION_VERBOSE:-0}

err() {
    printf '%s: error: %s\n' "$CURRENT_FILE" "$1"
    ERRORS=$((ERRORS + 1))
}

warn() {
    printf '%s: warning: %s\n' "$CURRENT_FILE" "$1"
    WARNINGS=$((WARNINGS + 1))
}

validate_one() {
    CURRENT_FILE=$1
    FILES=$((FILES + 1))
    local before_errors=$ERRORS
    local before_warnings=$WARNINGS
    local first_line=""
    local line=""
    local line_count=0
    local section=0
    local name=""
    local description=""
    local extra_frontmatter=""
    local file_bytes=0

    file_bytes=$(wc -c < "$CURRENT_FILE" | tr -d ' ')

    local filename=${CURRENT_FILE##*/}
    if [ "$filename" != "SKILL.md" ]; then
        err "filename must be exactly SKILL.md"
        return
    fi

    while IFS= read -r line || [ -n "$line" ]; do
        line_count=$((line_count + 1))
        [ "$line_count" -ne 1 ] || first_line=$line

        if [ "$line" = "---" ]; then
            section=$((section + 1))
            continue
        fi

        if [ "$section" -eq 1 ]; then
            case "$line" in
                "name:"*)
                    name=${line#name:}
                    name=${name#"${name%%[![:space:]]*}"}
                    ;;
                "description:"*)
                    description=${line#description:}
                    description=${description#"${description%%[![:space:]]*}"}
                    ;;
                "")
                    ;;
                *)
                    [ -n "$extra_frontmatter" ] || extra_frontmatter=$line
                    ;;
            esac
        fi
    done < "$CURRENT_FILE"

    if [ "$first_line" != "---" ]; then
        err "missing opening YAML frontmatter delimiter"
        return
    fi

    if [ "$section" -lt 2 ]; then
        err "missing closing YAML frontmatter delimiter"
        return
    fi

    if [ -z "$name" ]; then
        err "missing name"
    elif [ ${#name} -gt $MAX_NAME ]; then
        err "name exceeds $MAX_NAME characters"
    elif [[ ! "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
        err "name must be kebab-case"
    else
        case "$RESERVED_NAMES" in
            *"|$name|"*) err "repository skill name '$name' collides with a Codex built-in" ;;
        esac
        case "$SEEN_NAMES" in
            *"|$name|"*) err "duplicate repository skill name '$name'" ;;
            *) SEEN_NAMES="${SEEN_NAMES}${name}|" ;;
        esac
    fi

    local description_bytes=${#description}
    if [ -z "$description" ]; then
        err "missing description"
    elif [ "$description_bytes" -gt $MAX_DESC ]; then
        err "description is $description_bytes bytes; maximum is $MAX_DESC"
    elif [ "$description_bytes" -gt $MAX_DESC_PRACTICAL ]; then
        warn "description is $description_bytes bytes; prefer at most $MAX_DESC_PRACTICAL"
    fi

    if [ "$line_count" -gt $MAX_LINES ]; then
        err "SKILL.md is $line_count lines; split optional detail and keep at most $MAX_LINES"
    fi

    local parent=${CURRENT_FILE%/*}
    local directory=${parent##*/}
    if [ -n "$name" ] && [ "$directory" != "$name" ]; then
        warn "directory '$directory' differs from skill name '$name'"
    fi

    if [ -n "$extra_frontmatter" ]; then
        warn "move optional UI/policy metadata to agents/openai.yaml"
    fi

    local metadata="$parent/agents/openai.yaml"
    if [ ! -f "$metadata" ]; then
        err "missing agents/openai.yaml"
    else
        local display_name=""
        local short_description=""
        local default_prompt=""
        local invocation_policy=""

        display_name=$(sed -n 's/^  display_name: "\(.*\)"$/\1/p' "$metadata")
        short_description=$(sed -n 's/^  short_description: "\(.*\)"$/\1/p' "$metadata")
        default_prompt=$(sed -n 's/^  default_prompt: "\(.*\)"$/\1/p' "$metadata")
        invocation_policy=$(sed -n 's/^  allow_implicit_invocation: \(.*\)$/\1/p' "$metadata")

        [ -n "$display_name" ] || err "agents/openai.yaml needs a quoted interface.display_name"
        [ -n "$short_description" ] || err "agents/openai.yaml needs a quoted interface.short_description"
        [ -n "$default_prompt" ] || err "agents/openai.yaml needs a quoted interface.default_prompt"

        if [ -n "$short_description" ] &&
           { [ ${#short_description} -lt $MIN_SHORT_DESC ] ||
             [ ${#short_description} -gt $MAX_SHORT_DESC ]; }; then
            err "interface.short_description must be $MIN_SHORT_DESC-$MAX_SHORT_DESC characters"
        fi

        if [ -n "$default_prompt" ]; then
            case "$default_prompt" in
                *"\$$name"*) ;;
                *) err "interface.default_prompt must mention \$$name" ;;
            esac
        fi

        case "$invocation_policy" in
            ""|true|false) ;;
            *) err "policy.allow_implicit_invocation must be true or false" ;;
        esac

        if [ "$invocation_policy" = "false" ]; then
            if [ "$file_bytes" -gt "$MAX_EXPLICIT_BYTES" ]; then
                err "explicit-only SKILL.md is $file_bytes bytes; maximum is $MAX_EXPLICIT_BYTES"
            fi
        elif [ "$file_bytes" -gt "$MAX_IMPLICIT_BYTES" ]; then
            err "implicitly invokable SKILL.md is $file_bytes bytes; maximum is $MAX_IMPLICIT_BYTES"
        fi
    fi

    if [ "$VERBOSE" = "1" ] &&
       [ "$ERRORS" -eq "$before_errors" ] &&
       [ "$WARNINGS" -eq "$before_warnings" ]; then
        printf '%s: ok (%s lines, %s-byte description)\n' \
            "$CURRENT_FILE" "$line_count" "$description_bytes"
    fi
}

if [ "${1:-}" = "--all" ]; then
    REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}
    while IFS= read -r file; do
        validate_one "$file"
    done < <(
        find \
            "$REPO_ROOT/.agents/skills" \
            "$REPO_ROOT/api/.agents/skills" \
            "$REPO_ROOT/web/.agents/skills" \
            -name SKILL.md -not -path '*/.template/*' 2>/dev/null |
            sort
    )
elif [ -n "${1:-}" ]; then
    validate_one "$1"
else
    echo "Usage: validate-skill.sh <SKILL.md | --all>" >&2
    exit 2
fi

printf 'Skill validation: %s files, %s errors, %s warnings\n' \
    "$FILES" "$ERRORS" "$WARNINGS"

if [ "$ERRORS" -gt 0 ]; then
    exit 1
fi

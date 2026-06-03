#!/usr/bin/env bash

# List all skills the repo ships, with their scope, description, and
# whether they ship a scripts/ or examples/ directory.
#
# Run from the repo root or with REPO_ROOT set.

set -u

REPO_ROOT=${REPO_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}
cd "$REPO_ROOT"

# Pretty width settings
NAME_W=32
SCOPE_W=8
EXTRAS_W=12

bold()   { printf "\033[1m%s\033[0m" "$1"; }
dim()    { printf "\033[2m%s\033[0m" "$1"; }
yellow() { printf "\033[33m%s\033[0m" "$1"; }

# Extract a frontmatter field value from a SKILL.md (first occurrence).
field() {
    local file=$1 key=$2
    awk -v k="$key" '
        /^---$/ { state++; next }
        state == 1 && index($0, k ":") == 1 {
            sub("^" k ":[[:space:]]*", "")
            print
            exit
        }
    ' "$file"
}

# Truncate a string to width N, appending an ellipsis if cut.
truncate() {
    local s=$1 n=$2
    if [ ${#s} -le $n ]; then
        printf "%s" "$s"
    else
        printf "%s…" "${s:0:n-1}"
    fi
}

print_row() {
    local scope=$1 name=$2 desc=$3 extras=$4
    printf "  %-${SCOPE_W}s %-${NAME_W}s %-${EXTRAS_W}s %s\n" \
        "$scope" "$(truncate "$name" $NAME_W)" "$extras" \
        "$(truncate "$desc" $((COLUMNS - SCOPE_W - NAME_W - EXTRAS_W - 8)))"
}

COLUMNS=${COLUMNS:-$(tput cols 2>/dev/null || echo 120)}
[ "$COLUMNS" -lt 80 ] && COLUMNS=80

TOTAL=0
for_scope() {
    local scope=$1 dir=$2
    local header_printed=0
    while IFS= read -r skill_md; do
        TOTAL=$((TOTAL + 1))
        if [ $header_printed -eq 0 ]; then
            echo ""
            bold "$scope"
            echo " — $dir"
            header_printed=1
        fi
        local skill_dir
        skill_dir=$(dirname "$skill_md")
        local name desc extras
        name=$(field "$skill_md" name)
        desc=$(field "$skill_md" description)
        extras=""
        [ -d "$skill_dir/scripts" ] && extras="${extras}S"
        [ -d "$skill_dir/examples" ] && extras="${extras}E"
        [ -d "$skill_dir/references" ] && extras="${extras}R"
        [ -d "$skill_dir/templates" ] && extras="${extras}T"
        [ -d "$skill_dir/rules" ] && extras="${extras}r"
        [ -z "$extras" ] && extras=$(dim "—")
        print_row "$scope" "$name" "$desc" "$extras"
    done < <(find "$dir" -maxdepth 3 -name SKILL.md -not -path '*/.template/*' 2>/dev/null | sort)
}

echo "================================================"
bold "Luas Skill Index"
echo ""
echo "Legend: S=scripts/  E=examples/  R=references/  T=templates/  r=rules/"
echo ""
printf "  %-${SCOPE_W}s %-${NAME_W}s %-${EXTRAS_W}s %s\n" "scope" "name" "extras" "description"
printf "  %-${SCOPE_W}s %-${NAME_W}s %-${EXTRAS_W}s %s\n" "$(dim "-----")" "$(dim "----")" "$(dim "------")" "$(dim "-----------")"

for_scope "root" ".agents/skills"
for_scope "api"  "api/.agents/skills"
for_scope "web"  "web/.agents/skills"

echo ""
echo "================================================"
echo "Total: $TOTAL skill$([ "$TOTAL" -ne 1 ] && echo s)"

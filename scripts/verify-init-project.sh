#!/usr/bin/env bash
# Proves a project created with `make init-project` passes its own checks: copies this checkout,
# initializes it with every optional starter, and runs governance, the contract checks, and the API
# packages that know about the scaffold (registry/ is gone in a project). The checkout is unchanged.
#
#   scripts/verify-init-project.sh
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/fastbuild-init-XXXXXX")
trap 'rm -rf "$scratch"' EXIT

step() { printf '\n==> %s\n' "$*"; }

step "Copying tracked files to $scratch"
(cd "$repo" && git ls-files -z | xargs -0 -I{} rsync -R "{}" "$scratch/")
(cd "$scratch" && git init -q && git add -A && git -c user.name=ci -c user.email=ci@example.invalid commit -qm scaffold)
# Link dependencies after the commit so they are never tracked files of the copy.
for half in contracts web admin; do
  if [[ -d "$repo/$half/node_modules" ]]; then
    ln -s "$repo/$half/node_modules" "$scratch/$half/node_modules"
  fi
done

step "make init-project"
starters=$(cd "$scratch/api" && go run ./cmd/fastbuild starter:list --format=json |
  node -e 'const c = JSON.parse(require("fs").readFileSync(0, "utf8"));
    process.stdout.write(c.starters.filter(s => s.mode === "optional").map(s => s.name).join(","))')
(cd "$scratch" && make -s init-project NAME="Example Project" SLUG=example-project \
  MODULE=github.com/example-org/example-project/api STARTERS="$starters")
test ! -e "$scratch/registry" || { echo "registry/ must be removed" >&2; exit 1; }

step "Governance and contracts in the new project"
(cd "$scratch" && make -s governance && make -s contract-check)

step "API packages that reference the scaffold"
(cd "$scratch/api" && go build ./... && go test -count=1 ./internal/infra/... ./internal/starter/... ./pkg/...)

printf '\nInitialized project verified.\n'

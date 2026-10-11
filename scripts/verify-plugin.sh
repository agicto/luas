#!/usr/bin/env bash
# Proves one registry plugin end to end in a scratch copy of this checkout: install it, run the
# deterministic integration steps, build and test every half it touches, require a clean
# plugin:doctor, and remove it again. The checkout itself is never changed.
#
#   scripts/verify-plugin.sh <plugin>
#
# FASTBUILD_TEST_POSTGRES_DSN enables the plugin's PostgreSQL tests. contracts/, web/, and admin/
# need their dependencies installed (pnpm install) for the halves the plugin touches.
set -euo pipefail

name=${1:?usage: scripts/verify-plugin.sh <plugin>}
repo=$(cd "$(dirname "$0")/.." && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/fastbuild-verify-XXXXXX")
trap 'rm -rf "$scratch"' EXIT

step() { printf '\n==> %s\n' "$*"; }

step "Copying the checkout to $scratch"
rsync -a --exclude node_modules --exclude .git --exclude .next --exclude dist \
  --exclude .fastbuild-dev --exclude coverage "$repo/" "$scratch/"
for half in contracts web admin; do
  if [[ -d "$repo/$half/node_modules" ]]; then
    ln -s "$repo/$half/node_modules" "$scratch/$half/node_modules"
  fi
done

fastbuild() { (cd "$scratch/api" && go run ./cmd/fastbuild "$@"); }

info=$(fastbuild plugin:info "$name" --format=json)
field() {
  node -e 'const p = JSON.parse(require("fs").readFileSync(0, "utf8")).plugin;
    const files = Object.keys(p.files ?? {});
    const value = { kind: p.kind || "feature", starter: p.starter?.directory ?? "", fragment: p.fragment ?? "",
      api: files.some(f => f.startsWith("api/")), web: files.some(f => f.startsWith("web/")),
      admin: files.some(f => f.startsWith("admin/")) || (p.wiring ?? []).some(w => w.slot.startsWith("admin.")),
      webWiring: (p.wiring ?? []).some(w => w.slot.startsWith("web.")) }[process.argv[1]];
    process.stdout.write(String(value));' "$1" <<<"$info"
}
kind=$(field kind)
if [[ "$kind" == template ]]; then
  echo "$name is a template; verify each plugin it includes." >&2
  exit 2
fi

step "plugin:add $name ($kind)"
fastbuild plugin:add "$name"

if [[ -n "$(field fragment)" ]]; then
  step "Regenerating contract types"
  (cd "$scratch/contracts" && corepack pnpm -s generate && corepack pnpm -s check)
fi

if [[ "$(field api)" == true ]]; then
  starter=$(field starter)
  step "Building and testing the API"
  if [[ -n "$starter" ]]; then
    (cd "$scratch/api" && make -s wire)
  fi
  (cd "$scratch/api" && go build ./... && go vet ./...)
  if [[ -n "$starter" ]]; then
    (cd "$scratch/api" && go test -count=1 "./internal/modules/$starter/...")
  fi
fi

if [[ "$(field web)" == true || "$(field webWiring)" == true ]]; then
  step "Checking the Web application"
  (cd "$scratch/web" && corepack pnpm -s type-check && corepack pnpm -s lint && corepack pnpm -s vitest run)
fi

if [[ "$(field admin)" == true ]]; then
  step "Checking the Admin Console"
  (cd "$scratch/admin" && corepack pnpm -s type-check && corepack pnpm -s build)
fi

step "plugin:doctor"
fastbuild plugin:doctor

step "plugin:remove $name"
fastbuild plugin:remove "$name" --force

printf '\nPlugin %s verified.\n' "$name"

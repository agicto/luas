#!/usr/bin/env bash
# Verify that main is ready to be tagged as VERSION. Usage: scripts/release-check.sh vX.Y.Z
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
fail() { printf 'release check failed: %s\n' "$*" >&2; exit 1; }

[[ "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "VERSION must look like v0.22.0"
cd "${ROOT_DIR}"

[[ "$(git rev-parse --abbrev-ref HEAD)" == "main" ]] || fail "check out main"
[[ -z "$(git status --porcelain)" ]] || fail "the working tree is not clean"
git fetch --quiet origin main --tags
[[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/main)" ]] || fail "main differs from origin/main"
if git rev-parse --quiet --verify "refs/tags/${VERSION}" >/dev/null; then
  fail "tag ${VERSION} already exists"
fi

grep -qE "^## ${VERSION//./\\.} — [0-9]{4}-[0-9]{2}-[0-9]{2}$" CHANGELOG.md ||
  fail "CHANGELOG.md has no '## ${VERSION} — YYYY-MM-DD' section"
unreleased="$(awk '/^## Unreleased/{f=1;next} /^## /{f=0} f && NF' CHANGELOG.md)"
[[ -z "${unreleased}" ]] || fail "CHANGELOG.md still has notes under '## Unreleased'"

make check
printf 'Ready to tag %s: git tag -a %s -m "%s" && git push origin %s\n' \
  "${VERSION}" "${VERSION}" "${VERSION}" "${VERSION}"

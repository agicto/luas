#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPOSITORY_ROOT="$(git -C "${ROOT_DIR}" rev-parse --show-toplevel 2>/dev/null || dirname "${ROOT_DIR}")"
IMAGE_TAG="${1:-luas-web:container-check}"
CONTAINER_NAME="luas-web-container-check-$$"
TMP_DIR="$(mktemp -d)"
BUILD_METADATA_OUTPUT="${BUILD_METADATA_OUTPUT:-${TMP_DIR}/luas-web.build-metadata.json}"
OCI_SOURCE="${OCI_SOURCE:-https://github.com/zgiai/luas}"
OCI_REVISION="${OCI_REVISION:-$(git -C "${REPOSITORY_ROOT}" rev-parse HEAD 2>/dev/null || printf 'unknown')}"
OCI_VERSION="${OCI_VERSION:-$(git -C "${REPOSITORY_ROOT}" describe --tags --always 2>/dev/null || printf 'dev')}"

cleanup() {
  docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'web container verification failed: %s\n' "$1" >&2
  docker logs "${CONTAINER_NAME}" >&2 2>/dev/null || true
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

assert_label() {
  local key="$1" expected="$2" actual
  actual="$(docker image inspect "${IMAGE_TAG}" --format "{{ index .Config.Labels \"${key}\" }}")"
  [[ "${actual}" == "${expected}" ]] || fail "label ${key} is '${actual}', expected '${expected}'"
}

command -v docker >/dev/null 2>&1 || fail "docker is not installed"
command -v curl >/dev/null 2>&1 || fail "curl is not installed"
command -v python3 >/dev/null 2>&1 || fail "python3 is not installed"
command -v shasum >/dev/null 2>&1 || command -v sha256sum >/dev/null 2>&1 || fail "SHA-256 utility is not installed"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
docker buildx version >/dev/null 2>&1 || fail "docker buildx is unavailable"
mkdir -p "$(dirname "${BUILD_METADATA_OUTPUT}")"

BUILDX_METADATA_PROVENANCE=max BUILDX_METADATA_WARNINGS=1 \
  docker buildx build --progress=plain --load \
    --metadata-file "${BUILD_METADATA_OUTPUT}" \
    --build-arg "OCI_SOURCE=${OCI_SOURCE}" \
    --build-arg "OCI_REVISION=${OCI_REVISION}" \
    --build-arg "OCI_VERSION=${OCI_VERSION}" \
    --tag "${IMAGE_TAG}" "${ROOT_DIR}"

python3 - "${BUILD_METADATA_OUTPUT}" <<'PY'
import json
import re
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    metadata = json.load(handle)

digest = metadata.get("containerimage.digest", "")
if re.fullmatch(r"sha256:[0-9a-f]{64}", digest) is None:
    raise SystemExit("build metadata is missing a valid container image digest")

statement = metadata.get("buildx.build.provenance")
if not isinstance(statement, dict):
    raise SystemExit("build metadata is missing Buildx provenance")
predicate = statement.get("predicate", statement)
if predicate.get("buildType") != "https://mobyproject.org/buildkit@v1":
    raise SystemExit("build metadata has an unexpected provenance buildType")
entrypoint = predicate.get("invocation", {}).get("configSource", {}).get("entryPoint")
if entrypoint != "Dockerfile":
    raise SystemExit("build provenance does not identify Dockerfile as the entry point")

materials = predicate.get("materials")
if not isinstance(materials, list) or len(materials) < 2:
    raise SystemExit("build provenance must contain Dockerfile and Node materials")
observed = {
    value
    for material in materials
    if isinstance(material, dict)
    for value in [material.get("digest", {}).get("sha256")]
    if isinstance(value, str)
}
expected = {
    "87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89",
    "16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2",
}
if not expected.issubset(observed):
    raise SystemExit("build provenance does not contain every reviewed Web material digest")

print(f"validated BuildKit provenance with {len(materials)} immutable materials")
PY

assert_label "org.opencontainers.image.source" "${OCI_SOURCE}"
assert_label "org.opencontainers.image.revision" "${OCI_REVISION}"
assert_label "org.opencontainers.image.version" "${OCI_VERSION}"
assert_label "org.opencontainers.image.base.digest" "sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2"

image_user="$(docker image inspect "${IMAGE_TAG}" --format '{{.Config.User}}')"
case "${image_user}" in
  ""|0|root|0:0|root:root) fail "image must run as a non-root user" ;;
esac

healthcheck="$(docker image inspect "${IMAGE_TAG}" --format '{{json .Config.Healthcheck}}')"
[[ "${healthcheck}" != "null" ]] || fail "image has no HEALTHCHECK"

node_version="$(docker run --rm --entrypoint node "${IMAGE_TAG}" --version)"
[[ "${node_version}" == "v22.23.1" ]] || fail "runtime Node version is ${node_version}, expected v22.23.1"

docker run --rm --entrypoint node "${IMAGE_TAG}" -e '
  const fs = require("node:fs");
  const forbidden = [
    "/sbin/apk",
    "/usr/local/include",
    "/usr/local/lib/node_modules",
    "/usr/local/bin/corepack",
    "/usr/local/bin/npm",
    "/usr/local/bin/npx",
    "/usr/local/bin/pnpm",
    "/usr/local/bin/pnpx",
    "/usr/local/bin/yarn",
    "/usr/local/bin/yarnpkg",
  ];
  const present = forbidden.filter((path) => fs.existsSync(path));
  if (fs.existsSync("/opt")) {
    for (const name of fs.readdirSync("/opt")) {
      if (name.startsWith("yarn-")) present.push(`/opt/${name}`);
    }
  }
  for (const name of fs.readdirSync("/app")) {
    if (name === ".env" || name.startsWith(".env.")) present.push(`/app/${name}`);
  }
  if (present.length) {
    console.error(`forbidden runtime paths: ${present.join(", ")}`);
    process.exit(1);
  }
' || fail "image contains development tooling or an embedded environment file"

docker run --detach \
  --name "${CONTAINER_NAME}" \
  --publish 127.0.0.1::3000 \
  "${IMAGE_TAG}" >/dev/null

deadline=$((SECONDS + 60))
while (( SECONDS < deadline )); do
  container_health="$(docker inspect "${CONTAINER_NAME}" --format '{{.State.Health.Status}}')"
  case "${container_health}" in
    healthy) break ;;
    unhealthy) fail "container became unhealthy" ;;
  esac
  sleep 1
done
[[ "${container_health:-}" == "healthy" ]] || fail "container did not become healthy within 60 seconds"

published_port="$(docker port "${CONTAINER_NAME}" 3000/tcp | awk -F: 'NR == 1 { print $NF }')"
[[ -n "${published_port}" ]] || fail "container port 3000 was not published"

robots_status="$(curl --noproxy '*' --silent --show-error --output "${TMP_DIR}/robots.txt" --write-out '%{http_code}' "http://127.0.0.1:${published_port}/robots.txt")"
[[ "${robots_status}" == "200" ]] || fail "robots endpoint returned HTTP ${robots_status}"

docker stop --time 15 "${CONTAINER_NAME}" >/dev/null
exit_code="$(docker inspect "${CONTAINER_NAME}" --format '{{.State.ExitCode}}')"
oom_killed="$(docker inspect "${CONTAINER_NAME}" --format '{{.State.OOMKilled}}')"
[[ "${oom_killed}" == "false" ]] || fail "container was OOM-killed"
case "${exit_code}" in
  0|143) ;;
  *) fail "container exited with code ${exit_code} after SIGTERM" ;;
esac

image_size="$(docker image inspect "${IMAGE_TAG}" --format '{{.Size}}')"
printf 'web container image: %s bytes\n' "${image_size}"
printf 'web container user: %s\n' "${image_user}"
printf 'web container Node: %s\n' "${node_version}"
printf 'web container health: %s\n' "${container_health}"
printf 'robots status: %s\n' "${robots_status}"
printf 'development tooling/env: absent\n'
printf 'bounded SIGTERM exit code: %s\n' "${exit_code}"
printf 'build metadata SHA-256: %s\n' "$(sha256_file "${BUILD_METADATA_OUTPUT}")"

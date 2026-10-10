#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPOSITORY_ROOT="$(git -C "${ROOT_DIR}" rev-parse --show-toplevel 2>/dev/null || dirname "${ROOT_DIR}")"
IMAGE_TAG="${1:-luas-api:container-check}"
CONTAINER_NAME="luas-api-container-check-$$"
TMP_DIR="$(mktemp -d)"
BUILD_METADATA_OUTPUT="${BUILD_METADATA_OUTPUT:-${TMP_DIR}/luas-api.build-metadata.json}"
OCI_SOURCE="${OCI_SOURCE:-https://github.com/zgiai/luas}"
OCI_REVISION="${OCI_REVISION:-$(git -C "${REPOSITORY_ROOT}" rev-parse HEAD 2>/dev/null || printf 'unknown')}"
OCI_VERSION="${OCI_VERSION:-$(git -C "${REPOSITORY_ROOT}" describe --tags --always 2>/dev/null || printf 'dev')}"

cleanup() {
  docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'container verification failed: %s\n' "$1" >&2
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
if not isinstance(materials, list) or len(materials) < 3:
    raise SystemExit("build provenance must contain Dockerfile, builder, and runtime materials")
observed = {
    value
    for material in materials
    if isinstance(material, dict)
    for value in [material.get("digest", {}).get("sha256")]
    if isinstance(value, str)
}
expected = {
    "87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89",
    "cdfd4fe2da6b225d8b40c6b7a105736e548e83ff56d5d8f9394446eeb5eb84e0",
    "aef9602f8710ec12bde19d593fed1f76c708531bb7aba205110f1029786ead7b",
}
if not expected.issubset(observed):
    raise SystemExit("build provenance does not contain every reviewed API material digest")

print(f"validated BuildKit provenance with {len(materials)} immutable materials")
PY

assert_label "org.opencontainers.image.source" "${OCI_SOURCE}"
assert_label "org.opencontainers.image.revision" "${OCI_REVISION}"
assert_label "org.opencontainers.image.version" "${OCI_VERSION}"
assert_label "org.opencontainers.image.base.digest" "sha256:aef9602f8710ec12bde19d593fed1f76c708531bb7aba205110f1029786ead7b"

image_user="$(docker image inspect "${IMAGE_TAG}" --format '{{.Config.User}}')"
case "${image_user}" in
  ""|0|root|0:0|root:root) fail "image must run as a non-root user" ;;
esac

healthcheck="$(docker image inspect "${IMAGE_TAG}" --format '{{json .Config.Healthcheck}}')"
[[ "${healthcheck}" != "null" ]] || fail "image has no HEALTHCHECK"

docker run --detach \
  --name "${CONTAINER_NAME}" \
  --publish 127.0.0.1::8025 \
  --env DB_ENABLED=false \
  --env METRICS_ENABLED=false \
  --env CORS_ALLOW_ORIGINS=https://app.example.com \
  "${IMAGE_TAG}" >/dev/null

deadline=$((SECONDS + 45))
while (( SECONDS < deadline )); do
  container_health="$(docker inspect "${CONTAINER_NAME}" --format '{{.State.Health.Status}}')"
  case "${container_health}" in
    healthy) break ;;
    unhealthy) fail "container became unhealthy" ;;
  esac
  sleep 1
done
[[ "${container_health:-}" == "healthy" ]] || fail "container did not become healthy within 45 seconds"

published_port="$(docker port "${CONTAINER_NAME}" 8025/tcp | awk -F: 'NR == 1 { print $NF }')"
[[ -n "${published_port}" ]] || fail "container port 8025 was not published"

live_status="$(curl --noproxy '*' --silent --show-error --output "${TMP_DIR}/live.json" --write-out '%{http_code}' "http://127.0.0.1:${published_port}/health/live")"
[[ "${live_status}" == "200" ]] || fail "liveness returned HTTP ${live_status}"

ready_status="$(curl --noproxy '*' --silent --show-error --output "${TMP_DIR}/ready.json" --write-out '%{http_code}' "http://127.0.0.1:${published_port}/health/ready")"
[[ "${ready_status}" == "503" ]] || fail "database-disabled readiness returned HTTP ${ready_status} instead of 503"

sleep 1
docker logs "${CONTAINER_NAME}" >"${TMP_DIR}/container.log" 2>&1
grep -q '"message":"HTTP Request"' "${TMP_DIR}/container.log" || fail "request logs are not emitted as JSON to container stdout"

if docker cp "${CONTAINER_NAME}:/app/.env" "${TMP_DIR}/embedded.env" >/dev/null 2>&1; then
  fail "production image embeds /app/.env"
fi

docker stop --time 15 "${CONTAINER_NAME}" >/dev/null
exit_code="$(docker inspect "${CONTAINER_NAME}" --format '{{.State.ExitCode}}')"
[[ "${exit_code}" == "0" ]] || fail "container exited with code ${exit_code} after SIGTERM"

image_size="$(docker image inspect "${IMAGE_TAG}" --format '{{.Size}}')"
printf 'container image: %s bytes\n' "${image_size}"
printf 'container user: %s\n' "${image_user}"
printf 'container health: %s\n' "${container_health}"
printf 'liveness/readiness: %s/%s\n' "${live_status}" "${ready_status}"
printf 'embedded env: absent\n'
printf 'graceful exit code: %s\n' "${exit_code}"
printf 'build metadata SHA-256: %s\n' "$(sha256_file "${BUILD_METADATA_OUTPUT}")"

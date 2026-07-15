#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_TAG="${1:-luas-api:container-check}"
CONTAINER_NAME="luas-api-container-check-$$"
TMP_DIR="$(mktemp -d)"

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

command -v docker >/dev/null 2>&1 || fail "docker is not installed"
command -v curl >/dev/null 2>&1 || fail "curl is not installed"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"

docker build --progress=plain --tag "${IMAGE_TAG}" "${ROOT_DIR}"

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

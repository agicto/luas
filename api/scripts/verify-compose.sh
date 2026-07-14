#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_TAG="${1:-luas-api:compose-check}"
PROJECT_NAME="luas-compose-check-$$"
TMP_DIR="$(mktemp -d)"

compose() {
  docker compose --project-name "${PROJECT_NAME}" --file "${ROOT_DIR}/docker-compose.yml" "$@"
}

cleanup() {
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'compose verification failed: %s\n' "$1" >&2
  compose logs --no-color >&2 2>/dev/null || true
  exit 1
}

command -v docker >/dev/null 2>&1 || fail "docker is not installed"
command -v curl >/dev/null 2>&1 || fail "curl is not installed"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"

if ! docker image inspect "${IMAGE_TAG}" >/dev/null 2>&1; then
  docker build --progress=plain --tag "${IMAGE_TAG}" "${ROOT_DIR}"
fi

export LUAS_API_IMAGE="${IMAGE_TAG}"
export LUAS_API_PORT=0
export LUAS_DB_PORT=0

compose config --quiet
compose up --detach --no-build --wait --wait-timeout 120

api_container="$(compose ps --quiet api)"
[[ -n "${api_container}" ]] || fail "API container is missing"
api_health="$(docker inspect "${api_container}" --format '{{.State.Health.Status}}')"
[[ "${api_health}" == "healthy" ]] || fail "API container health is ${api_health}"

published_address="$(docker port "${api_container}" 8025/tcp | awk 'NR == 1 { print }')"
[[ "${published_address}" == 127.0.0.1:* ]] || fail "API port is not loopback-bound: ${published_address}"
published_port="${published_address##*:}"

ready_status="$(curl --noproxy '*' --silent --show-error --output "${TMP_DIR}/ready.json" --write-out '%{http_code}' "http://127.0.0.1:${published_port}/health/ready")"
[[ "${ready_status}" == "200" ]] || fail "readiness returned HTTP ${ready_status}"

register_status="$(curl --noproxy '*' --silent --show-error \
  --output "${TMP_DIR}/register.json" \
  --write-out '%{http_code}' \
  --header 'Content-Type: application/json' \
  --data '{"username":"compose_check","email":"compose-check@example.com","password":"secret12"}' \
  "http://127.0.0.1:${published_port}/v1/register")"
[[ "${register_status}" == "201" ]] || fail "post-migration registration returned HTTP ${register_status}"

printf 'compose API health: %s\n' "${api_health}"
printf 'compose loopback address: %s\n' "${published_address}"
printf 'compose readiness/register: %s/%s\n' "${ready_status}" "${register_status}"

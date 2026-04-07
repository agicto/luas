#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_DIR="${ROOT_DIR}/tmp/kest"
PORT="${SERVER_PORT:-8025}"
BASE_URL="http://127.0.0.1:${PORT}"
DB_PATH="${TMP_DIR}/zgo-kest.sqlite"
SERVER_LOG="${TMP_DIR}/server.log"
RUN_ID="$(date +%s)"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

mkdir -p "${TMP_DIR}"
rm -f "${DB_PATH}" "${SERVER_LOG}"

if ! command -v kest >/dev/null 2>&1; then
  echo "kest is not installed or not in PATH" >&2
  exit 1
fi

COMMON_ENV=(
  APP_ENV=test
  APP_DEBUG=false
  APP_URL="${BASE_URL}"
  SERVER_MODE=release
  SERVER_PORT="${PORT}"
  DB_ENABLED=true
  DB_DRIVER=sqlite
  DB_NAME="${DB_PATH}"
  JWT_SECRET=kest-test-secret
  AI_ENABLED=false
  TRACING_ENABLED=false
  LOG_CH_ENABLED=false
)

cd "${ROOT_DIR}"

env "${COMMON_ENV[@]}" go run ./cmd/zgo/main.go migrate >/dev/null
env "${COMMON_ENV[@]}" go run ./cmd/server/main.go >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "${BASE_URL}/v1/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -fsS "${BASE_URL}/v1/health" >/dev/null 2>&1; then
  echo "server failed to start for kest run" >&2
  cat "${SERVER_LOG}" >&2
  exit 1
fi

if [[ $# -eq 0 ]]; then
  set -- "${ROOT_DIR}"/tests/kest/*.flow.md
fi

for flow in "$@"; do
  kest run "${flow}" -e local --fail-fast --var "run_id=${RUN_ID}"
done

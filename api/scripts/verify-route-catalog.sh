#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/luas-route-catalog.XXXXXX")"
CLI_FILE="${WORK_DIR}/luas"
CATALOG_FILE="${WORK_DIR}/catalog.json"
trap 'rm -f "${CATALOG_FILE}" "${CLI_FILE}"; rmdir "${WORK_DIR}"' EXIT

cd "${ROOT_DIR}"
go build -o "${CLI_FILE}" ./cmd/luas

(
  cd "${WORK_DIR}"
  env \
    APP_ENV=test \
    DB_ENABLED=false \
    AI_ENABLED=false \
    METRICS_ENABLED=true \
    OPTIONAL_STARTERS= \
    "${CLI_FILE}" route:list --format=json >"${CATALOG_FILE}"
)

python3 scripts/validate-route-catalog.py \
  "${CATALOG_FILE}" \
  --expect-starter audit \
  --expect-starter apikey \
  --expect-starter user \
  --require-route GET / \
  --require-route GET /health \
  --require-route GET /health/live \
  --require-route GET /health/ready \
  --require-route GET /metrics \
  --require-route POST /v1/login \
  --require-route GET /v1/users/profile

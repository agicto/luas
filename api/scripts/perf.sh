#!/usr/bin/env bash
# Runs the k6 performance baseline against a release build of the API.
#
#   LUAS_PERF_POSTGRES_DSN=postgres://user:pass@127.0.0.1:5432/luas_perf?sslmode=disable make perf
#
# The database must be disposable: the script migrates and seeds it. k6 comes from $K6 or PATH.
# The k6 summary is written to $LUAS_PERF_OUTPUT (default: perf/results/summary.json).
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
K6="${K6:-k6}"
PORT="${LUAS_PERF_PORT:-18080}"
OUTPUT="${LUAS_PERF_OUTPUT:-${ROOT_DIR}/perf/results/summary.json}"
TMP_DIR="$(mktemp -d)"
API_PID=""

cleanup() {
  if [[ -n "${API_PID}" ]] && kill -0 "${API_PID}" 2>/dev/null; then
    kill "${API_PID}" 2>/dev/null || true
    wait "${API_PID}" 2>/dev/null || true
  fi
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'perf: %s\n' "$1" >&2
  [[ -f "${TMP_DIR}/api.log" ]] && tail -n 40 "${TMP_DIR}/api.log" >&2
  exit 1
}

[[ -n "${LUAS_PERF_POSTGRES_DSN:-}" ]] || fail "LUAS_PERF_POSTGRES_DSN must name a disposable PostgreSQL database"
command -v "${K6}" >/dev/null 2>&1 || fail "k6 not found; install it or set K6=/path/to/k6"

# Split the DSN into the DB_* variables the API reads.
assignments="$(python3 - "${LUAS_PERF_POSTGRES_DSN}" <<'PY'
import shlex, sys
from urllib.parse import parse_qs, unquote, urlsplit
url = urlsplit(sys.argv[1])
if url.scheme not in ("postgres", "postgresql") or not url.hostname or not url.path.strip("/"):
    sys.exit(2)
values = {
    "DB_HOST": url.hostname,
    "DB_PORT": str(url.port or 5432),
    "DB_NAME": url.path.strip("/"),
    "DB_USERNAME": unquote(url.username or ""),
    "DB_PASSWORD": unquote(url.password or ""),
    "DB_SSLMODE": parse_qs(url.query).get("sslmode", ["disable"])[0],
}
for key, value in values.items():
    print(f"export {key}={shlex.quote(value)}")
PY
)" || fail "LUAS_PERF_POSTGRES_DSN must be postgres://user:pass@host:port/database"
eval "${assignments}"

export LUAS_ENV_FILE=""
export APP_ENV=development
export SERVER_MODE=release
export SERVER_HOST=127.0.0.1
export SERVER_PORT="${PORT}"
export DB_ENABLED=true
export DB_DRIVER=postgres
export DB_LOG_LEVEL=silent
export LOG_LEVEL=warning
export LOG_JSON=true
export AI_ENABLED=false
export OPTIONAL_STARTERS=""
# The baseline measures request cost, not abuse controls; both limiters would reject the load.
export MIDDLEWARE_RATE_LIMIT_ENABLED=false
export AUTH_RATE_LIMIT_ENABLED=false

cd "${ROOT_DIR}"
go build -trimpath -o "${TMP_DIR}/luas" ./cmd/luas
"${TMP_DIR}/luas" migrate >"${TMP_DIR}/migrate.log" 2>&1 || { cat "${TMP_DIR}/migrate.log" >&2; fail "migration failed"; }
"${TMP_DIR}/luas" db:seed >"${TMP_DIR}/seed.log" 2>&1 || { cat "${TMP_DIR}/seed.log" >&2; fail "seeding failed"; }

"${TMP_DIR}/luas" serve >"${TMP_DIR}/api.log" 2>&1 &
API_PID=$!
for _ in $(seq 60); do
  curl -fsS "http://127.0.0.1:${PORT}/health/ready" >/dev/null 2>&1 && break
  kill -0 "${API_PID}" 2>/dev/null || fail "API exited before becoming ready"
  sleep 0.5
done
curl -fsS "http://127.0.0.1:${PORT}/health/ready" >/dev/null 2>&1 || fail "API did not become ready"

mkdir -p "$(dirname "${OUTPUT}")"
LUAS_PERF_BASE_URL="http://127.0.0.1:${PORT}" "${K6}" run \
  --quiet \
  --summary-export "${OUTPUT}" \
  "${ROOT_DIR}/perf/k6/baseline.js"
printf 'perf: summary written to %s\n' "${OUTPUT}"

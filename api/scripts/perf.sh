#!/usr/bin/env bash
# Runs the k6 performance baseline against a release build of the API.
#
#   LUAS_PERF_POSTGRES_DSN=postgres://user:pass@127.0.0.1:5432/luas_perf?sslmode=disable make perf
#
# The database must be disposable: the script migrates and seeds it. k6 comes from $K6 or PATH.
# The k6 summary is written to $LUAS_PERF_OUTPUT (default: perf/results/summary.json).
# LUAS_PERF_AUDIT_ROWS (default 1000000) audit rows are seeded with psql so history reads run
# against a large table; set it to 0 to skip.
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
K6="${K6:-k6}"
PORT="${LUAS_PERF_PORT:-18080}"
OUTPUT="${LUAS_PERF_OUTPUT:-${ROOT_DIR}/perf/results/summary.json}"
AUDIT_ROWS="${LUAS_PERF_AUDIT_ROWS:-1000000}"
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
[[ "${AUDIT_ROWS}" =~ ^[0-9]+$ ]] || fail "LUAS_PERF_AUDIT_ROWS must be a non-negative integer"
if (( AUDIT_ROWS > 0 )); then
  command -v psql >/dev/null 2>&1 || fail "psql is required to seed audit rows; install it or set LUAS_PERF_AUDIT_ROWS=0"
fi

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
if (( AUDIT_ROWS > 0 )); then
  # Half of the history belongs to the measured account, one second apart, so its reads page
  # through hundreds of thousands of rows. Reruns top up to the target instead of growing.
  PGPASSWORD="${DB_PASSWORD}" psql --quiet --no-psqlrc --set ON_ERROR_STOP=1 --set rows="${AUDIT_ROWS}" \
    "host=${DB_HOST} port=${DB_PORT} dbname=${DB_NAME} user=${DB_USERNAME} sslmode=${DB_SSLMODE}" <<'SQL' >"${TMP_DIR}/audit-seed.log" 2>&1 || { cat "${TMP_DIR}/audit-seed.log" >&2; fail "audit seeding failed"; }
INSERT INTO audit_logs (created_at, updated_at, user_id, actor_type, actor_id, action, resource, result,
  method, path, route_name, status_code, request_id, ip_address, user_agent, changes, metadata)
SELECT now() - g * interval '1 second', now(),
  CASE WHEN g % 2 = 0 THEN (SELECT id FROM users WHERE email = 'user@example.com') ELSE NULL END,
  'user', NULL, 'store', 'api_keys', 'success', 'POST', '/v1/api-keys', 'apikey.store', 201,
  md5(g::text), '127.0.0.1', 'perf-seed', '', ''
FROM generate_series(1, greatest(:rows - (SELECT count(*) FROM audit_logs), 0)) AS g;
ANALYZE audit_logs;
SQL
fi

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

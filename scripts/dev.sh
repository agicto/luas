#!/usr/bin/env bash
# One-command local stack: PostgreSQL in Docker, then the API, Web, and Admin on the host.
#
#   make dev          start everything and stream prefixed logs; Ctrl-C stops the processes
#   make dev-down     stop the database container and keep its data
#   make dev-reset    stop the database container and delete its data
#
# Ports and the starter selection are environment variables; see `print_usage`.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.luas-dev"
COMPOSE=(docker compose --file "${ROOT_DIR}/compose.dev.yaml")

API_PORT="${LUAS_API_PORT:-8025}"
WEB_PORT="${LUAS_WEB_PORT:-3000}"
ADMIN_PORT="${LUAS_ADMIN_PORT:-4173}"
export LUAS_DB_PORT="${LUAS_DB_PORT:-55432}"
SERVICES="${LUAS_DEV_SERVICES:-api,web,admin}"
STARTERS="${LUAS_DEV_STARTERS:-organization,permission,notification,asset,setting,usage,webhook,operator}"
WEB_FEATURES="${LUAS_DEV_WEB_FEATURES:-organization,permission,notification,asset,setting,usage,webhook}"
ADMIN_FEATURES="${LUAS_DEV_ADMIN_FEATURES:-operator,organization,webhook,notification}"

print_usage() {
  cat <<USAGE
Usage: scripts/dev.sh [up|down|reset]

Environment:
  LUAS_API_PORT            API port (default 8025)
  LUAS_WEB_PORT            Web port (default 3000)
  LUAS_ADMIN_PORT          Admin Console port (default 4173)
  LUAS_DB_PORT             PostgreSQL host port (default 55432)
  LUAS_DEV_SERVICES        Comma list of api, web, admin (default all)
  LUAS_DEV_STARTERS        API OPTIONAL_STARTERS (default every optional starter)
  LUAS_DEV_WEB_FEATURES    Web NEXT_PUBLIC_OPTIONAL_FEATURES
  LUAS_DEV_ADMIN_FEATURES  Admin VITE_OPTIONAL_FEATURES

Seeded accounts (password "secret"): admin@example.com is a platform operator, user@example.com is
a regular user.
USAGE
}

log() { printf '\033[1m[dev]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[dev]\033[0m %s\n' "$*" >&2; exit 1; }
wants() { [[ ",${SERVICES}," == *",$1,"* ]]; }

require_tools() {
  command -v docker >/dev/null || fail "Docker is required for the development database."
  docker compose version >/dev/null 2>&1 || fail "Docker Compose v2 is required."
  docker info >/dev/null 2>&1 || fail "The Docker daemon is not running."
  if wants api; then command -v go >/dev/null || fail "Go is required for the API."; fi
  if wants web || wants admin; then
    command -v node >/dev/null || fail "Node.js 22.12+ is required for the browser shells."
    local major
    major="$(node -p 'process.versions.node.split(".")[0]')"
    [[ "${major}" -ge 22 ]] || fail "Node.js 22.12+ is required; found $(node --version)."
  fi
}

port_free() {
  ! (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
}

require_ports() {
  local name port
  for name in API WEB ADMIN; do
    case "${name}" in
      API) wants api || continue; port="${API_PORT}" ;;
      WEB) wants web || continue; port="${WEB_PORT}" ;;
      ADMIN) wants admin || continue; port="${ADMIN_PORT}" ;;
    esac
    port_free "${port}" || fail "Port ${port} is in use. Choose another with LUAS_${name}_PORT=<port>."
  done
}

start_database() {
  log "Starting PostgreSQL on 127.0.0.1:${LUAS_DB_PORT}"
  "${COMPOSE[@]}" up --detach --wait postgres >/dev/null
}

api_environment() {
  export APP_ENV=development
  export LUAS_ENV_FILE=
  export SERVER_HOST=127.0.0.1
  export SERVER_PORT="${API_PORT}"
  export DB_ENABLED=true DB_DRIVER=postgres DB_HOST=127.0.0.1 DB_PORT="${LUAS_DB_PORT}"
  export DB_NAME=luas DB_USERNAME=luas DB_PASSWORD=luas-local-only DB_SSLMODE=disable
  export OPTIONAL_STARTERS="${STARTERS}"
  export AI_ENABLED=false
  export QUEUE_DRIVER=postgres
  export CORS_ALLOW_ORIGINS="http://localhost:${WEB_PORT},http://127.0.0.1:${WEB_PORT}"
  export OPERATOR_ALLOWED_ORIGINS="http://127.0.0.1:${ADMIN_PORT},http://localhost:${ADMIN_PORT}"
  export OBJECT_STORAGE_LOCAL_ROOT="${STATE_DIR}/objects"
  export ASSET_TRANSFER_SIGNING_KEY=luas-local-only-asset-transfer-signing-key-0123456789
  export WEBHOOK_ENCRYPTION_KEY=luas-local-only-webhook-encryption-key-0123456789
  export WEBHOOK_ALLOW_INSECURE_HTTP=true
  export WEBHOOK_ALLOW_PRIVATE_TARGETS=true
}

prepare_api() {
  log "Building the API and preparing the database"
  mkdir -p "${STATE_DIR}/bin" "${STATE_DIR}/objects"
  (cd "${ROOT_DIR}/api" && go build -o "${STATE_DIR}/bin/luas" ./cmd/luas)
  api_environment
  # The API resolves runtime paths such as storage/logs from its working directory.
  (
    cd "${ROOT_DIR}/api"
    "${STATE_DIR}/bin/luas" migrate >/dev/null
    "${STATE_DIR}/bin/luas" seed >/dev/null
    if [[ ",${STARTERS}," == *",operator,"* ]]; then
      "${STATE_DIR}/bin/luas" operator:grant admin@example.com >/dev/null
    fi
  )
}

install_node_dependencies() {
  local unit="$1"
  if [[ ! -d "${ROOT_DIR}/${unit}/node_modules" ]]; then
    log "Installing ${unit} dependencies"
    (cd "${ROOT_DIR}/${unit}" && CI=true corepack pnpm install --frozen-lockfile >/dev/null)
  fi
}

PIDS=()

run_prefixed() {
  local label="$1"
  shift
  "$@" > >(sed -u "s/^/[${label}] /") 2>&1 &
  PIDS+=("$!")
  # The script polls its processes itself; disowning stops bash from reporting each termination.
  disown "$!"
}

stop_processes() {
  trap - INT TERM EXIT
  if [[ -n "${PIDS[*]-}" ]]; then
    log "Stopping processes (the database keeps running; make dev-down stops it)"
    local pid
    for pid in ${PIDS[@]+"${PIDS[@]}"}; do
      pkill -TERM -P "${pid}" 2>/dev/null || true
      kill -TERM "${pid}" 2>/dev/null || true
    done
    for _ in $(seq 1 20); do
      local alive=0
      for pid in ${PIDS[@]+"${PIDS[@]}"}; do
        kill -0 "${pid}" 2>/dev/null && alive=1
      done
      [[ "${alive}" -eq 0 ]] && break
      sleep 0.5
    done
  fi
}

wait_for() {
  local label="$1" url="$2"
  for _ in $(seq 1 120); do
    if curl -fsS -o /dev/null "${url}" 2>/dev/null; then
      log "${label} ready at ${url%/health/ready}"
      return 0
    fi
    sleep 1
  done
  fail "${label} did not become ready at ${url}"
}

up() {
  require_tools
  require_ports
  start_database
  if wants api; then prepare_api; fi
  if wants web; then install_node_dependencies web; fi
  if wants admin; then install_node_dependencies admin; fi

  trap stop_processes EXIT
  trap 'stop_processes; exit 0' INT TERM
  if wants api; then
    run_prefixed api bash -c "cd '${ROOT_DIR}/api' && exec '${STATE_DIR}/bin/luas' serve"
    wait_for API "http://127.0.0.1:${API_PORT}/health/ready"
  fi
  if wants web; then
    run_prefixed web env \
      NEXT_PUBLIC_API_URL=/api \
      NEXT_PUBLIC_APP_URL="http://localhost:${WEB_PORT}" \
      NEXT_PUBLIC_OPTIONAL_FEATURES="${WEB_FEATURES}" \
      API_ADAPTER_ENABLED=true \
      API_UPSTREAM_URL="http://127.0.0.1:${API_PORT}/v1" \
      MOCK_BFF_ENABLED=false \
      bash -c "cd '${ROOT_DIR}/web' && exec corepack pnpm exec next dev --turbopack --port ${WEB_PORT}"
    wait_for Web "http://localhost:${WEB_PORT}/"
  fi
  if wants admin; then
    run_prefixed admin env \
      SPA_API_PROXY_TARGET="http://127.0.0.1:${API_PORT}" \
      VITE_OPTIONAL_FEATURES="${ADMIN_FEATURES}" \
      bash -c "cd '${ROOT_DIR}/admin' && exec corepack pnpm exec vite --host 127.0.0.1 --port ${ADMIN_PORT} --strictPort"
    wait_for Admin "http://127.0.0.1:${ADMIN_PORT}/"
  fi

  log "Stack is up. Sign in with admin@example.com / secret (operator) or user@example.com / secret."
  wants web && log "Web:   http://localhost:${WEB_PORT}"
  wants admin && log "Admin: http://127.0.0.1:${ADMIN_PORT}"
  wants api && log "API:   http://127.0.0.1:${API_PORT}"
  # Portable to macOS bash 3.2, which has no `wait -n`: stop when any process exits.
  while true; do
    for pid in "${PIDS[@]}"; do
      kill -0 "${pid}" 2>/dev/null || fail "A process exited; see the log above."
    done
    sleep 1
  done
}

case "${1:-up}" in
  up) up ;;
  down) "${COMPOSE[@]}" down ;;
  reset) "${COMPOSE[@]}" down --volumes && rm -rf "${STATE_DIR}" ;;
  -h | --help | help) print_usage ;;
  *) print_usage; exit 2 ;;
esac

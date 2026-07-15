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

if (( $# == 0 )); then
  docker build --progress=plain --tag "${IMAGE_TAG}" "${ROOT_DIR}"
elif ! docker image inspect "${IMAGE_TAG}" >/dev/null 2>&1; then
  fail "explicit image ${IMAGE_TAG} does not exist; verify it before the Compose lifecycle"
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

organization_flow="skipped"
case ",${OPTIONAL_STARTERS:-}," in
  *,organization,*)
    command -v python3 >/dev/null 2>&1 || fail "python3 is required for optional starter response checks"

    login_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/login.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --data '{"username":"compose-check@example.com","password":"secret12"}' \
      "http://127.0.0.1:${published_port}/v1/login")"
    [[ "${login_status}" == "200" ]] || fail "organization owner login returned HTTP ${login_status}"
    if ! owner_token="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["data"]["access_token"])' "${TMP_DIR}/login.json")"; then
      fail "organization owner login response has no access token"
    fi

    organization_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/organization.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${owner_token}" \
      --data '{"name":"Compose Check Organization","slug":"compose-check-organization"}' \
      "http://127.0.0.1:${published_port}/v1/organizations")"
    [[ "${organization_status}" == "201" ]] || fail "organization creation returned HTTP ${organization_status}"
    if ! organization_id="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["data"]["id"])' "${TMP_DIR}/organization.json")"; then
      fail "organization creation response has no ID"
    fi

    invitation_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/invitation.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${owner_token}" \
      --data '{"email":"compose-invitee@example.com","role":"member"}' \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/invitations")"
    [[ "${invitation_status}" == "201" ]] || fail "organization invitation returned HTTP ${invitation_status}"
    if ! invitation_id="$(python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
if payload["email_send_status"] != "not_configured":
    raise SystemExit(1)
if "token" in payload["invitation"] or "token_hash" in payload["invitation"]:
    raise SystemExit(1)
print(payload["invitation"]["id"])
' "${TMP_DIR}/invitation.json")"; then
      fail "organization invitation response violates the token or email-status contract"
    fi

    duplicate_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/invitation-duplicate.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${owner_token}" \
      --data '{"email":"compose-invitee@example.com","role":"member"}' \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/invitations")"
    [[ "${duplicate_status}" == "409" ]] || fail "duplicate invitation returned HTTP ${duplicate_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "ORGANIZATION.INVITATION.ALREADY_PENDING" else 1)' "${TMP_DIR}/invitation-duplicate.json"; then
      fail "duplicate invitation returned the wrong error_code"
    fi

    invitation_list_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/invitation-list.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/invitations")"
    [[ "${invitation_list_status}" == "200" ]] || fail "invitation list returned HTTP ${invitation_list_status}"

    revoke_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/invitations/${invitation_id}")"
    [[ "${revoke_status}" == "204" ]] || fail "invitation revoke returned HTTP ${revoke_status}"

    replacement_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/invitation-replacement.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${owner_token}" \
      --data '{"email":"compose-invitee@example.com","role":"admin"}' \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/invitations")"
    [[ "${replacement_status}" == "201" ]] || fail "replacement invitation returned HTTP ${replacement_status}"
    organization_flow="${organization_status}/${invitation_status}/${duplicate_status}/${invitation_list_status}/${revoke_status}/${replacement_status}"
    ;;
esac

printf 'compose API health: %s\n' "${api_health}"
printf 'compose loopback address: %s\n' "${published_address}"
printf 'compose readiness/register: %s/%s\n' "${ready_status}" "${register_status}"
printf 'compose organization/invitation flow: %s\n' "${organization_flow}"

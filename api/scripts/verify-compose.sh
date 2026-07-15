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

register_and_login_user() {
  local username="$1"
  local email="$2"
  local prefix="$3"
  local status

  status="$(curl --noproxy '*' --silent --show-error \
    --output "${TMP_DIR}/${prefix}-register.json" \
    --write-out '%{http_code}' \
    --header 'Content-Type: application/json' \
    --data "{\"username\":\"${username}\",\"email\":\"${email}\",\"password\":\"secret12\"}" \
    "http://127.0.0.1:${published_port}/v1/register")"
  [[ "${status}" == "201" ]] || fail "${prefix} registration returned HTTP ${status}"

  status="$(curl --noproxy '*' --silent --show-error \
    --output "${TMP_DIR}/${prefix}-login.json" \
    --write-out '%{http_code}' \
    --header 'Content-Type: application/json' \
    --data "{\"username\":\"${email}\",\"password\":\"secret12\"}" \
    "http://127.0.0.1:${published_port}/v1/login")"
  [[ "${status}" == "200" ]] || fail "${prefix} login returned HTTP ${status}"

  python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
print(payload["user"]["id"], payload["access_token"])
' "${TMP_DIR}/${prefix}-login.json"
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
organization_context_flow="skipped"
membership_flow="skipped"
permission_flow="skipped"
permission_migration_flow="skipped"
account_race_flow="skipped"
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
    if ! owner_user_id="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["data"]["user"]["id"])' "${TMP_DIR}/login.json")"; then
      fail "organization owner login response has no user ID"
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

    context_preflight_status="$(curl --noproxy '*' --silent --show-error \
      --dump-header "${TMP_DIR}/organization-context-preflight.headers" \
      --output /dev/null \
      --write-out '%{http_code}' \
      --request OPTIONS \
      --header 'Origin: http://localhost:3000' \
      --header 'Access-Control-Request-Method: GET' \
      --header 'Access-Control-Request-Headers: Authorization, Organization-Id' \
      "http://127.0.0.1:${published_port}/v1/organization-context")"
    [[ "${context_preflight_status}" == "204" ]] || fail "organization context preflight returned HTTP ${context_preflight_status}"
    if ! python3 -c '
import sys
headers = open(sys.argv[1], encoding="utf-8").read().splitlines()
allowed = ",".join(line.split(":", 1)[1] for line in headers if line.lower().startswith("access-control-allow-headers:"))
raise SystemExit(0 if "organization-id" in allowed.lower() else 1)
' "${TMP_DIR}/organization-context-preflight.headers"; then
      fail "organization context preflight does not allow Organization-Id"
    fi

    context_required_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/organization-context-required.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      "http://127.0.0.1:${published_port}/v1/organization-context")"
    [[ "${context_required_status}" == "400" ]] || fail "missing organization context returned HTTP ${context_required_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "ORGANIZATION.CONTEXT_REQUIRED" else 1)' "${TMP_DIR}/organization-context-required.json"; then
      fail "missing organization context returned the wrong error_code"
    fi

    context_owner_status="$(curl --noproxy '*' --silent --show-error \
      --dump-header "${TMP_DIR}/organization-context-owner.headers" \
      --output "${TMP_DIR}/organization-context-owner.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      --header "Organization-Id: ${organization_id}" \
      "http://127.0.0.1:${published_port}/v1/organization-context")"
    [[ "${context_owner_status}" == "200" ]] || fail "owner organization context returned HTTP ${context_owner_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
headers = open(sys.argv[2], encoding="utf-8").read().splitlines()
vary = ",".join(line.split(":", 1)[1] for line in headers if line.lower().startswith("vary:"))
valid = (
    payload["organization_id"] == int(sys.argv[3])
    and payload["user_id"] == int(sys.argv[4])
    and payload["membership_id"] > 0
    and payload["role"] == "owner"
    and "organization-id" in vary.lower()
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/organization-context-owner.json" "${TMP_DIR}/organization-context-owner.headers" "${organization_id}" "${owner_user_id}"; then
      fail "owner organization context violates identity, role, membership, or Vary contract"
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

    read -r admin_user_id admin_token < <(register_and_login_user "compose_admin" "compose-admin@example.com" "organization-admin")
    read -r member_a_user_id member_a_token < <(register_and_login_user "compose_member_a" "compose-member-a@example.com" "organization-member-a")
    read -r member_b_user_id member_b_token < <(register_and_login_user "compose_member_b" "compose-member-b@example.com" "organization-member-b")

    context_outsider_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/organization-context-outsider.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${admin_token}" \
      --header "Organization-Id: ${organization_id}" \
      "http://127.0.0.1:${published_port}/v1/organization-context")"
    [[ "${context_outsider_status}" == "404" ]] || fail "non-member organization context returned HTTP ${context_outsider_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "ORGANIZATION.NOT_FOUND" else 1)' "${TMP_DIR}/organization-context-outsider.json"; then
      fail "non-member organization context returned the wrong error_code"
    fi

    if ! compose exec -T postgres psql --username luas --dbname luas --set ON_ERROR_STOP=1 --command "
INSERT INTO organization_memberships (organization_id, user_id, role, created_at, updated_at)
VALUES
  (${organization_id}, ${admin_user_id}, 'admin', NOW(), NOW()),
  (${organization_id}, ${member_a_user_id}, 'member', NOW(), NOW()),
  (${organization_id}, ${member_b_user_id}, 'member', NOW(), NOW());
" >/dev/null; then
      fail "organization member fixture insertion failed"
    fi

    context_member_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/organization-context-member.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${admin_token}" \
      --header "Organization-Id: ${organization_id}" \
      "http://127.0.0.1:${published_port}/v1/organization-context")"
    [[ "${context_member_status}" == "200" ]] || fail "member organization context returned HTTP ${context_member_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
valid = (
    payload["organization_id"] == int(sys.argv[2])
    and payload["user_id"] == int(sys.argv[3])
    and payload["membership_id"] > 0
    and payload["role"] == "admin"
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/organization-context-member.json" "${organization_id}" "${admin_user_id}"; then
      fail "member organization context violates current membership semantics"
    fi
    organization_context_flow="${context_preflight_status}/${context_required_status}/${context_owner_status}/${context_outsider_status}/${context_member_status}"

    member_list_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/member-list.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${member_a_token}" \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members")"
    [[ "${member_list_status}" == "200" ]] || fail "organization member list returned HTTP ${member_list_status}"
    if ! read -r owner_member_id admin_member_id member_a_member_id member_b_member_id < <(python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))
items = payload["data"]
if payload["meta"]["total"] != 4 or len(items) != 4:
    raise SystemExit(1)
for item in items:
    if any(field in item for field in ("email", "phone", "status", "password")):
        raise SystemExit(1)
by_user = {item["user_id"]: item["id"] for item in items}
print(*(by_user[int(value)] for value in sys.argv[2:]))
' "${TMP_DIR}/member-list.json" "${owner_user_id}" "${admin_user_id}" "${member_a_user_id}" "${member_b_user_id}"); then
      fail "organization member list violates count, identity, or privacy contract"
    fi

    case ",${OPTIONAL_STARTERS:-}," in
      *,permission,*)
        permission_owner_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-owner.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/permission-context")"
        [[ "${permission_owner_status}" == "200" ]] || fail "owner permission context returned HTTP ${permission_owner_status}"
        if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
expected = {
    "permission.roles.read",
    "permission.roles.manage",
    "permission.assignments.read",
    "permission.assignments.manage",
}
valid = payload["is_owner"] is True and set(payload["permissions"]) == expected
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/permission-owner.json"; then
          fail "owner permission context does not expose the registered catalog"
        fi

        permission_admin_denied_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-admin-denied.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${admin_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/access-roles")"
        [[ "${permission_admin_denied_status}" == "403" ]] || fail "ungranted access-role list returned HTTP ${permission_admin_denied_status}"

        permission_role_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-manager-role.json" \
          --write-out '%{http_code}' \
          --header 'Content-Type: application/json' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          --data '{"name":"Role Manager","slug":"role-manager","permissions":["permission.roles.read","permission.roles.manage"]}' \
          "http://127.0.0.1:${published_port}/v1/access-roles")"
        [[ "${permission_role_status}" == "201" ]] || fail "permission manager role creation returned HTTP ${permission_role_status}"
        permission_role_id="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["data"]["id"])' "${TMP_DIR}/permission-manager-role.json")"

        permission_assign_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-admin-assignment.json" \
          --write-out '%{http_code}' \
          --request PUT \
          --header 'Content-Type: application/json' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          --data "{\"access_role_ids\":[${permission_role_id}]}" \
          "http://127.0.0.1:${published_port}/v1/organization-members/${admin_member_id}/access-roles")"
        [[ "${permission_assign_status}" == "200" ]] || fail "permission manager assignment returned HTTP ${permission_assign_status}"

        permission_delegated_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-delegated-role.json" \
          --write-out '%{http_code}' \
          --header 'Content-Type: application/json' \
          --header "Authorization: Bearer ${admin_token}" \
          --header "Organization-Id: ${organization_id}" \
          --data '{"name":"Role Reader","slug":"role-reader","permissions":["permission.roles.read"]}' \
          "http://127.0.0.1:${published_port}/v1/access-roles")"
        [[ "${permission_delegated_status}" == "201" ]] || fail "delegated subset role creation returned HTTP ${permission_delegated_status}"

        permission_escalation_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-escalation-denied.json" \
          --write-out '%{http_code}' \
          --header 'Content-Type: application/json' \
          --header "Authorization: Bearer ${admin_token}" \
          --header "Organization-Id: ${organization_id}" \
          --data '{"name":"Assignment Manager","slug":"assignment-manager","permissions":["permission.assignments.manage"]}' \
          "http://127.0.0.1:${published_port}/v1/access-roles")"
        [[ "${permission_escalation_status}" == "403" ]] || fail "delegated privilege escalation returned HTTP ${permission_escalation_status}"

        permission_delete_status="$(curl --noproxy '*' --silent --show-error \
          --output /dev/null \
          --write-out '%{http_code}' \
          --request DELETE \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/access-roles/${permission_role_id}")"
        [[ "${permission_delete_status}" == "204" ]] || fail "permission role cascade delete returned HTTP ${permission_delete_status}"

        permission_admin_revoked_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-admin-revoked.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${admin_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/access-roles")"
        [[ "${permission_admin_revoked_status}" == "403" ]] || fail "deleted role did not revoke delegated permission"
        permission_flow="${permission_owner_status}/${permission_admin_denied_status}/${permission_role_status}/${permission_assign_status}/${permission_delegated_status}/${permission_escalation_status}/${permission_delete_status}/${permission_admin_revoked_status}"

        if ! compose exec -T api /app/luas db:rollback --step=1 >"${TMP_DIR}/permission-rollback.log" 2>&1; then
          fail "permission migration rollback failed"
        fi
        permission_tables_down="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('permission_roles', 'permission_role_grants', 'permission_role_assignments');
")"
        [[ "${permission_tables_down}" == "0" ]] || fail "permission migration rollback left ${permission_tables_down} table(s)"

        if ! compose exec -T api /app/luas db:migrate >"${TMP_DIR}/permission-migrate.log" 2>&1; then
          fail "permission migration re-apply failed"
        fi
        permission_tables_up="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('permission_roles', 'permission_role_grants', 'permission_role_assignments');
")"
        [[ "${permission_tables_up}" == "3" ]] || fail "permission migration re-apply created ${permission_tables_up}/3 tables"
        permission_post_migrate_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/permission-post-migrate.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/permission-context")"
        [[ "${permission_post_migrate_status}" == "200" ]] || fail "permission context after migration re-apply returned HTTP ${permission_post_migrate_status}"
        permission_migration_flow="down:${permission_tables_down}/up:${permission_tables_up}/http:${permission_post_migrate_status}"
        ;;
    esac

    admin_role_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/member-role-admin-denied.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${admin_token}" \
      --data '{"role":"admin"}' \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members/${member_a_member_id}")"
    [[ "${admin_role_status}" == "403" ]] || fail "admin member role change returned HTTP ${admin_role_status}"

    role_promote_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/member-role-promote.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${owner_token}" \
      --data '{"role":"admin"}' \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members/${member_a_member_id}")"
    [[ "${role_promote_status}" == "200" ]] || fail "owner member promotion returned HTTP ${role_promote_status}"

    role_demote_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/member-role-demote.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${owner_token}" \
      --data '{"role":"member"}' \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members/${member_a_member_id}")"
    [[ "${role_demote_status}" == "200" ]] || fail "owner member demotion returned HTTP ${role_demote_status}"

    membership_delete_blocked_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/member-account-delete-blocked.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${member_a_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/users/account")"
    [[ "${membership_delete_blocked_status}" == "409" ]] || fail "member account deletion guard returned HTTP ${membership_delete_blocked_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED" else 1)' "${TMP_DIR}/member-account-delete-blocked.json"; then
      fail "member account deletion guard returned the wrong error_code"
    fi

    member_remove_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${admin_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members/${member_a_member_id}")"
    [[ "${member_remove_status}" == "204" ]] || fail "admin member removal returned HTTP ${member_remove_status}"

    removed_account_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${member_a_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/users/account")"
    [[ "${removed_account_status}" == "204" ]] || fail "removed member account deletion returned HTTP ${removed_account_status}"

    # Run two ownership requests together against PostgreSQL. Exactly one may commit.
    (
      curl --noproxy '*' --silent --show-error \
        --output "${TMP_DIR}/ownership-transfer-admin.json" \
        --write-out '%{http_code}' \
        --header 'Content-Type: application/json' \
        --header "Authorization: Bearer ${owner_token}" \
        --data "{\"new_owner_member_id\":${admin_member_id}}" \
        "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/ownership-transfer" \
        >"${TMP_DIR}/ownership-transfer-admin.status"
    ) &
    transfer_admin_pid=$!
    (
      curl --noproxy '*' --silent --show-error \
        --output "${TMP_DIR}/ownership-transfer-member.json" \
        --write-out '%{http_code}' \
        --header 'Content-Type: application/json' \
        --header "Authorization: Bearer ${owner_token}" \
        --data "{\"new_owner_member_id\":${member_b_member_id}}" \
        "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/ownership-transfer" \
        >"${TMP_DIR}/ownership-transfer-member.status"
    ) &
    transfer_member_pid=$!
    wait "${transfer_admin_pid}"
    wait "${transfer_member_pid}"
    transfer_admin_status="$(<"${TMP_DIR}/ownership-transfer-admin.status")"
    transfer_member_status="$(<"${TMP_DIR}/ownership-transfer-member.status")"
    transfer_statuses="$(printf '%s\n%s\n' "${transfer_admin_status}" "${transfer_member_status}" | sort -n | paste -sd/ -)"
    [[ "${transfer_statuses}" == "200/403" ]] || fail "concurrent ownership transfer returned HTTP ${transfer_statuses}"

    post_transfer_list_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/member-list-post-transfer.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members")"
    [[ "${post_transfer_list_status}" == "200" ]] || fail "post-transfer member list returned HTTP ${post_transfer_list_status}"
    if ! read -r current_owner_member_id current_owner_user_id < <(python3 -c '
import json, sys
items = json.load(open(sys.argv[1]))["data"]
owners = [item for item in items if item["role"] == "owner"]
previous = [item for item in items if item["user_id"] == int(sys.argv[2])]
if len(owners) != 1 or len(previous) != 1 or previous[0]["role"] != "admin":
    raise SystemExit(1)
print(owners[0]["id"], owners[0]["user_id"])
' "${TMP_DIR}/member-list-post-transfer.json" "${owner_user_id}"); then
      fail "ownership transfer did not preserve exactly one owner and demote the previous owner"
    fi

    if [[ "${current_owner_user_id}" == "${admin_user_id}" ]]; then
      current_owner_token="${admin_token}"
    elif [[ "${current_owner_user_id}" == "${member_b_user_id}" ]]; then
      current_owner_token="${member_b_token}"
    else
      fail "ownership transfer selected an unexpected owner"
    fi

    self_transfer_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/ownership-transfer-self.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${current_owner_token}" \
      --data "{\"new_owner_member_id\":${current_owner_member_id}}" \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/ownership-transfer")"
    [[ "${self_transfer_status}" == "409" ]] || fail "self ownership transfer returned HTTP ${self_transfer_status}"

    previous_owner_leave_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members/${owner_member_id}")"
    [[ "${previous_owner_leave_status}" == "204" ]] || fail "previous owner leave returned HTTP ${previous_owner_leave_status}"

    previous_owner_delete_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${owner_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/users/account")"
    [[ "${previous_owner_delete_status}" == "204" ]] || fail "previous owner account deletion returned HTTP ${previous_owner_delete_status}"

    current_owner_leave_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/current-owner-leave.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${current_owner_token}" \
      --request DELETE \
      "http://127.0.0.1:${published_port}/v1/organizations/${organization_id}/members/${current_owner_member_id}")"
    [[ "${current_owner_leave_status}" == "409" ]] || fail "current owner leave returned HTTP ${current_owner_leave_status}"
    membership_flow="${member_list_status}/${admin_role_status}/${role_promote_status}/${role_demote_status}/${membership_delete_blocked_status}/${member_remove_status}/${removed_account_status}/${transfer_statuses}/${post_transfer_list_status}/${self_transfer_status}/${previous_owner_leave_status}/${previous_owner_delete_status}/${current_owner_leave_status}"

    read -r race_user_id race_user_token < <(register_and_login_user "compose_race_user" "compose-race-user@example.com" "organization-race-user")
    # Account deletion and membership creation share a user-row lock. Either may win, never both.
    (
      curl --noproxy '*' --silent --show-error \
        --output "${TMP_DIR}/race-account-delete.json" \
        --write-out '%{http_code}' \
        --header "Authorization: Bearer ${race_user_token}" \
        --request DELETE \
        "http://127.0.0.1:${published_port}/v1/users/account" \
        >"${TMP_DIR}/race-account-delete.status"
    ) &
    race_delete_pid=$!
    (
      curl --noproxy '*' --silent --show-error \
        --output "${TMP_DIR}/race-organization-create.json" \
        --write-out '%{http_code}' \
        --header 'Content-Type: application/json' \
        --header "Authorization: Bearer ${race_user_token}" \
        --data '{"name":"Compose Race Organization","slug":"compose-race-organization"}' \
        "http://127.0.0.1:${published_port}/v1/organizations" \
        >"${TMP_DIR}/race-organization-create.status"
    ) &
    race_create_pid=$!
    wait "${race_delete_pid}"
    wait "${race_create_pid}"
    race_delete_status="$(<"${TMP_DIR}/race-account-delete.status")"
    race_create_status="$(<"${TMP_DIR}/race-organization-create.status")"
    race_statuses="$(printf '%s\n%s\n' "${race_delete_status}" "${race_create_status}" | sort -n | paste -sd/ -)"
    if [[ "${race_statuses}" != "201/409" && "${race_statuses}" != "204/404" ]]; then
      fail "concurrent account deletion and membership creation returned HTTP ${race_statuses}"
    fi
    orphaned_memberships="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM organization_memberships AS memberships
JOIN users ON users.id = memberships.user_id
WHERE memberships.user_id = ${race_user_id} AND users.deleted_at IS NOT NULL;
")"
    [[ "${orphaned_memberships}" == "0" ]] || fail "concurrent account deletion left ${orphaned_memberships} stale membership(s)"
    account_race_flow="${race_create_status}/${race_delete_status}/orphans:${orphaned_memberships}"
    ;;
esac

printf 'compose API health: %s\n' "${api_health}"
printf 'compose loopback address: %s\n' "${published_address}"
printf 'compose readiness/register: %s/%s\n' "${ready_status}" "${register_status}"
printf 'compose organization/invitation flow: %s\n' "${organization_flow}"
printf 'compose organization/context flow: %s\n' "${organization_context_flow}"
printf 'compose organization/member flow: %s\n' "${membership_flow}"
printf 'compose permission flow: %s\n' "${permission_flow}"
printf 'compose permission migration flow: %s\n' "${permission_migration_flow}"
printf 'compose organization/account race: %s\n' "${account_race_flow}"

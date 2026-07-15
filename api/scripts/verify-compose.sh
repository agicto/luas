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
command -v python3 >/dev/null 2>&1 || fail "python3 is required for authentication response checks"
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

login_status="$(curl --noproxy '*' --silent --show-error \
  --output "${TMP_DIR}/authentication-login.json" \
  --write-out '%{http_code}' \
  --header 'Content-Type: application/json' \
  --data '{"username":"compose-check@example.com","password":"secret12"}' \
  "http://127.0.0.1:${published_port}/v1/login")"
[[ "${login_status}" == "200" ]] || fail "authentication login returned HTTP ${login_status}"
if ! authentication_token="$(python3 -c '
import base64, json, re, sys

payload = json.load(open(sys.argv[1]))["data"]
token = payload["access_token"]
expires_in = payload["expires_in"]
if payload["token_type"] != "Bearer":
    raise SystemExit(1)
if not isinstance(expires_in, int) or isinstance(expires_in, bool) or expires_in <= 0:
    raise SystemExit(1)
if not isinstance(token, str) or re.fullmatch(r"[A-Za-z0-9_-]{43}", token) is None:
    raise SystemExit(1)
if len(base64.urlsafe_b64decode(token + "=")) != 32:
    raise SystemExit(1)
print(token)
' "${TMP_DIR}/authentication-login.json")"; then
  fail "authentication login response does not satisfy the opaque credential contract"
fi

profile_status="$(curl --noproxy '*' --silent --show-error \
  --output "${TMP_DIR}/authentication-profile.json" \
  --write-out '%{http_code}' \
  --header "Authorization: Bearer ${authentication_token}" \
  "http://127.0.0.1:${published_port}/v1/users/profile")"
[[ "${profile_status}" == "200" ]] || fail "authenticated profile returned HTTP ${profile_status}"

logout_status="$(curl --noproxy '*' --silent --show-error \
  --output "${TMP_DIR}/authentication-logout.json" \
  --write-out '%{http_code}' \
  --request POST \
  --header "Authorization: Bearer ${authentication_token}" \
  "http://127.0.0.1:${published_port}/v1/logout")"
[[ "${logout_status}" == "200" ]] || fail "authentication logout returned HTTP ${logout_status}"

revoked_status="$(curl --noproxy '*' --silent --show-error \
  --output "${TMP_DIR}/authentication-revoked.json" \
  --write-out '%{http_code}' \
  --header "Authorization: Bearer ${authentication_token}" \
  "http://127.0.0.1:${published_port}/v1/users/profile")"
[[ "${revoked_status}" == "401" ]] || fail "revoked authentication credential returned HTTP ${revoked_status}"
authentication_flow="${login_status}/${profile_status}/${logout_status}/${revoked_status}"

organization_flow="skipped"
organization_context_flow="skipped"
membership_flow="skipped"
permission_flow="skipped"
permission_migration_flow="skipped"
notification_flow="skipped"
notification_migration_flow="skipped"
asset_flow="skipped"
asset_migration_flow="skipped"
asset_account_race_flow="skipped"
setting_flow="skipped"
setting_migration_flow="skipped"
setting_account_cleanup_flow="skipped"
usage_flow="skipped"
usage_migration_flow="skipped"
usage_account_cleanup_flow="skipped"
webhook_flow="skipped"
webhook_migration_flow="skipped"
account_race_flow="skipped"
case ",${OPTIONAL_STARTERS:-}," in
  *,notification,*)
    command -v python3 >/dev/null 2>&1 || fail "python3 is required for notification response checks"

    notification_login_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-login.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --data '{"username":"compose-check@example.com","password":"secret12"}' \
      "http://127.0.0.1:${published_port}/v1/login")"
    [[ "${notification_login_status}" == "200" ]] || fail "notification recipient login returned HTTP ${notification_login_status}"
    if ! read -r notification_user_id notification_token < <(python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
print(payload["user"]["id"], payload["access_token"])
' "${TMP_DIR}/notification-login.json"); then
      fail "notification recipient login response is incomplete"
    fi
    read -r notification_outsider_id notification_outsider_token < <(
      register_and_login_user "compose_notification_outsider" "compose-notification-outsider@example.com" "notification-outsider"
    )
    [[ "${notification_outsider_id}" != "${notification_user_id}" ]] || fail "notification users are not isolated"

    if ! notification_id="$(compose exec -T postgres psql \
      --username luas \
      --dbname luas \
      --set ON_ERROR_STOP=1 \
      --quiet \
      --tuples-only \
      --no-align \
      --command "
INSERT INTO notifications (
  user_id,
  idempotency_key,
  publication_hash,
  kind,
  title,
  body,
  action_url,
  created_at,
  updated_at
)
VALUES (
  ${notification_user_id},
  'compose:notification:ready',
  repeat('a', 64),
  'starter.compose_ready',
  'Compose notification ready',
  'PostgreSQL notification state is available.',
  '/console',
  NOW(),
  NOW()
)
RETURNING id;
")"; then
      fail "notification fixture insertion failed"
    fi
    notification_id="$(printf '%s' "${notification_id}" | tr -d '[:space:]')"
    [[ "${notification_id}" =~ ^[1-9][0-9]*$ ]] || fail "notification fixture returned invalid ID ${notification_id}"

    if ! compose exec -T postgres psql --username luas --dbname luas --set ON_ERROR_STOP=1 --command "
INSERT INTO notification_deliveries (
  notification_id,
  channel,
  status,
  attempts,
  available_at,
  lease_token,
  destination_hash,
  delivered_at,
  last_failure_code,
  created_at,
  updated_at
)
VALUES
  (${notification_id}, 'in_app', 'delivered', 0, NOW(), '', '', NOW(), '', NOW(), NOW()),
  (${notification_id}, 'email', 'pending', 0, NOW(), '', '', NULL, '', NOW(), NOW());
" >/dev/null; then
      fail "notification delivery fixture insertion failed"
    fi

    notification_list_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-list.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${notification_token}" \
      "http://127.0.0.1:${published_port}/v1/notifications?status=unread&per_page=10")"
    [[ "${notification_list_status}" == "200" ]] || fail "notification list returned HTTP ${notification_list_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))
item = payload["data"][0]
forbidden = {"user_id", "idempotency_key", "publication_hash", "deliveries", "recipient", "provider_response"}
valid = (
    payload["meta"]["total"] == 1
    and len(payload["data"]) == 1
    and item["id"] == int(sys.argv[2])
    and item["kind"] == "starter.compose_ready"
    and item["action_url"] == "/console"
    and item["is_read"] is False
    and not forbidden.intersection(item)
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/notification-list.json" "${notification_id}"; then
      fail "notification list violates identity, visibility, or privacy contract"
    fi

    notification_status_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-status.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${notification_token}" \
      "http://127.0.0.1:${published_port}/v1/notification-status")"
    [[ "${notification_status_status}" == "200" ]] || fail "notification status returned HTTP ${notification_status_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["data"]["unread_count"] == 1 else 1)' "${TMP_DIR}/notification-status.json"; then
      fail "notification status returned the wrong unread count"
    fi

    notification_invalid_filter_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-invalid-filter.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${notification_token}" \
      "http://127.0.0.1:${published_port}/v1/notifications?status=everything")"
    [[ "${notification_invalid_filter_status}" == "422" ]] || fail "invalid notification filter returned HTTP ${notification_invalid_filter_status}"

    notification_outsider_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-outsider.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${notification_outsider_token}" \
      --data '{"is_read":true}' \
      "http://127.0.0.1:${published_port}/v1/notifications/${notification_id}")"
    [[ "${notification_outsider_status}" == "404" ]] || fail "cross-user notification update returned HTTP ${notification_outsider_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "NOTIFICATION.NOT_FOUND" else 1)' "${TMP_DIR}/notification-outsider.json"; then
      fail "cross-user notification update returned the wrong error_code"
    fi

    notification_preference_get_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-preference-get.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${notification_token}" \
      "http://127.0.0.1:${published_port}/v1/notification-preferences")"
    [[ "${notification_preference_get_status}" == "200" ]] || fail "notification preference read returned HTTP ${notification_preference_get_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if payload == {"in_app_enabled": True, "email_enabled": True} else 1)
' "${TMP_DIR}/notification-preference-get.json"; then
      fail "default notification preferences are not enabled"
    fi

    notification_preference_put_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-preference-put.json" \
      --write-out '%{http_code}' \
      --request PUT \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${notification_token}" \
      --data '{"in_app_enabled":false,"email_enabled":false}' \
      "http://127.0.0.1:${published_port}/v1/notification-preferences")"
    [[ "${notification_preference_put_status}" == "200" ]] || fail "notification preference replacement returned HTTP ${notification_preference_put_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if payload == {"in_app_enabled": False, "email_enabled": False} else 1)
' "${TMP_DIR}/notification-preference-put.json"; then
      fail "notification preference replacement did not persist both values"
    fi

    notification_read_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-read-state.json" \
      --write-out '%{http_code}' \
      --request PUT \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${notification_token}" \
      --data "{\"through_id\":${notification_id}}" \
      "http://127.0.0.1:${published_port}/v1/notification-read-state")"
    [[ "${notification_read_status}" == "200" ]] || fail "notification read high-water update returned HTTP ${notification_read_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if payload == {"updated_count": 1, "unread_count": 0} else 1)
' "${TMP_DIR}/notification-read-state.json"; then
      fail "notification read high-water update returned the wrong counts"
    fi

    if ! compose exec -T api /app/luas notification:work --once >"${TMP_DIR}/notification-worker.log" 2>&1; then
      fail "notification worker one-shot dispatch failed"
    fi
    notification_delivery_state="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT status || ':' || attempts || ':' || last_failure_code || ':' || length(destination_hash)
FROM notification_deliveries
WHERE notification_id = ${notification_id} AND channel = 'email';
")"
    [[ "${notification_delivery_state}" == "failed:1:EMAIL.NOT_CONFIGURED:64" ]] || fail "notification worker produced ${notification_delivery_state}"
    notification_recipient_columns="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'notification_deliveries'
  AND column_name IN ('recipient', 'recipient_email', 'provider_response', 'error_message');
")"
    [[ "${notification_recipient_columns}" == "0" ]] || fail "notification delivery ledger exposes ${notification_recipient_columns} sensitive column(s)"
    notification_flow="${notification_list_status}/${notification_status_status}/${notification_invalid_filter_status}/${notification_outsider_status}/${notification_preference_get_status}/${notification_preference_put_status}/${notification_read_status}/worker:${notification_delivery_state}"

    if ! compose exec -T api /app/luas db:rollback --step=1 >"${TMP_DIR}/notification-rollback.log" 2>&1; then
      fail "notification migration rollback failed"
    fi
    notification_tables_down="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('notifications', 'notification_deliveries', 'notification_preferences');
")"
    [[ "${notification_tables_down}" == "0" ]] || fail "notification migration rollback left ${notification_tables_down} table(s)"

    if ! compose exec -T api /app/luas db:migrate >"${TMP_DIR}/notification-migrate.log" 2>&1; then
      fail "notification migration re-apply failed"
    fi
    notification_tables_up="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('notifications', 'notification_deliveries', 'notification_preferences');
")"
    [[ "${notification_tables_up}" == "3" ]] || fail "notification migration re-apply created ${notification_tables_up}/3 tables"
    notification_post_migrate_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/notification-post-migrate.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${notification_token}" \
      "http://127.0.0.1:${published_port}/v1/notification-status")"
    [[ "${notification_post_migrate_status}" == "200" ]] || fail "notification status after migration re-apply returned HTTP ${notification_post_migrate_status}"
    notification_migration_flow="down:${notification_tables_down}/up:${notification_tables_up}/http:${notification_post_migrate_status}"
    ;;
esac

case ",${OPTIONAL_STARTERS:-}," in
  *,setting,*)
    command -v python3 >/dev/null 2>&1 || fail "python3 is required for setting response checks"

    setting_public_status="$(curl --noproxy '*' --silent --show-error \
      --dump-header "${TMP_DIR}/setting-public.headers" \
      --output "${TMP_DIR}/setting-public.json" \
      --write-out '%{http_code}' \
      "http://127.0.0.1:${published_port}/v1/settings/public")"
    [[ "${setting_public_status}" == "200" ]] || fail "public setting list returned HTTP ${setting_public_status}"
    if ! python3 -c '
import json, sys
values = json.load(open(sys.argv[1]))["data"]
expected = [
    ("app", "branding.display_name", "Luas", 0, "default"),
    ("app", "localization.locale", "en-US", 0, "default"),
]
actual = [(item["scope"], item["key"], item["value"], item["version"], item["source"]) for item in values]
raise SystemExit(0 if actual == expected else 1)
' "${TMP_DIR}/setting-public.json"; then
      fail "public setting defaults violate the finite catalog contract"
    fi
    setting_public_etag="$(awk 'tolower($1) == "etag:" { gsub("\r", "", $2); print $2; exit }' "${TMP_DIR}/setting-public.headers")"
    [[ "${setting_public_etag}" =~ ^\"settings-[a-f0-9]{64}\"$ ]] || fail "public setting ETag is invalid: ${setting_public_etag}"
    setting_revalidate_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --header "If-None-Match: ${setting_public_etag}" \
      "http://127.0.0.1:${published_port}/v1/settings/public")"
    [[ "${setting_revalidate_status}" == "304" ]] || fail "public setting revalidation returned HTTP ${setting_revalidate_status}"

    if ! compose exec -T api /app/luas setting:list >"${TMP_DIR}/setting-list.log" 2>&1; then
      fail "setting:list failed"
    fi
    grep -q 'branding.display_name' "${TMP_DIR}/setting-list.log" || fail "setting:list omitted the branding definition"
    for writer in 1 2; do
      (
        set +e
        compose exec -T api /app/luas setting:set \
          --key=branding.display_name \
          '--value="Compose Luas"' \
          --expected-version=0 >"${TMP_DIR}/setting-set-${writer}.log" 2>&1
        printf '%s\n' "$?" >"${TMP_DIR}/setting-set-${writer}.status"
      ) &
    done
    wait
    setting_set_successes="$(awk '$1 == 0 { count++ } END { print count + 0 }' "${TMP_DIR}"/setting-set-*.status)"
    setting_set_conflicts="$(grep -il 'setting version conflict' "${TMP_DIR}"/setting-set-*.log | wc -l | tr -d ' ')"
    [[ "${setting_set_successes}" == "1" && "${setting_set_conflicts}" == "1" ]] ||
      fail "concurrent setting:set expected one success and one version conflict"
    grep -q 'version 1' "${TMP_DIR}"/setting-set-*.log || fail "setting:set did not report version 1"
    if grep -q 'Compose Luas' "${TMP_DIR}"/setting-set-*.log; then
      fail "setting:set output exposed the setting value"
    fi
    setting_cli_audit_count="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM audit_logs
WHERE actor_type = 'system'
  AND method = 'CLI'
  AND path = 'setting:set'
  AND target_id = 'app:0:branding.display_name'
  AND metadata::jsonb ->> 'key' = 'branding.display_name'
  AND NOT (metadata::jsonb ? 'value');
")"
    [[ "${setting_cli_audit_count}" == "1" ]] || fail "setting:set wrote ${setting_cli_audit_count} valid minimized audit entries"
    setting_public_after_set_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-public-after-set.json" \
      --write-out '%{http_code}' \
      "http://127.0.0.1:${published_port}/v1/settings/public")"
    [[ "${setting_public_after_set_status}" == "200" ]] || fail "public settings after CLI set returned HTTP ${setting_public_after_set_status}"
    if ! python3 -c '
import json, sys
item = json.load(open(sys.argv[1]))["data"][0]
raise SystemExit(0 if item["key"] == "branding.display_name" and item["value"] == "Compose Luas" and item["version"] == 1 and item["source"] == "override" else 1)
' "${TMP_DIR}/setting-public-after-set.json"; then
      fail "public settings did not expose the committed app override"
    fi

    read -r setting_user_id setting_user_token < <(
      register_and_login_user "compose_setting_user" "compose-setting-user@example.com" "setting-user"
    )
    setting_user_list_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-user-list.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${setting_user_token}" \
      "http://127.0.0.1:${published_port}/v1/settings/user")"
    [[ "${setting_user_list_status}" == "200" ]] || fail "user setting list returned HTTP ${setting_user_list_status}"
    if ! python3 -c '
import json, sys
values = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if [(item["key"], item["value"], item["version"]) for item in values] == [("localization.locale", "en-US", 0), ("localization.timezone", "UTC", 0)] else 1)
' "${TMP_DIR}/setting-user-list.json"; then
      fail "user setting defaults violate the finite catalog contract"
    fi
    setting_missing_precondition_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-missing-precondition.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${setting_user_token}" \
      --data '{"value":"zh-Hans"}' \
      "http://127.0.0.1:${published_port}/v1/settings/user/localization.locale")"
    [[ "${setting_missing_precondition_status}" == "428" ]] || fail "missing setting precondition returned HTTP ${setting_missing_precondition_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "SETTING.PRECONDITION_REQUIRED" else 1)' "${TMP_DIR}/setting-missing-precondition.json"; then
      fail "missing setting precondition returned the wrong error_code"
    fi
    setting_user_set_status="$(curl --noproxy '*' --silent --show-error \
      --dump-header "${TMP_DIR}/setting-user-set.headers" \
      --output "${TMP_DIR}/setting-user-set.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${setting_user_token}" \
      --header 'If-Match: "setting-v0"' \
      --data '{"value":"zh-Hans"}' \
      "http://127.0.0.1:${published_port}/v1/settings/user/localization.locale")"
    [[ "${setting_user_set_status}" == "200" ]] || fail "user setting update returned HTTP ${setting_user_set_status}"
    grep -qi '^etag: "setting-v1"' "${TMP_DIR}/setting-user-set.headers" || fail "user setting update omitted version ETag"
    setting_stale_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-stale.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${setting_user_token}" \
      --header 'If-Match: "setting-v0"' \
      --data '{"value":"en-US"}' \
      "http://127.0.0.1:${published_port}/v1/settings/user/localization.locale")"
    [[ "${setting_stale_status}" == "412" ]] || fail "stale setting update returned HTTP ${setting_stale_status}"
    if ! python3 -c 'import json, sys; raise SystemExit(0 if json.load(open(sys.argv[1]))["error_code"] == "SETTING.VERSION_CONFLICT" else 1)' "${TMP_DIR}/setting-stale.json"; then
      fail "stale setting update returned the wrong error_code"
    fi
    setting_user_reset_status="$(curl --noproxy '*' --silent --show-error \
      --dump-header "${TMP_DIR}/setting-user-reset.headers" \
      --output /dev/null \
      --write-out '%{http_code}' \
      --request DELETE \
      --header "Authorization: Bearer ${setting_user_token}" \
      --header 'If-Match: "setting-v1"' \
      "http://127.0.0.1:${published_port}/v1/settings/user/localization.locale")"
    [[ "${setting_user_reset_status}" == "204" ]] || fail "user setting reset returned HTTP ${setting_user_reset_status}"
    grep -qi '^etag: "setting-v2"' "${TMP_DIR}/setting-user-reset.headers" || fail "user setting reset omitted monotonic ETag"

    setting_timezone_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-timezone.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${setting_user_token}" \
      --header 'If-Match: "setting-v0"' \
      --data '{"value":"Europe/Dublin"}' \
      "http://127.0.0.1:${published_port}/v1/settings/user/localization.timezone")"
    [[ "${setting_timezone_status}" == "200" ]] || fail "user timezone setting returned HTTP ${setting_timezone_status}"
    setting_rows_before_delete="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM settings WHERE scope = 'user' AND user_id = ${setting_user_id};")"
    [[ "${setting_rows_before_delete}" == "2" ]] || fail "user setting history has ${setting_rows_before_delete}/2 rows before account deletion"
    setting_account_delete_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --request DELETE \
      --header "Authorization: Bearer ${setting_user_token}" \
      "http://127.0.0.1:${published_port}/v1/users/account")"
    [[ "${setting_account_delete_status}" == "204" ]] || fail "setting user account deletion returned HTTP ${setting_account_delete_status}"
    setting_rows_after_delete="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM settings WHERE scope = 'user' AND user_id = ${setting_user_id};")"
    [[ "${setting_rows_after_delete}" == "0" ]] || fail "account deletion left ${setting_rows_after_delete} user setting row(s)"
    setting_account_cleanup_flow="before:${setting_rows_before_delete}/delete:${setting_account_delete_status}/after:${setting_rows_after_delete}"

    read -r setting_owner_id setting_owner_token < <(
      register_and_login_user "compose_setting_owner" "compose-setting-owner@example.com" "setting-owner"
    )
    [[ -n "${setting_owner_id}" ]] || fail "setting organization owner ID is missing"
    setting_organization_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-organization.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${setting_owner_token}" \
      --data '{"name":"Compose Setting Organization","slug":"compose-setting-organization"}' \
      "http://127.0.0.1:${published_port}/v1/organizations")"
    [[ "${setting_organization_status}" == "201" ]] || fail "setting organization creation returned HTTP ${setting_organization_status}"
    setting_organization_id="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["data"]["id"])' "${TMP_DIR}/setting-organization.json")"
    setting_organization_set_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-organization-set.json" \
      --write-out '%{http_code}' \
      --request PATCH \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${setting_owner_token}" \
      --header "Organization-Id: ${setting_organization_id}" \
      --header 'If-Match: "setting-v0"' \
      --data '{"value":"zh-Hans"}' \
      "http://127.0.0.1:${published_port}/v1/organization-settings/localization.locale")"
    [[ "${setting_organization_set_status}" == "200" ]] || fail "organization setting update returned HTTP ${setting_organization_set_status}"
    if ! python3 -c '
import json, sys
item = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if item["scope"] == "organization" and item["value"] == "zh-Hans" and item["version"] == 1 else 1)
' "${TMP_DIR}/setting-organization-set.json"; then
      fail "organization setting update returned an invalid effective value"
    fi

    if ! compose exec -T api /app/luas setting:reset \
      --key=branding.display_name \
      --expected-version=1 >"${TMP_DIR}/setting-reset.log" 2>&1; then
      fail "setting:reset failed"
    fi
    setting_flow="public:${setting_public_status}/${setting_revalidate_status}/cli-cas:${setting_set_successes}/${setting_set_conflicts}/cli-audit:${setting_cli_audit_count}/user:${setting_user_list_status}/${setting_missing_precondition_status}/${setting_user_set_status}/${setting_stale_status}/${setting_user_reset_status}/organization:${setting_organization_set_status}"

    if ! compose exec -T api /app/luas db:rollback --step=1 >"${TMP_DIR}/setting-rollback.log" 2>&1; then
      fail "setting migration rollback failed"
    fi
    setting_tables_down="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'settings';")"
    [[ "${setting_tables_down}" == "0" ]] || fail "setting migration rollback left ${setting_tables_down} table(s)"
    if ! compose exec -T api /app/luas db:migrate >"${TMP_DIR}/setting-migrate.log" 2>&1; then
      fail "setting migration re-apply failed"
    fi
    setting_tables_up="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'settings';")"
    [[ "${setting_tables_up}" == "1" ]] || fail "setting migration re-apply created ${setting_tables_up}/1 tables"
    setting_post_migrate_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/setting-post-migrate.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${setting_owner_token}" \
      "http://127.0.0.1:${published_port}/v1/settings/user")"
    [[ "${setting_post_migrate_status}" == "200" ]] || fail "setting list after migration re-apply returned HTTP ${setting_post_migrate_status}"
    setting_migration_flow="down:${setting_tables_down}/up:${setting_tables_up}/http:${setting_post_migrate_status}"
    ;;
esac

case ",${OPTIONAL_STARTERS:-}," in
  *,usage,*)
    command -v python3 >/dev/null 2>&1 || fail "python3 is required for usage response checks"

    read -r usage_user_id usage_user_token < <(
      register_and_login_user "compose_usage_user" "compose-usage-user@example.com" "usage-user"
    )
    [[ -n "${usage_user_id}" && -n "${usage_user_token}" ]] || fail "usage user login response is incomplete"

    if ! compose exec -T api /app/luas usage:quota:set \
      --scope=user \
      --subject-id="${usage_user_id}" \
      --metric=api.requests \
      --limit=3 \
      --expected-version=0 >"${TMP_DIR}/usage-quota-set.log" 2>&1; then
      fail "usage:quota:set failed"
    fi
    grep -q 'version 1' "${TMP_DIR}/usage-quota-set.log" || fail "usage:quota:set did not report version 1"

    for replay in 1 2; do
      (
        set +e
        compose exec -T api /app/luas usage:consume \
          --scope=user \
          --subject-id="${usage_user_id}" \
          --metric=api.requests \
          --quantity=2 \
          --source=compose.concurrent \
          --event-id=exact-replay >"${TMP_DIR}/usage-replay-${replay}.log" 2>&1
        printf '%s\n' "$?" >"${TMP_DIR}/usage-replay-${replay}.status"
      ) &
    done
    wait
    usage_replay_successes="$(awk '$1 == 0 { count++ } END { print count + 0 }' "${TMP_DIR}"/usage-replay-*.status)"
    [[ "${usage_replay_successes}" == "2" ]] || fail "concurrent exact usage replay returned ${usage_replay_successes}/2 successes"
    usage_exact_receipts="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM usage_events WHERE source = 'compose.concurrent' AND event_id = 'exact-replay';")"
    usage_counter_after_replay="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT value FROM usage_counters WHERE scope = 'user' AND subject_id = ${usage_user_id} AND metric = 'api.requests';")"
    [[ "${usage_exact_receipts}" == "1" && "${usage_counter_after_replay}" == "2" ]] ||
      fail "exact usage replay produced ${usage_exact_receipts} receipt(s) and counter ${usage_counter_after_replay}"

    set +e
    compose exec -T api /app/luas usage:consume \
      --scope=user \
      --subject-id="${usage_user_id}" \
      --metric=api.requests \
      --quantity=1 \
      --source=compose.concurrent \
      --event-id=exact-replay >"${TMP_DIR}/usage-conflict.log" 2>&1
    usage_conflict_status="$?"
    set -e
    [[ "${usage_conflict_status}" != "0" ]] || fail "usage idempotency conflict unexpectedly succeeded"
    grep -qi 'usage event idempotency conflict' "${TMP_DIR}/usage-conflict.log" || fail "usage idempotency conflict returned the wrong error"
    usage_receipts_after_conflict="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM usage_events WHERE source = 'compose.concurrent' AND event_id = 'exact-replay';")"
    usage_counter_after_conflict="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT value FROM usage_counters WHERE scope = 'user' AND subject_id = ${usage_user_id} AND metric = 'api.requests';")"
    [[ "${usage_receipts_after_conflict}" == "1" && "${usage_counter_after_conflict}" == "2" ]] ||
      fail "usage idempotency conflict changed receipt count or counter"

    for consumer in 1 2 3 4; do
      (
        set +e
        compose exec -T api /app/luas usage:consume \
          --scope=user \
          --subject-id="${usage_user_id}" \
          --metric=api.requests \
          --quantity=1 \
          --source=compose.quota \
          --event-id="consumer-${consumer}" >"${TMP_DIR}/usage-consumer-${consumer}.log" 2>&1
        printf '%s\n' "$?" >"${TMP_DIR}/usage-consumer-${consumer}.status"
      ) &
    done
    wait
    usage_consume_successes="$(awk '$1 == 0 { count++ } END { print count + 0 }' "${TMP_DIR}"/usage-consumer-*.status)"
    usage_consume_denials="$(grep -il 'usage quota exceeded' "${TMP_DIR}"/usage-consumer-*.log | wc -l | tr -d ' ')"
    [[ "${usage_consume_successes}" == "1" && "${usage_consume_denials}" == "3" ]] ||
      fail "concurrent usage quota expected one success and three denials"
    usage_final_counter="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT value FROM usage_counters WHERE scope = 'user' AND subject_id = ${usage_user_id} AND metric = 'api.requests';")"
    usage_denied_receipts="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM usage_events WHERE scope = 'user' AND subject_id = ${usage_user_id} AND decision = 'denied';")"
    [[ "${usage_final_counter}" == "3" && "${usage_denied_receipts}" == "3" ]] ||
      fail "usage quota ended at counter ${usage_final_counter} with ${usage_denied_receipts} denied receipt(s)"

    usage_user_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/usage-user.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${usage_user_token}" \
      "http://127.0.0.1:${published_port}/v1/usage/user")"
    [[ "${usage_user_status}" == "200" ]] || fail "user usage list returned HTTP ${usage_user_status}"
    if ! python3 -c '
import json, sys
values = json.load(open(sys.argv[1]))["data"]
expected = {"api.requests", "ai.input_tokens", "ai.output_tokens", "asset.transfer_bytes", "workflow.runs"}
item = next(value for value in values if value["metric"] == "api.requests")
forbidden = {"event_id", "source", "fingerprint", "dimensions"}
valid = (
    len(values) == 5
    and {value["metric"] for value in values} == expected
    and item["used"] == 3
    and item["limit"] == 3
    and item["remaining"] == 0
    and item["quota_source"] == "override"
    and not any(forbidden.intersection(value) for value in values)
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/usage-user.json"; then
      fail "user usage summary violates finite catalog, quota, or privacy semantics"
    fi

    read -r usage_owner_id usage_owner_token < <(
      register_and_login_user "compose_usage_owner" "compose-usage-owner@example.com" "usage-owner"
    )
    usage_organization_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/usage-organization-create.json" \
      --write-out '%{http_code}' \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${usage_owner_token}" \
      --data '{"name":"Compose Usage Organization","slug":"compose-usage-organization"}' \
      "http://127.0.0.1:${published_port}/v1/organizations")"
    [[ "${usage_organization_status}" == "201" ]] || fail "usage organization creation returned HTTP ${usage_organization_status}"
    usage_organization_id="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["data"]["id"])' "${TMP_DIR}/usage-organization-create.json")"
    usage_organization_list_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/usage-organization.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${usage_owner_token}" \
      --header "Organization-Id: ${usage_organization_id}" \
      "http://127.0.0.1:${published_port}/v1/organization-usage")"
    [[ "${usage_organization_list_status}" == "200" ]] || fail "organization usage list returned HTTP ${usage_organization_list_status}"
    if ! python3 -c 'import json, sys; values = json.load(open(sys.argv[1]))["data"]; raise SystemExit(0 if len(values) == 5 and all(value["scope"] == "organization" and value["used"] == 0 for value in values) else 1)' "${TMP_DIR}/usage-organization.json"; then
      fail "organization usage defaults violate scope or finite catalog semantics"
    fi

    usage_rows_before_delete="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT (SELECT COUNT(*) FROM usage_events WHERE user_id = ${usage_user_id}) + (SELECT COUNT(*) FROM usage_counters WHERE user_id = ${usage_user_id}) + (SELECT COUNT(*) FROM usage_quotas WHERE user_id = ${usage_user_id});")"
    [[ "${usage_rows_before_delete}" -gt "0" ]] || fail "usage account cleanup fixture has no owned rows"
    usage_account_delete_status="$(curl --noproxy '*' --silent --show-error \
      --output /dev/null \
      --write-out '%{http_code}' \
      --request DELETE \
      --header "Authorization: Bearer ${usage_user_token}" \
      "http://127.0.0.1:${published_port}/v1/users/account")"
    [[ "${usage_account_delete_status}" == "204" ]] || fail "usage user account deletion returned HTTP ${usage_account_delete_status}"
    usage_rows_after_delete="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT (SELECT COUNT(*) FROM usage_events WHERE user_id = ${usage_user_id}) + (SELECT COUNT(*) FROM usage_counters WHERE user_id = ${usage_user_id}) + (SELECT COUNT(*) FROM usage_quotas WHERE user_id = ${usage_user_id});")"
    [[ "${usage_rows_after_delete}" == "0" ]] || fail "account deletion left ${usage_rows_after_delete} user usage row(s)"
    usage_account_cleanup_flow="before:${usage_rows_before_delete}/delete:${usage_account_delete_status}/after:${usage_rows_after_delete}"

    if ! compose exec -T api /app/luas usage:prune \
      --before=2000-01-01T00:00:00Z >"${TMP_DIR}/usage-prune.log" 2>&1; then
      fail "usage:prune failed"
    fi
    usage_flow="replay:${usage_replay_successes}/receipts:${usage_exact_receipts}/conflict:${usage_conflict_status}/quota:${usage_consume_successes}/${usage_consume_denials}/counter:${usage_final_counter}/http:${usage_user_status}/${usage_organization_list_status}"

    if ! compose exec -T api /app/luas db:rollback --step=1 >"${TMP_DIR}/usage-rollback.log" 2>&1; then
      fail "usage migration rollback failed"
    fi
    usage_tables_down="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('usage_events', 'usage_counters', 'usage_quotas');")"
    [[ "${usage_tables_down}" == "0" ]] || fail "usage migration rollback left ${usage_tables_down} table(s)"
    if ! compose exec -T api /app/luas db:migrate >"${TMP_DIR}/usage-migrate.log" 2>&1; then
      fail "usage migration re-apply failed"
    fi
    usage_tables_up="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('usage_events', 'usage_counters', 'usage_quotas');")"
    [[ "${usage_tables_up}" == "3" ]] || fail "usage migration re-apply created ${usage_tables_up}/3 tables"
    usage_post_migrate_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/usage-post-migrate.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${usage_owner_token}" \
      "http://127.0.0.1:${published_port}/v1/usage/user")"
    [[ "${usage_post_migrate_status}" == "200" ]] || fail "usage list after migration re-apply returned HTTP ${usage_post_migrate_status}"
    usage_migration_flow="down:${usage_tables_down}/up:${usage_tables_up}/http:${usage_post_migrate_status}"
    ;;
esac

case ",${OPTIONAL_STARTERS:-}," in
  *,asset,*)
    command -v python3 >/dev/null 2>&1 || fail "python3 is required for asset response checks"
    command -v cmp >/dev/null 2>&1 || fail "cmp is required for asset byte checks"

    read -r asset_user_id asset_token < <(
      register_and_login_user "compose_asset_owner" "compose-asset-owner@example.com" "asset-owner"
    )
    [[ -n "${asset_user_id}" && -n "${asset_token}" ]] || fail "asset owner login response is incomplete"

    printf 'Luas asset compose check\n' >"${TMP_DIR}/asset-upload.txt"
    asset_size="$(wc -c <"${TMP_DIR}/asset-upload.txt" | tr -d ' ')"
    asset_intent_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-intent.json" \
      --write-out '%{http_code}' \
      --request POST \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer ${asset_token}" \
      --data "{\"idempotency_key\":\"compose-asset-upload\",\"original_name\":\"compose-check.txt\",\"media_type\":\"text/plain\",\"size_bytes\":${asset_size}}" \
      "http://127.0.0.1:${published_port}/v1/assets/upload-intents")"
    [[ "${asset_intent_status}" == "201" ]] || fail "asset upload intent returned HTTP ${asset_intent_status}"
    if ! read -r asset_id asset_upload_path < <(python3 -c '
import json, sys
from urllib.parse import urlsplit
payload = json.load(open(sys.argv[1]))["data"]
parts = urlsplit(payload["upload"]["url"])
path = parts.path + (("?" + parts.query) if parts.query else "")
print(payload["asset"]["id"], path)
' "${TMP_DIR}/asset-intent.json"); then
      fail "asset upload intent response is incomplete"
    fi

    asset_upload_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-upload-response.json" \
      --write-out '%{http_code}' \
      --request PUT \
      --header 'Content-Type: text/plain' \
      --data-binary "@${TMP_DIR}/asset-upload.txt" \
      "http://127.0.0.1:${published_port}${asset_upload_path}")"
    [[ "${asset_upload_status}" == "204" ]] || fail "asset byte upload returned HTTP ${asset_upload_status}"

    asset_complete_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-complete.json" \
      --write-out '%{http_code}' \
      --request POST \
      --header "Authorization: Bearer ${asset_token}" \
      "http://127.0.0.1:${published_port}/v1/assets/${asset_id}/complete")"
    [[ "${asset_complete_status}" == "200" ]] || fail "asset completion returned HTTP ${asset_complete_status}"
    if ! python3 -c '
import json, sys
asset = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if asset["status"] == "ready" and asset["ready_at"] else 1)
' "${TMP_DIR}/asset-complete.json"; then
      fail "asset completion did not return a ready asset"
    fi

    asset_grant_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-grant.json" \
      --write-out '%{http_code}' \
      --request POST \
      --header "Authorization: Bearer ${asset_token}" \
      "http://127.0.0.1:${published_port}/v1/assets/${asset_id}/download-grant")"
    [[ "${asset_grant_status}" == "200" ]] || fail "asset download grant returned HTTP ${asset_grant_status}"
    if ! asset_download_path="$(python3 -c '
import json, sys
from urllib.parse import urlsplit
parts = urlsplit(json.load(open(sys.argv[1]))["data"]["url"])
print(parts.path + (("?" + parts.query) if parts.query else ""))
' "${TMP_DIR}/asset-grant.json")"; then
      fail "asset download grant response is incomplete"
    fi
    asset_download_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-download.txt" \
      --write-out '%{http_code}' \
      "http://127.0.0.1:${published_port}${asset_download_path}")"
    [[ "${asset_download_status}" == "200" ]] || fail "asset download returned HTTP ${asset_download_status}"
    cmp --silent "${TMP_DIR}/asset-upload.txt" "${TMP_DIR}/asset-download.txt" || fail "asset download bytes differ from uploaded bytes"

    asset_delete_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-delete.json" \
      --write-out '%{http_code}' \
      --request DELETE \
      --header "Authorization: Bearer ${asset_token}" \
      "http://127.0.0.1:${published_port}/v1/assets/${asset_id}")"
    [[ "${asset_delete_status}" == "204" ]] || fail "asset deletion returned HTTP ${asset_delete_status}"
    asset_deleted_grant_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-deleted-grant.json" \
      --write-out '%{http_code}' \
      --request POST \
      --header "Authorization: Bearer ${asset_token}" \
      "http://127.0.0.1:${published_port}/v1/assets/${asset_id}/download-grant")"
    [[ "${asset_deleted_grant_status}" == "404" ]] || fail "deleted asset download grant returned HTTP ${asset_deleted_grant_status}"
    if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))
raise SystemExit(0 if payload.get("error_code") == "ASSET.NOT_FOUND" else 1)
' "${TMP_DIR}/asset-deleted-grant.json"; then
      fail "deleted asset did not preserve ASSET.NOT_FOUND non-disclosure"
    fi
    asset_flow="${asset_intent_status}/${asset_upload_status}/${asset_complete_status}/${asset_grant_status}/${asset_download_status}/${asset_delete_status}/${asset_deleted_grant_status}"

    read -r asset_race_user_id asset_race_token < <(
      register_and_login_user "compose_asset_race" "compose-asset-race@example.com" "asset-race"
    )
    (
      curl --noproxy '*' --silent --show-error \
        --output "${TMP_DIR}/asset-race-create.json" \
        --write-out '%{http_code}' \
        --request POST \
        --header 'Content-Type: application/json' \
        --header "Authorization: Bearer ${asset_race_token}" \
        --data '{"idempotency_key":"asset-account-race","original_name":"race.txt","media_type":"text/plain","size_bytes":4}' \
        "http://127.0.0.1:${published_port}/v1/assets/upload-intents" \
        >"${TMP_DIR}/asset-race-create.status"
    ) &
    asset_race_create_pid=$!
    (
      curl --noproxy '*' --silent --show-error \
        --output "${TMP_DIR}/asset-race-delete.json" \
        --write-out '%{http_code}' \
        --request DELETE \
        --header "Authorization: Bearer ${asset_race_token}" \
        "http://127.0.0.1:${published_port}/v1/users/account" \
        >"${TMP_DIR}/asset-race-delete.status"
    ) &
    asset_race_delete_pid=$!
    wait "${asset_race_create_pid}"
    wait "${asset_race_delete_pid}"
    asset_race_create_status="$(cat "${TMP_DIR}/asset-race-create.status")"
    asset_race_delete_status="$(cat "${TMP_DIR}/asset-race-delete.status")"
    if [[ "${asset_race_create_status}/${asset_race_delete_status}" != "201/409" && \
          "${asset_race_create_status}/${asset_race_delete_status}" != "404/204" ]]; then
      fail "asset/account race returned ${asset_race_create_status}/${asset_race_delete_status}"
    fi
    orphaned_assets="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM assets AS a
JOIN users AS u ON u.id = a.user_id
WHERE a.deleted_at IS NULL AND u.deleted_at IS NOT NULL;
")"
    [[ "${orphaned_assets}" == "0" ]] || fail "concurrent account deletion left ${orphaned_assets} active orphan asset(s)"
    asset_account_race_flow="${asset_race_create_status}/${asset_race_delete_status}/orphans:${orphaned_assets}/user:${asset_race_user_id}"

    if ! compose exec -T api /app/luas db:rollback --step=1 >"${TMP_DIR}/asset-rollback.log" 2>&1; then
      fail "asset migration rollback failed"
    fi
    asset_tables_down="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'assets';
")"
    [[ "${asset_tables_down}" == "0" ]] || fail "asset migration rollback left ${asset_tables_down} table(s)"
    if ! compose exec -T api /app/luas db:migrate >"${TMP_DIR}/asset-migrate.log" 2>&1; then
      fail "asset migration re-apply failed"
    fi
    asset_tables_up="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'assets';
")"
    [[ "${asset_tables_up}" == "1" ]] || fail "asset migration re-apply created ${asset_tables_up}/1 tables"
    asset_post_migrate_status="$(curl --noproxy '*' --silent --show-error \
      --output "${TMP_DIR}/asset-post-migrate.json" \
      --write-out '%{http_code}' \
      --header "Authorization: Bearer ${asset_token}" \
      "http://127.0.0.1:${published_port}/v1/assets")"
    [[ "${asset_post_migrate_status}" == "200" ]] || fail "asset list after migration re-apply returned HTTP ${asset_post_migrate_status}"
    asset_migration_flow="down:${asset_tables_down}/up:${asset_tables_up}/http:${asset_post_migrate_status}"
    ;;
esac

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

    case ",${OPTIONAL_STARTERS:-}," in
      *,webhook,*)
        webhook_catalog_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-catalog.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/webhook-event-types")"
        [[ "${webhook_catalog_status}" == "200" ]] || fail "webhook event catalog returned HTTP ${webhook_catalog_status}"
        if ! python3 -c '
import json, sys
items = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if items == ["webhook.test"] else 1)
' "${TMP_DIR}/webhook-catalog.json"; then
          fail "webhook event catalog is not finite or contains the wrong starter event"
        fi

        webhook_create_status="$(curl --noproxy '*' --silent --show-error \
          --dump-header "${TMP_DIR}/webhook-create.headers" \
          --output "${TMP_DIR}/webhook-create.json" \
          --write-out '%{http_code}' \
          --header 'Content-Type: application/json' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          --data '{"name":"Compose local receiver","url":"http://127.0.0.1:8025/health/live","event_types":["webhook.test"]}' \
          "http://127.0.0.1:${published_port}/v1/webhook-endpoints")"
        [[ "${webhook_create_status}" == "201" ]] || fail "webhook endpoint creation returned HTTP ${webhook_create_status}"
        if ! read -r webhook_endpoint_id webhook_secret webhook_version < <(python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))["data"]
endpoint = payload["endpoint"]
secret = payload["signing_secret"]
forbidden = {"secret_ciphertext", "previous_secret_ciphertext"}
if not secret.startswith("whsec_") or forbidden.intersection(endpoint):
    raise SystemExit(1)
print(endpoint["id"], secret, endpoint["version"])
' "${TMP_DIR}/webhook-create.json"); then
          fail "webhook endpoint creation violates one-time secret or ciphertext contract"
        fi
        if ! grep -qi '^etag: "webhook-endpoint-v1"' "${TMP_DIR}/webhook-create.headers"; then
          fail "webhook endpoint creation did not return the canonical ETag"
        fi

        webhook_list_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-list.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/webhook-endpoints")"
        [[ "${webhook_list_status}" == "200" ]] || fail "webhook endpoint list returned HTTP ${webhook_list_status}"
        if ! WEBHOOK_SECRET="${webhook_secret}" python3 -c '
import json, os, sys
raw = open(sys.argv[1], encoding="utf-8").read()
payload = json.loads(raw)
item = payload["data"][0]
forbidden = {"signing_secret", "secret_ciphertext", "previous_secret_ciphertext"}
valid = (
    len(payload["data"]) == 1
    and item["id"] == int(sys.argv[2])
    and item["organization_id"] == int(sys.argv[3])
    and item["event_types"] == ["webhook.test"]
    and not forbidden.intersection(item)
    and os.environ["WEBHOOK_SECRET"] not in raw
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/webhook-list.json" "${webhook_endpoint_id}" "${organization_id}"; then
          fail "webhook endpoint list leaked secret material or crossed organization scope"
        fi

        webhook_test_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-test.json" \
          --write-out '%{http_code}' \
          --request POST \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          --header 'Idempotency-Key: compose:webhook:test:1' \
          "http://127.0.0.1:${published_port}/v1/webhook-endpoints/${webhook_endpoint_id}/tests")"
        [[ "${webhook_test_status}" == "202" ]] || fail "webhook endpoint test returned HTTP ${webhook_test_status}"
        if ! read -r webhook_delivery_id webhook_message_id < <(python3 -c '
import json, sys
item = json.load(open(sys.argv[1]))["data"]
if item["status"] != "pending" or item["event_type"] != "webhook.test" or not item["message_id"].startswith("msg_"):
    raise SystemExit(1)
print(item["id"], item["message_id"])
' "${TMP_DIR}/webhook-test.json"); then
          fail "webhook endpoint test did not return one pending canonical delivery"
        fi

        webhook_replay_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-test-replay.json" \
          --write-out '%{http_code}' \
          --request POST \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          --header 'Idempotency-Key: compose:webhook:test:1' \
          "http://127.0.0.1:${published_port}/v1/webhook-endpoints/${webhook_endpoint_id}/tests")"
        [[ "${webhook_replay_status}" == "202" ]] || fail "idempotent webhook endpoint test returned HTTP ${webhook_replay_status}"
        if ! python3 -c '
import json, sys
item = json.load(open(sys.argv[1]))["data"]
raise SystemExit(0 if item["id"] == int(sys.argv[2]) and item["message_id"] == sys.argv[3] else 1)
' "${TMP_DIR}/webhook-test-replay.json" "${webhook_delivery_id}" "${webhook_message_id}"; then
          fail "idempotent webhook endpoint test created a duplicate delivery or message ID"
        fi

        if ! compose exec -T api /app/luas webhook:work --batch=10 --once >"${TMP_DIR}/webhook-worker.log" 2>&1; then
          fail "webhook worker batch failed"
        fi
        webhook_delivery_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-deliveries.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/webhook-deliveries?endpoint_id=${webhook_endpoint_id}")"
        [[ "${webhook_delivery_status}" == "200" ]] || fail "webhook delivery list returned HTTP ${webhook_delivery_status}"
        if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))
item = payload["data"][0]
forbidden = {"url", "payload", "payload_json", "signature", "response_body", "error"}
valid = (
    len(payload["data"]) == 1
    and item["id"] == int(sys.argv[2])
    and item["message_id"] == sys.argv[3]
    and item["status"] == "failed"
    and item["attempt_count"] == 1
    and item["http_status"] == 404
    and item["failure_code"] == "WEBHOOK.HTTP_404"
    and not forbidden.intersection(item)
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/webhook-deliveries.json" "${webhook_delivery_id}" "${webhook_message_id}"; then
          fail "webhook terminal delivery violates status, identity, or privacy contract"
        fi

        webhook_attempt_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-attempts.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/webhook-deliveries/${webhook_delivery_id}/attempts")"
        [[ "${webhook_attempt_status}" == "200" ]] || fail "webhook attempt list returned HTTP ${webhook_attempt_status}"
        if ! python3 -c '
import json, sys
payload = json.load(open(sys.argv[1]))
item = payload["data"][0]
forbidden = {"url", "payload", "payload_json", "signature", "response_body", "error", "error_message"}
valid = (
    len(payload["data"]) == 1
    and item["delivery_id"] == int(sys.argv[2])
    and item["number"] == 1
    and item["outcome"] == "failed"
    and item["http_status"] == 404
    and item["failure_code"] == "WEBHOOK.HTTP_404"
    and not forbidden.intersection(item)
)
raise SystemExit(0 if valid else 1)
' "${TMP_DIR}/webhook-attempts.json" "${webhook_delivery_id}"; then
          fail "webhook attempt ledger violates minimized outcome contract"
        fi

        webhook_secret_rows="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*) FROM webhook_endpoints
WHERE id = ${webhook_endpoint_id}
  AND secret_ciphertext NOT LIKE 'whsec_%'
  AND length(secret_ciphertext) > 32;
")"
        [[ "${webhook_secret_rows}" == "1" ]] || fail "webhook signing secret is not encrypted at rest"
        webhook_forbidden_columns="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name IN ('webhook_deliveries', 'webhook_delivery_attempts')
  AND column_name IN ('url', 'payload', 'payload_json', 'signature', 'response_body', 'error', 'error_message');
")"
        [[ "${webhook_forbidden_columns}" == "0" ]] || fail "webhook ledger contains ${webhook_forbidden_columns} forbidden sensitive column(s)"
        webhook_query_indexes="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM pg_indexes
WHERE schemaname = 'public'
  AND (
    (indexname = 'idx_webhook_endpoints_organization_status' AND indexdef LIKE '%(organization_id, status, id)%')
    OR (indexname = 'idx_webhook_endpoints_organization_created' AND indexdef LIKE '%(organization_id, created_at, id)%')
    OR (indexname = 'idx_webhook_events_created' AND indexdef LIKE '%(created_at, id)%')
    OR (indexname = 'idx_webhook_deliveries_due' AND indexdef LIKE '%(status, available_at, id)%')
    OR (indexname = 'idx_webhook_deliveries_lease_expiry' AND indexdef LIKE '%(status, lease_expires_at)%')
    OR (indexname = 'idx_webhook_deliveries_organization_endpoint_created' AND indexdef LIKE '%(organization_id, endpoint_id, created_at, id)%')
    OR (indexname = 'idx_webhook_deliveries_organization_status_created' AND indexdef LIKE '%(organization_id, status, created_at, id)%')
  );
")"
        [[ "${webhook_query_indexes}" == "7" ]] || fail "webhook schema has ${webhook_query_indexes}/7 query-shaped indexes"
        webhook_flow="${webhook_catalog_status}/${webhook_create_status}/${webhook_list_status}/${webhook_test_status}/${webhook_replay_status}/${webhook_delivery_status}/${webhook_attempt_status}:failed:404:private:${webhook_forbidden_columns}:indexes:${webhook_query_indexes}"

        if ! compose exec -T api /app/luas db:rollback --step=1 >"${TMP_DIR}/webhook-rollback.log" 2>&1; then
          fail "webhook migration rollback failed"
        fi
        webhook_tables_down="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('webhook_endpoints', 'webhook_subscriptions', 'webhook_events', 'webhook_deliveries', 'webhook_delivery_attempts');
")"
        [[ "${webhook_tables_down}" == "0" ]] || fail "webhook migration rollback left ${webhook_tables_down} table(s)"
        if ! compose exec -T api /app/luas db:migrate >"${TMP_DIR}/webhook-migrate.log" 2>&1; then
          fail "webhook migration re-apply failed"
        fi
        webhook_tables_up="$(compose exec -T postgres psql --username luas --dbname luas --tuples-only --no-align --command "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('webhook_endpoints', 'webhook_subscriptions', 'webhook_events', 'webhook_deliveries', 'webhook_delivery_attempts');
")"
        [[ "${webhook_tables_up}" == "5" ]] || fail "webhook migration re-apply created ${webhook_tables_up}/5 tables"
        webhook_post_migrate_status="$(curl --noproxy '*' --silent --show-error \
          --output "${TMP_DIR}/webhook-post-migrate.json" \
          --write-out '%{http_code}' \
          --header "Authorization: Bearer ${owner_token}" \
          --header "Organization-Id: ${organization_id}" \
          "http://127.0.0.1:${published_port}/v1/webhook-endpoints")"
        [[ "${webhook_post_migrate_status}" == "200" ]] || fail "webhook endpoint list after migration re-apply returned HTTP ${webhook_post_migrate_status}"
        webhook_migration_flow="down:${webhook_tables_down}/up:${webhook_tables_up}/http:${webhook_post_migrate_status}"
        ;;
    esac

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
printf 'compose authentication login/profile/logout/revoked: %s\n' "${authentication_flow}"
printf 'compose organization/invitation flow: %s\n' "${organization_flow}"
printf 'compose organization/context flow: %s\n' "${organization_context_flow}"
printf 'compose organization/member flow: %s\n' "${membership_flow}"
printf 'compose permission flow: %s\n' "${permission_flow}"
printf 'compose permission migration flow: %s\n' "${permission_migration_flow}"
printf 'compose notification flow: %s\n' "${notification_flow}"
printf 'compose notification migration flow: %s\n' "${notification_migration_flow}"
printf 'compose asset flow: %s\n' "${asset_flow}"
printf 'compose asset migration flow: %s\n' "${asset_migration_flow}"
printf 'compose asset/account race: %s\n' "${asset_account_race_flow}"
printf 'compose setting flow: %s\n' "${setting_flow}"
printf 'compose setting migration flow: %s\n' "${setting_migration_flow}"
printf 'compose setting/account cleanup: %s\n' "${setting_account_cleanup_flow}"
printf 'compose usage flow: %s\n' "${usage_flow}"
printf 'compose usage migration flow: %s\n' "${usage_migration_flow}"
printf 'compose usage/account cleanup: %s\n' "${usage_account_cleanup_flow}"
printf 'compose webhook flow: %s\n' "${webhook_flow}"
printf 'compose webhook migration flow: %s\n' "${webhook_migration_flow}"
printf 'compose organization/account race: %s\n' "${account_race_flow}"

# Platform Operator Contracts

The optional `operator` starter lets platform operators run the deployment from the static Admin
Console. It depends on the default `user` and `audit` starters and is selected with
`OPTIONAL_STARTERS=operator`. When it is not selected, no `/v1/operator` route and no `operator:*`
command exists.

Implementation plans: [`admin-operator-console.md`](../docs/plans/admin-operator-console.md) and
[`admin-operator-starters.md`](../docs/plans/admin-operator-starters.md).

## Operators

A platform operator is a user with a current operator grant. Grants are created and removed only
from the CLI on a host with database access:

```bash
luas operator:grant <email>
luas operator:revoke <email>
luas operator:list
```

Granting or revoking twice is a no-op. Each change writes an audit record with the system as actor
and the account's user ID as target. Deleting an account removes its grant. A grant is not an
organization role and not a permission key.

## Browser Session

The Admin Console has no server runtime, so the Go API issues the operator session directly to the
Admin origin. The console enables sign-in when built with `VITE_OPTIONAL_FEATURES=operator`. Deployments must route the Admin origin's `/api/*` prefix to the Go API on the same
origin so the cookie is first-party.

| Rule | Value |
|---|---|
| Cookie name | `OPERATOR_SESSION_COOKIE_NAME`; default `__Host-luas_operator` in production, `luas_operator` otherwise |
| Cookie attributes | HttpOnly, `Secure` in production, `SameSite=Strict`, `Path=/`, no `Domain`, `Max-Age` = session lifetime |
| Credential | The opaque authentication session from the `user` starter; never returned in a response body |
| Session lifetime | The existing absolute and idle session limits |
| Unsafe methods | Require `Origin` exactly equal to one of `OPERATOR_ALLOWED_ORIGINS` and header `X-CSRF-Token` |
| CSRF token | Returned by sign-in and `GET /v1/operator/session`; bound to one session |
| Caching | Every `/v1/operator` response sends `Cache-Control: private, no-store` |

Each protected request re-checks the session, the account status, and the operator grant against
current persistence, so disabling an account or revoking a grant takes effect on the next request.
Unsafe operator requests are recorded by the audit middleware with the operator as actor.

### Endpoints

| Operation | Method and path | Request | Success |
|---|---|---|---|
| Sign in | `POST /v1/operator/session` | `Origin`; `{ identifier, password }` | `200 { operator, csrf_token }`, sets the cookie |
| Current operator | `GET /v1/operator/session` | Cookie | `200 { operator, csrf_token }` |
| Sign out | `DELETE /v1/operator/session` | Cookie, `Origin`, `X-CSRF-Token` | `204`, expires the cookie |

`operator` is `{ id, username, email, nickname }`. `identifier` is a username or email of at most
100 characters; `password` is at most 128 characters.

Sign-in checks the operator grant after verifying credentials and before creating a session, so a
non-operator never receives a session. It shares the public login per-IP and per-account quotas.
Unknown, wrong, and disabled accounts all return `AUTH.INVALID_CREDENTIALS`.

Sign-out without a cookie, or with an already revoked session, returns `204` and expires the cookie.

### Errors

| Condition | HTTP | `error_code` |
|---|---:|---|
| Malformed body | 400 | `COMMON.INVALID_INPUT` |
| Field violation | 422 | `COMMON.VALIDATION_FAILED` |
| Missing, unknown, expired, or revoked session | 401 | `AUTH.UNAUTHORIZED` |
| Wrong credentials, unknown or disabled account at sign-in | 401 | `AUTH.INVALID_CREDENTIALS` |
| Session whose account was later disabled | 403 | `AUTH.ACCOUNT_DISABLED` |
| Caller has no operator grant | 403 | `OPERATOR.FORBIDDEN` |
| `Origin` missing on an unsafe method, or not allowed by the operator starter | 403 | `OPERATOR.ORIGIN_REJECTED` |
| `Origin` outside both `CORS_ALLOW_ORIGINS` and `OPERATOR_ALLOWED_ORIGINS` | 403 | none: the kernel CORS policy rejects it before routing |
| `X-CSRF-Token` missing or not bound to this session | 403 | `OPERATOR.CSRF_REJECTED` |
| Target account holds an operator grant | 409 | `OPERATOR.TARGET_PROTECTED` |
| Sign-in quota exceeded | 429 | `COMMON.RATE_LIMITED` |
| Persistence unavailable | 503 | `COMMON.SERVICE_UNAVAILABLE` |

On `401` or `OPERATOR.FORBIDDEN` the Admin Console returns to sign-in. On
`OPERATOR.CSRF_REJECTED` it refetches `GET /v1/operator/session` once and retries.

## Users

All routes require an operator session; unsafe methods also require `Origin` and `X-CSRF-Token`.

| Operation | Method and path | Success |
|---|---|---|
| List users | `GET /v1/operator/users?q=&status=&page=&per_page=` | `200` paginated `managed user` list |
| Get user | `GET /v1/operator/users/:id` | `200 managed user` |
| Disable | `POST /v1/operator/users/:id/disable` | `200 managed user` |
| Enable | `POST /v1/operator/users/:id/enable` | `200 managed user` |
| End sessions | `POST /v1/operator/users/:id/sessions/revoke` | `204` |

A managed user is `{ id, username, email, nickname, status, is_operator, created_at, last_login }`,
where `status` is `active` or `disabled` and `last_login` may be `null`. Password hashes, phone, bio,
and avatar are never returned.

- `q` (at most 100 characters) matches username or email case-insensitively as a literal substring;
  `%` and `_` are not wildcards. `status` is `active`, `disabled`, or `all` (default).
  `per_page` is 1–100 (default 15). Results are ordered by descending ID. Invalid values return
  `400 COMMON.INVALID_INPUT`.
- Disabling revokes all of the account's sessions in the same transaction, so every existing
  credential fails on its next API call with `401 AUTH.UNAUTHORIZED`, and new logins fail with
  `AUTH.INVALID_CREDENTIALS`. Enabling does not restore sessions. Repeating either command returns
  the current account unchanged.
- Ending sessions revokes every active session without changing status.
- Accounts that hold an operator grant, including the caller's own, cannot be disabled, enabled, or
  signed out here: `409 OPERATOR.TARGET_PROTECTED`. Change operators from the CLI.
- An unknown account returns `404 USER.NOT_FOUND`; a non-numeric ID returns
  `400 COMMON.INVALID_INPUT`.

## Audit Logs

`GET /v1/operator/audit-logs` returns platform-wide audit history in the audit entry shape defined
by [`AUDIT.md`](AUDIT.md), paginated, newest first (`created_at` then `id`, descending).

| Query | Rule |
|---|---|
| `from`, `to` | RFC 3339 instants; `from` inclusive, `to` exclusive. `to` defaults to now, `from` to 30 days before `to`. The range must be ordered and at most 92 days |
| `user_id` | Positive user ID of the recorded user |
| `action`, `resource`, `method`, `request_id`, `status_code` | Exact matches with the same bounds as `AUDIT.md` |
| `page`, `per_page` | `per_page` 1–100, default 15 |

Invalid values and ranges return `400 COMMON.INVALID_INPUT`. The endpoint reads only; operators
cannot modify or delete audit records.

## App Settings

Registered only when the `setting` starter is also selected; otherwise these paths return `404`.
Behavior, validation, `If-Match` preconditions, `ETag`s, and errors are exactly the app-scope rules of
[`SETTINGS.md`](SETTINGS.md); the setting starter serves them and the operator starter only
authorizes.

| Operation | Method and path | Request | Success |
|---|---|---|---|
| List app settings | `GET /v1/operator/settings` | — | `200` effective app settings, including non-public keys |
| Set override | `PATCH /v1/operator/settings/:key` | `If-Match: "setting-v{n}"`, `{ "value": … }` | `200` setting + `ETag` |
| Reset override | `DELETE /v1/operator/settings/:key` | `If-Match: "setting-v{n}"` | `204` + `ETag` |

Each change produces one audit record with the operator as actor; the setting starter adds the
setting's business change to that record.

## Organizations

Registered only when the `organization` starter is also selected; otherwise these paths return
`404`. An operator is not an organization member: these routes apply no membership check and never
set organization context. They are read-only.

| Operation | Method and path | Success |
|---|---|---|
| List organizations | `GET /v1/operator/organizations?q=&page=&per_page=` | `200` paginated `operator organization` list |
| Get organization | `GET /v1/operator/organizations/:id` | `200 operator organization` |
| List members | `GET /v1/operator/organizations/:id/members?page=&per_page=` | `200` paginated `operator member` list |

An operator organization is `{ id, name, slug, created_by, member_count, created_at, updated_at }`.
An operator member is `{ id, user_id, username, nickname, email, role, joined_at }`, where `id` is
the membership ID and `role` is `owner`, `admin`, or `member`. Memberships of deleted accounts are
excluded from both the list and `member_count`.

- `q` (at most 100 characters) matches name or slug case-insensitively as a literal substring; `%`
  and `_` are not wildcards. Organizations are ordered by descending ID, members by ascending user
  ID. `per_page` is 1–100 (default 15). Invalid values return `400 COMMON.INVALID_INPUT`.
- An unknown organization returns `404 ORGANIZATION.NOT_FOUND`. Operator scope is global, so this
  discloses only that the ID does not exist.

## Webhooks

Registered only when the `webhook` starter is also selected. Every route is nested under one
organization and reuses the organization-scoped queries and representations of
[`WEBHOOKS.md`](WEBHOOKS.md), so signing secrets, ciphertext, event payloads, signatures, and target
response bodies are never returned.

| Operation | Method and path | Success |
|---|---|---|
| List endpoints | `GET /v1/operator/organizations/:id/webhook-endpoints` | `200` paginated endpoint list |
| List deliveries | `GET /v1/operator/organizations/:id/webhook-deliveries?endpoint_id=&status=` | `200` paginated delivery list |
| List attempts | `GET /v1/operator/organizations/:id/webhook-deliveries/:delivery_id/attempts` | `200` paginated attempt list |
| Replay delivery | `POST /v1/operator/organizations/:id/webhook-deliveries/:delivery_id/replay` | `200` delivery |

- An unknown organization returns `404 ORGANIZATION.NOT_FOUND` before any webhook lookup. A delivery
  that belongs to another organization returns `404 WEBHOOK.DELIVERY_NOT_FOUND`.
- `status` is a delivery status and `endpoint_id` a positive integer; invalid filters and IDs return
  `400 COMMON.INVALID_INPUT`.
- Replay is the operator replay rule of `WEBHOOKS.md` with the operator as actor: the delivery must
  be terminal, below the replay limit, and its endpoint active, otherwise
  `409 WEBHOOK.REPLAY_NOT_ALLOWED`. It keeps the original message ID and returns the delivery as
  `pending`. Each accepted replay starts one new delivery cycle, so the command is not idempotent.
  It writes the webhook starter's replay audit record in addition to the request audit record.
- Operators cannot create, update, disable, delete, or test endpoints or rotate secrets.

## Notification Deliveries

Registered only when the `notification` starter is also selected.

`GET /v1/operator/notification-deliveries?status=&channel=&user_id=&page=&per_page=` returns the
platform's channel delivery ledger, newest first (descending delivery ID), paginated.

A delivery is `{ id, notification_id, user_id, kind, channel, status, attempts, last_failure_code,
available_at, delivered_at, created_at, updated_at }`. `status` is `pending`, `processing`,
`delivered`, or `failed`; `channel` is `in_app` or `email`; `user_id` is the recipient;
`delivered_at` may be `null`. Notification titles, bodies, action URLs, and destination addresses are
never returned. Invalid filters return `400 COMMON.INVALID_INPUT`. The endpoint reads only.

## System Status

`GET /v1/operator/system` returns `{ version, revision, go_version, starters, database }`.
`version` and `revision` come from Go build metadata (`revision` is the first 12 characters of the
VCS revision, empty when unavailable); `starters` lists the default starters followed by the selected
optional starters; `database` is `ok` or `unavailable`.

## Configuration

| Variable | Required | Rule |
|---|---|---|
| `OPERATOR_ALLOWED_ORIGINS` | When selected | Exact `scheme://host[:port]` origins, comma separated; `https` in production; no wildcards or paths |
| `OPERATOR_SESSION_COOKIE_NAME` | No | Cookie token; the `__Host-` prefix is allowed only in production |

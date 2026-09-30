# Admin Console Operator Access — Implementation Plan

## 1. Executive Summary

**Outcome.** A platform operator signs in to the static Admin Console and manages the running
product: find and disable users, force a user's sessions to end, read the global audit trail, change
app-scoped settings, and see system status. Every protected operation is authorized by the Go API.

**Problem.** The Admin Console ships no protected feature. It is static, so it cannot hold a server
credential, and Luas has no platform-operator identity: app-scoped settings can only be changed
through the CLI.

**In scope.** An optional `operator` starter (API), a Go-issued browser session for the Admin
Console, operator grants managed from the CLI, four operator feature areas, and matching Admin
Console screens.

**Non-goals.** Operator self-service registration or granting from the UI; editing user profiles,
passwords, or emails; hard account deletion; impersonation; organization, permission, usage,
webhook, notification, or asset administration; multi-factor authentication; Web (Next.js) changes.

**Deployable units.** `api/` and `admin/`, joined by `contracts/OPERATORS.md` and `openapi.yaml`.

**Success signal.** With `OPTIONAL_STARTERS=operator`, a user granted by `luas operator:grant`
signs in to the Admin Console, disables another user (whose next API call fails with
`AUTH.ACCOUNT_DISABLED`), finds that action in the global audit log, and signs out; a non-operator
with valid credentials cannot sign in.

**Decisions from the owner (2026-09-30).**

1. Operator identity is a flag on the user, granted and revoked only from the CLI, with an audit
   record. It is stored as a one-to-one `platform_operators` grant keyed by user so that the `user`
   module stays the only writer of `users` and the capability remains an optional starter.
2. First release: user management, audit log viewer, app-scoped settings, system status.
3. Session transport: the Go API issues its own HttpOnly cookie to the Admin origin (no separate
   gateway), with exact-Origin checks and a CSRF token for unsafe methods.

## 2. Actors and Access Surfaces

| Actor | Goal | Entry points | Read scope | Mutation scope | Approval boundary | Denial behavior |
|---|---|---|---|---|---|---|
| Platform operator | Operate the deployment | Admin Console via `/v1/operator/*` | All users (minimized fields), all audit logs, app settings, system status | Disable/enable and end sessions of non-operator users; set/reset app settings | None; every action is audited | `403 OPERATOR.FORBIDDEN` when the grant is missing |
| Deployment owner | Decide who operates | `luas operator:grant`, `operator:revoke`, `operator:list` on a host with database access | Operator grants | Create/delete grants | Shell access to the deployment | Command exits non-zero |
| Signed-in user without a grant | Use the product | Web, public API | Own data only | Own data only | — | `403 OPERATOR.FORBIDDEN` on operator routes, before any resource lookup |
| Anonymous caller | — | Public routes | Public data | None | — | `401 AUTH.UNAUTHORIZED` |

An operator cannot disable, enable, or end sessions for any account that holds an operator grant,
including their own (`409 OPERATOR.TARGET_PROTECTED`). Removing an operator is a CLI decision.

## 3. End-to-End Workflows

### 3.1 Operator sign-in

1. Operator opens the Admin Console; `GET /v1/operator/session` returns `401`; the console shows the
   sign-in form.
2. `POST /v1/operator/session` with `{ identifier, password }` and a matching `Origin`.
3. The API verifies credentials through the user starter, requires an operator grant, issues a
   normal authentication session, sets the HttpOnly session cookie, and returns the operator
   profile plus a CSRF token.
4. Rejections: unknown, wrong, or disabled account `401 AUTH.INVALID_CREDENTIALS` (the public login
   rule, so sign-in never reveals account state); no grant `403 OPERATOR.FORBIDDEN`, checked before
   any session is issued; wrong Origin `403 OPERATOR.ORIGIN_REJECTED`; throttled
   `429 COMMON.RATE_LIMITED`.

### 3.2 Protected operator request

Every `/v1/operator/*` request other than sign-in runs the `operator` middleware group: read the
cookie, authenticate the session with current user state, require a current operator grant, and for
unsafe methods require an allowed `Origin` and a valid `X-CSRF-Token`. The global audit middleware
then records unsafe requests with the operator as actor.

### 3.3 Disable a user

Operator selects a user and confirms. `POST /v1/operator/users/:id/disable` sets the account to
disabled and revokes all of its sessions in one transaction. The user's next API call fails with
`403 AUTH.ACCOUNT_DISABLED`. Repeating the command returns the current state (`200`, idempotent).

### 3.4 Other flows

- Enable: inverse of disable; sessions are not restored.
- End sessions: revoke all sessions of an active user without disabling.
- Audit viewer: filtered, paginated read of all audit logs.
- App settings: list app-scope settings; set or reset with `If-Match` version preconditions.
- System status: version, active starters, database readiness.
- Sign-out: revoke the presented session and expire the cookie; idempotent.
- Grant revoked or account disabled mid-session: the next request fails with `403`; the console
  returns to sign-in.
- Session expiry follows the existing absolute and idle limits; no new timer.

## 4. Module Decomposition

| Module | Responsibility | Inputs | Outputs | Data ownership | Dependencies | Consumers | Priority |
|---|---|---|---|---|---|---|---|
| `operator` (new optional starter) | Operator grants, operator browser session, operator routes, CLI commands | HTTP, CLI | JSON, cookie | `platform_operators` | `user`, `audit`; `setting` optional | Admin Console | P0 |
| `user` (existing) | Account status transitions and session revocation | Domain calls | Domain values | `users`, `authentication_sessions` | — | `operator` | P0 |
| `audit` (existing) | Global audit query | Domain calls | Domain values | `audit_logs` | — | `operator` | P1 |
| `setting` (existing, optional) | App-scope read and write | Domain calls | Domain values | `settings` | `organization` | `operator` | P1 |

### Module: operator

- **Responsibilities:** grant persistence; cookie session issue/resolve/end; Origin and CSRF
  enforcement; operator route handlers that delegate to owning modules; CLI grant commands.
- **Exclusions:** does not write `users`, `authentication_sessions`, `audit_logs`, or `settings`
  directly; never stores credentials in the browser-visible response.
- **Entry points:** routes in section 7; commands `operator:grant <email>`,
  `operator:revoke <email>`, `operator:list`.
- **Invariants:** a grant references an existing user; at most one grant per user; operator routes
  never run for a caller without a current grant; protected targets cannot be mutated.
- **Transactions:** grant writes are single-row; status changes are owned by `user`.
- **Failure and recovery:** database outage yields `503 COMMON.SERVICE_UNAVAILABLE`; the console
  shows a retryable state. Lost grants are restored with `operator:grant`.
- **Acceptance:** see section 12.

### Module changes in existing starters

- `user` exposes `domain.UserAdministrator`: `ListUsers(filter, page)`, `GetUser(id)`,
  `SetUserStatus(id, status) (*User, changed bool)` which revokes all sessions in the same
  transaction when disabling, and `RevokeUserSessions(id)`. It also exposes
  `domain.CredentialVerifier` so the operator starter can sign in without duplicating password
  checks, timing protection, or throttling.
- `audit` exposes `domain.AuditLogQuery.ListAll(filter, page)`.
- `setting` already exposes `domain.SettingReader` and `domain.AppSettingWriter`.

## 5. Domain Model and Vocabulary

Add to `CONTEXT.md`:

- **Platform operator:** a user with a current operator grant. Operates the deployment through the
  Admin Console. Not an organization role and not a permission key.
- **Operator grant:** the durable record that makes a user a platform operator.
- **Operator session:** an ordinary authentication session carried to the Admin Console in an
  HttpOnly cookie instead of a bearer header.

| Object | Identity | Lifecycle | Invariants |
|---|---|---|---|
| OperatorGrant | `user_id` | created by `operator:grant`, deleted by `operator:revoke` | one per user; user must exist; deleted with the user (cascade) |
| User status (existing) | `users.id` | `active (1)` ⇄ `disabled (0)` | disabling revokes all sessions atomically |

## 6. Persistence Design

Migration `2026_10_01_000000_create_platform_operators_table` (owned by the `operator` manifest,
frozen SQL):

```sql
CREATE TABLE platform_operators (
    user_id bigint NOT NULL,
    granted_at timestamptz NOT NULL,
    CONSTRAINT platform_operators_pkey PRIMARY KEY (user_id),
    CONSTRAINT fk_platform_operators_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
)
```

- No updates: a grant exists or it does not. Time is UTC.
- Rollback: `DROP TABLE IF EXISTS platform_operators`; no backfill; no seed data.
- Account deletion removes the grant by cascade.
- The golden schema file is regenerated in the same change.

No new columns on existing tables. The user list query uses existing indexes
(`users_pkey`, `idx_users_username`, `uni_users_email`); substring search is bounded by page size.

## 7. HTTP Contract

All routes live under `/v1/operator` and are documented in `contracts/OPERATORS.md` and
`contracts/openapi.yaml`. Envelope, pagination, and `request_id` follow `contracts/README.md`.

| Method | Path | Operation | Auth | Success |
|---|---|---|---|---|
| `POST` | `/v1/operator/session` | Sign in | Origin + credentials | `200` operator + `csrf_token`, sets cookie |
| `GET` | `/v1/operator/session` | Current operator | Cookie | `200` operator + `csrf_token` |
| `DELETE` | `/v1/operator/session` | Sign out | Cookie + Origin + CSRF | `204`, expires cookie |
| `GET` | `/v1/operator/users` | List users | Operator | `200` page |
| `GET` | `/v1/operator/users/:id` | Get user | Operator | `200` user |
| `POST` | `/v1/operator/users/:id/disable` | Disable | Operator + CSRF | `200` user |
| `POST` | `/v1/operator/users/:id/enable` | Enable | Operator + CSRF | `200` user |
| `POST` | `/v1/operator/users/:id/sessions/revoke` | End sessions | Operator + CSRF | `204` |
| `GET` | `/v1/operator/audit-logs` | Global audit | Operator | `200` page |
| `GET` | `/v1/operator/settings` | App settings | Operator; `setting` selected | `200` list |
| `PATCH` | `/v1/operator/settings/:key` | Set | Operator + CSRF + `If-Match` | `200` setting + `ETag` |
| `DELETE` | `/v1/operator/settings/:key` | Reset | Operator + CSRF + `If-Match` | `204` + `ETag` |
| `GET` | `/v1/operator/system` | System status | Operator | `200` status |

Settings routes are registered only when the `setting` starter is selected; otherwise they return
`404 COMMON.NOT_FOUND` and `GET /v1/operator/system` reports `setting` as inactive.

### Cookie and CSRF

- Cookie `OPERATOR_SESSION_COOKIE_NAME` (default `__Host-luas_operator` in production,
  `luas_operator` otherwise): HttpOnly, `Secure` in production, `SameSite=Strict`, `Path=/`, no
  `Domain`, `Max-Age` equal to the session's absolute lifetime.
- CSRF token: `base64url(HMAC-SHA256(key = session credential, "luas-operator-csrf/v1"))`. It is
  bound to one session, needs no server storage, and cannot be computed cross-site. Unsafe methods
  require header `X-CSRF-Token`, compared in constant time.
- `OPERATOR_ALLOWED_ORIGINS`: exact origins. Required and validated when the starter is selected;
  wildcards and credentials-with-`*` are rejected. Unsafe methods require an exact `Origin` match.
- Responses under `/v1/operator` send `Cache-Control: private, no-store`.

### Error table

| Condition | HTTP | `error_code` | Disclosure | Retryable |
|---|---:|---|---|---|
| Malformed body or query | 400 | `COMMON.INVALID_INPUT` | Field-neutral | No |
| Field violation | 422 | `COMMON.VALIDATION_FAILED` | Reviewed field keys | After correction |
| No or invalid session cookie | 401 | `AUTH.UNAUTHORIZED` | None | After sign-in |
| Wrong credentials | 401 | `AUTH.INVALID_CREDENTIALS` | Same for unknown/wrong/disabled | After correction |
| Existing session whose account was disabled | 403 | `AUTH.ACCOUNT_DISABLED` | Own state only | No |
| Caller not an operator | 403 | `OPERATOR.FORBIDDEN` | No resource detail | No |
| Origin missing or not allowed | 403 | `OPERATOR.ORIGIN_REJECTED` | None | No |
| CSRF token missing or wrong | 403 | `OPERATOR.CSRF_REJECTED` | None | After refetching session |
| Target holds an operator grant | 409 | `OPERATOR.TARGET_PROTECTED` | Target exists | No |
| User not found | 404 | `USER.NOT_FOUND` | Operator scope is global | No |
| Setting unknown / invalid / stale / no `If-Match` | 404/422/412/428 | `SETTING.*` (existing) | As in `SETTINGS.md` | Per code |
| Sign-in throttled | 429 | `COMMON.RATE_LIMITED` | No bucket detail | After delay |
| Database unavailable | 503 | `COMMON.SERVICE_UNAVAILABLE` | None | Yes |

### POST /v1/operator/session — sign in

```text
COMMAND OperatorSignIn(request)
BIND request.identifier, request.password FROM JSON body (max 1 KiB)
REQUIRE request.Origin IN OPERATOR_ALLOWED_ORIGINS ELSE OPERATOR.ORIGIN_REJECTED
VALIDATE identifier length 1..100, password length 1..128 ELSE COMMON.VALIDATION_FAILED
REQUIRE per-IP and per-identifier sign-in quota ELSE COMMON.RATE_LIMITED
LOAD user FROM user.CredentialVerifier USING identifier, password
  OR ELSE AUTH.INVALID_CREDENTIALS
REQUIRE user.status == active ELSE AUTH.INVALID_CREDENTIALS
LOAD grant FROM operator.platform_operators USING user.id
  OR ELSE OPERATOR.FORBIDDEN
BEGIN TRANSACTION user.sessions
  INSERT authentication session FOR user
COMMIT
DERIVE csrf_token = HMAC(credential, "luas-operator-csrf/v1")
RETURN 200 { operator: { id, username, email, nickname }, csrf_token } WITH Set-Cookie
```

Disabled accounts return the same `AUTH.INVALID_CREDENTIALS` as the public login, so sign-in does
not reveal account state. The grant check happens before a session is issued, so no session exists
for a non-operator.

### GET /v1/operator/session — current operator

```text
QUERY CurrentOperator(cookie)
BIND credential FROM cookie ELSE AUTH.UNAUTHORIZED
LOAD identity FROM user.SessionAuthenticator USING credential
  OR ELSE AUTH.UNAUTHORIZED (unknown, expired, revoked) | AUTH.ACCOUNT_DISABLED
LOAD grant FROM operator.platform_operators USING identity.user_id OR ELSE OPERATOR.FORBIDDEN
LOAD user FROM user.UserAdministrator USING identity.user_id
DERIVE csrf_token = HMAC(credential, "luas-operator-csrf/v1")
RETURN 200 { operator: { id, username, email, nickname }, csrf_token }
```

Safe method: no Origin or CSRF check. The console calls it on load and after a CSRF rejection.

### DELETE /v1/operator/session — sign out

```text
COMMAND OperatorSignOut(cookie, headers)
BIND credential FROM cookie
REQUIRE Origin allowed ELSE OPERATOR.ORIGIN_REJECTED
REQUIRE X-CSRF-Token == HMAC(credential) ELSE OPERATOR.CSRF_REJECTED
BEGIN TRANSACTION user.sessions
  UPDATE session SET revoked_at = now WHERE token_hash = sha256(credential) AND revoked_at IS NULL
COMMIT
RETURN 204 WITH expired cookie
```

A missing or already revoked session still returns `204` and expires the cookie.

### POST /v1/operator/users/:id/disable — disable a user

```text
COMMAND DisableUser(operator, path.id)
BIND operator FROM operator middleware
VALIDATE path.id is a positive integer ELSE COMMON.INVALID_INPUT
LOAD target FROM user.UserAdministrator USING path.id OR ELSE USER.NOT_FOUND
REQUIRE target.id HAS NO operator grant ELSE OPERATOR.TARGET_PROTECTED
BEGIN TRANSACTION user.users
  UPDATE users SET status = 0 WHERE id = target.id AND status = 1
  UPDATE authentication_sessions SET revoked_at = now, revocation_reason = 'account_disabled'
    WHERE user_id = target.id AND revoked_at IS NULL
COMMIT
RETURN 200 { user } built from the committed row
```

Idempotent: an already disabled user returns `200` unchanged and revokes nothing new. Enable is the
same command with `status = 1` and no session change. Concurrent disable/enable is last-writer-wins
on a single row; both outcomes are valid states and are audited.

### POST /v1/operator/users/:id/sessions/revoke — end sessions

```text
COMMAND RevokeUserSessions(operator, path.id)
LOAD target FROM user.UserAdministrator USING path.id OR ELSE USER.NOT_FOUND
REQUIRE target.id HAS NO operator grant ELSE OPERATOR.TARGET_PROTECTED
BEGIN TRANSACTION user.sessions
  UPDATE authentication_sessions SET revoked_at = now, revocation_reason = 'operator'
    WHERE user_id = target.id AND revoked_at IS NULL
COMMIT
RETURN 204
```

### PATCH /v1/operator/settings/:key and DELETE /v1/operator/settings/:key

```text
COMMAND SetAppSetting(operator, path.key, If-Match, body.value)
REQUIRE setting starter selected ELSE COMMON.NOT_FOUND
REQUIRE If-Match present ELSE SETTING.PRECONDITION_REQUIRED
DERIVE expected_version FROM If-Match "setting-v{n}" ELSE COMMON.INVALID_INPUT
BEGIN TRANSACTION setting.settings
  CAS_UPDATE app setting path.key EXPECT expected_version ELSE SETTING.VERSION_CONFLICT
COMMIT
RETURN 200 { setting } WITH ETag "setting-v{version}"
```

Reset follows the same guards and returns `204` with the new `ETag`. Validation and catalog rules
are the existing `setting` rules.

### Queries

```text
QUERY ListUsers(operator, q, status, page, page_size)
VALIDATE q length 0..100; status IN {active, disabled, all}; page >= 1; page_size 1..100 (default 20)
LOAD page FROM user.UserAdministrator
  WHERE (q empty OR username ILIKE %q% OR email ILIKE %q%) AND status filter AND deleted_at IS NULL
  ORDER BY id DESC
  LIMIT page_size
RETURN 200 { items: [{ id, username, email, nickname, status, is_operator, created_at, last_login }], pagination }
```

`q` is escaped for `LIKE` wildcards. Password hashes, phone, and bio are never returned.
`is_operator` is derived by the operator starter with one grant lookup for the page's user IDs.

```text
QUERY ListAuditLogs(operator, filters, page, page_size)
VALIDATE user_id >= 1; action, resource <= 180 chars; method IN HTTP methods;
  status_code 100..599; from < to and to - from <= 92 days; page_size 1..100 (default 50)
LOAD page FROM audit.AuditLogQuery WHERE filters ORDER BY created_at DESC, id DESC LIMIT page_size
RETURN 200 { items: [audit log DTO as in AUDIT.md], pagination }
```

```text
QUERY SystemStatus(operator)
LOAD version FROM build metadata
LOAD starters FROM starter registry (active names in dependency order)
LOAD database FROM readiness probe (ok | unavailable)
RETURN 200 { version, starters, database }
```

## 8. State Machines

User status (owned by `user`):

| Current | Command | Guard | Next | Actor | Side effects | Failure |
|---|---|---|---|---|---|---|
| active | disable | target not an operator | disabled | operator | revoke all sessions (same transaction) | `OPERATOR.TARGET_PROTECTED` |
| disabled | disable | — | disabled | operator | none | — |
| disabled | enable | target not an operator | active | operator | none | `OPERATOR.TARGET_PROTECTED` |
| active | enable | — | active | operator | none | — |

Operator grant: `absent → granted` (`operator:grant`), `granted → absent` (`operator:revoke` or
account deletion). Repeating either command is a no-op with exit code 0.

## 9. Authorization Matrix

| Capability | Anonymous | User | Operator |
|---|---|---|---|
| Sign in to Admin | credentials + grant required | forbidden (`OPERATOR.FORBIDDEN`) | allowed |
| List/get users | forbidden | forbidden | all users |
| Disable/enable/end sessions | forbidden | forbidden | non-operator users only |
| Global audit logs | forbidden | own logs via `/v1/audit-logs` | all logs |
| App settings | public keys via `/v1/settings/public` | forbidden | all app keys |
| System status | forbidden | forbidden | allowed |
| Grant/revoke operators | forbidden | forbidden | forbidden (CLI only) |

The Admin Console route guard only hides screens. The API enforces every row above.

## 10. Asynchronous Work and Events

No new jobs, schedules, or events. Operator sessions are ordinary authentication sessions, pruned
by the existing `auth-session:prune` command. Audit records are written synchronously by the
existing audit middleware.

| Trigger | Source record | Generated record/action | Owner | Idempotency key | Failure recovery |
|---|---|---|---|---|---|
| Unsafe `/v1/operator/*` request | HTTP request | `audit_logs` row | `audit` | request ID | Existing audit middleware behavior |
| `operator:grant` / `operator:revoke` | CLI invocation | `audit_logs` row (`actor_type = system`) | `operator` | none (single write) | Re-run the command |
| Disable | `users` row | session revocations | `user` | same transaction | Atomic |

## 11. Security, Privacy, Audit, and Operations

- **Sensitive data:** credentials never leave the cookie; the CSRF token is derived and non-secret
  to the session owner. User lists expose id, username, email, nickname, status, operator flag,
  and timestamps only.
- **Browser boundary:** HttpOnly `SameSite=Strict` cookie, exact Origin, CSRF header, `no-store`
  caching. The Admin Console never stores tokens.
- **Abuse:** sign-in reuses the user starter's per-IP and per-identifier throttles and dummy-hash
  timing; the same generic failure is returned for unknown, wrong, and disabled accounts.
- **Audit:** every unsafe operator request is recorded with the operator as actor; grant changes
  record the target user and the system actor.
- **Observability:** existing request logs, metrics, and `request_id`; the console shows
  `request_id` on failures.
- **Configuration:** `OPERATOR_ALLOWED_ORIGINS` (required when selected),
  `OPERATOR_SESSION_COOKIE_NAME` (optional). Validation fails startup on invalid values.
- **Disable switch:** remove `operator` from `OPTIONAL_STARTERS`; routes, middleware, and CLI
  commands disappear; grants remain for re-enablement.
- **Deployment:** the Admin origin must route `/api/*` to the Go API on the same origin so the
  cookie is first-party.

## 12. Browser Surface (Admin Console)

| Route | Feature | Data | States |
|---|---|---|---|
| `/login` | `operator-session` | `POST/GET/DELETE /v1/operator/session` | idle, submitting, invalid credentials, forbidden, rate limited, unavailable |
| `/console` (guarded) | layout | session query | loading, unauthenticated → `/login` |
| `/console/users` | `users` | list, disable, enable, end sessions | loading, empty, list, confirm dialog, protected target, not found, unavailable |
| `/console/audit` | `audit` | audit list with filters in URL search params | loading, empty, list, invalid range |
| `/console/settings` | `settings` | list, set, reset | loading, inactive starter, stale version (refetch prompt), invalid value |
| `/console` index | `system` | system status (extends existing readiness panel) | loading, degraded |

- TanStack Query owns server state; mutations invalidate the affected list; no retries on mutations.
- The HTTP client sends `credentials: 'same-origin'` and adds `X-CSRF-Token` from the session query
  to unsafe requests; a `403 OPERATOR.CSRF_REJECTED` refetches the session once.
- Any `401` or `403 OPERATOR.FORBIDDEN` clears query cache and routes to `/login`.
- Zod validates every response; i18next keys live under each feature; accessible dialogs confirm
  destructive actions; layouts work from 360 px up.
- No mock BFF (Admin has none by design); tests stub `fetch`.

## 13. Test and Verification Strategy

| Concern | Proof |
|---|---|
| CSRF derivation, Origin matching, cookie attributes | Unit tests in `operator` |
| Middleware: 401/403 precedence, grant revoked mid-session, disabled account | Handler tests with test doubles |
| Grant repository, cascade on user delete | PostgreSQL test |
| Disable revokes sessions atomically; idempotent repeat | PostgreSQL test in `user` |
| Protected target rule, including self | Service test |
| Audit global query ordering, filters, range limit | PostgreSQL test in `audit` |
| Settings `If-Match` flow through operator routes | Handler test with setting doubles |
| Starter disabled: no routes, no commands | Starter catalog and route-catalog tests |
| Migration | Golden schema, rollback, frozen-history tests |
| Contract | `make contract-check`; OpenAPI route parity |
| Admin screens | Vitest hook/component tests, type-check, lint, build |
| End to end | Local run: grant via CLI, sign in, disable user, see audit entry, sign out |

## 14. Delivery Plan

| Slice | Outcome | Units | Contract / migration | Rollback |
|---|---|---|---|---|
| 1. Operator identity and session | Operator can sign in and out of Admin; non-operators cannot | api, admin, contracts | `OPERATORS.md` session section; `platform_operators` migration | Remove `operator` from `OPTIONAL_STARTERS` |
| 2. User management | List, disable, enable, end sessions | api (`user`, `operator`), admin | users section | Same |
| 3. Audit viewer | Global audit list with filters | api (`audit`, `operator`), admin | audit section | Same |
| 4. App settings and system status | Set/reset app settings; status page | api (`operator`, `setting`), admin | settings and system sections | Same |

Critical path: slice 1 → 2. Slices 3 and 4 depend only on slice 1 and can proceed in either order.
Each slice updates `contracts/`, `openapi.yaml`, generated types, `UPGRADING.md`, and tests.

## 15. Acceptance Criteria

1. Given a granted operator, when they sign in from an allowed origin, then the response sets an
   HttpOnly `SameSite=Strict` cookie, returns a CSRF token, and exposes no credential.
2. Given a user without a grant and correct credentials, when they sign in, then the API returns
   `403 OPERATOR.FORBIDDEN` and no session row exists for that attempt.
3. Given an operator session, when an unsafe request lacks the CSRF token or comes from another
   origin, then the API returns `403` with the matching code and changes nothing.
4. Given an active non-operator user with sessions, when an operator disables them, then status is
   disabled, all sessions are revoked in the same transaction, and the audit log shows the operator.
5. Given an operator account as target, when another operator disables it, then the API returns
   `409 OPERATOR.TARGET_PROTECTED`.
6. Given the grant is revoked by CLI, when the operator makes any request, then it fails with
   `403 OPERATOR.FORBIDDEN`.
7. Given `setting` is not selected, when the operator opens settings, then the console shows the
   inactive state and the API returns `404`.
8. Given `operator` is not selected, then no `/v1/operator` route and no `operator:*` command exists.

## 15a. Decisions Made During Slice 1

- **Kernel CORS:** `gin-contrib/cors` rejects an `Origin` outside `CORS_ALLOW_ORIGINS` with a bare
  `403` before routing, and a CDN or dev proxy usually rewrites `Host`, so same-origin Admin requests
  were blocked. When `operator` is selected, the kernel CORS policy now also allows
  `OPERATOR_ALLOWED_ORIGINS`, the `X-CSRF-Token` header, and `PATCH`. An origin outside both lists
  still receives the kernel's bare `403`.
- **Sign-out attribution:** sign-out resolves the session before revoking it so the audit record
  names the operator; an already invalid session still signs out with `204`.
- **Dev origin:** the Admin dev server runs at `http://127.0.0.1:4173`.

## 16. Open Decisions

None blocking. Recorded assumptions:

- **A1.** Operators cannot act on accounts that hold an operator grant, including their own.
  Reversible by relaxing one guard.
- **A2.** Operator sign-in reuses the public login throttles rather than a separate budget.
- **A3.** Audit range is capped at 92 days per query to bound scans; operators page within it.
- **A4.** Multi-factor authentication is out of scope; the design leaves the sign-in command as the
  single place to add it.

## 17. Readiness Gate

- Product and domain: outcome, non-goals, actors, rejection and expiry flows, and the user-status
  state machine are defined; vocabulary additions are listed for `CONTEXT.md`.
- Ownership: `operator` owns grants and routes; `user`, `audit`, and `setting` keep sole write
  ownership of their records; no transaction crosses a runtime boundary.
- Data and contract: migration, rollback, constraints, pagination bounds, ordering, error codes,
  idempotency, and concurrency are defined for every mutation.
- Side effects: no asynchronous work; audit rows come from the existing middleware.
- Delivery: four dependency-ordered slices, each with acceptance criteria and PostgreSQL proof where
  SQL behavior matters.

readiness: READY WITH RECORDED ASSUMPTIONS

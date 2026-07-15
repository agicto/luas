# Authentication Contracts

Luas currently ships two authentication boundaries. They share the global HTTP envelope and
`error_code` vocabulary, but their paths, credentials, user DTOs, and session ownership are not
interchangeable.

## Browser Auth Contract

The Web feature service and development mock BFF use a browser-oriented, same-origin contract:

| Operation | Browser endpoint | Request | Successful `data` |
|---|---|---|---|
| Login | `POST /api/auth/login` | `{ email, password }` | `{ user: { id, email, name } }` |
| Register | `POST /api/auth/register` | `{ name, email, password }` | `{ user: { id, email, name } }` |
| Current session | `GET /api/auth/me` | none | `{ user: { id, email, name } }` |
| Logout | `POST /api/auth/logout` | none | `{ success: true }` |

The browser user view deliberately has no `role` or permission field. The optional permission
starter is organization-scoped and resolves current grants through its own context endpoint;
assigning `admin`, `member`, or global permissions to the auth user view would invent authorization
meaning that the Go API cannot prove.

The development implementation owns an HttpOnly signed mock cookie. The production adapter owns a
separate HttpOnly API-token cookie. Unsafe operations must reject cross-origin browser requests
before reading a body or mutating state.

## Go API User Starter Contract

The Go `user` starter exposes server-side authentication-session endpoints under `/v1`:

| Operation | API endpoint | Request | Successful `data` |
|---|---|---|---|
| Login | `POST /v1/login` | `{ username, password }` | `{ access_token, token_type: "Bearer", expires_in, user }` |
| Logout | `POST /v1/logout` | Bearer session token | `{ success: true }` |
| Register | `POST /v1/register` | `{ username, password, email, nickname?, phone? }` | API user |
| Request password reset | `POST /v1/password/reset` | `{ email }` | Generic message |
| Confirm password reset | `POST /v1/password/reset/confirm` | `{ token, new_password }` | Generic message |
| Profile | `GET /v1/users/profile` | Bearer token | API user |

`access_token` is an opaque, cryptographically random session credential. It is not a JWT and has
no client-readable claims. The API stores only its SHA-256 hash and resolves identity, account
status, revocation, absolute expiry, and idle expiry from current persistence on every protected
request. `expires_in` is the number of whole seconds remaining until the absolute session expiry;
the server remains authoritative if an idle timeout or security event ends the session earlier.

API keys remain the default machine-to-machine credential. Authentication sessions represent one
signed-in user session and must not be reused as API-key scope, organization role, or permission
evidence.

### Session Lifecycle

- A successful login creates one new session with at least 256 bits of CSPRNG entropy. Plaintext
  session credentials exist only in the login response and caller custody; persistence, logs,
  traces, audit metadata, and errors contain neither plaintext nor a reversible value.
- The default absolute lifetime is 30 days and the default idle timeout is 7 days. Activity is
  touched at a bounded interval so authentication adds one indexed read per request without adding
  one write per request. Both limits are enforced server-side.
- `POST /v1/logout` revokes the presented session before returning success. A revoked, expired, or
  unknown credential receives `401 AUTH.UNAUTHORIZED` on later requests.
- Password change, successful password reset, account disablement, and account deletion invalidate
  existing access. Password change and reset revoke every session for that user in the same
  database transaction as the password update.
- Disabled accounts receive `403 AUTH.ACCOUNT_DISABLED` even when their session has not otherwise
  expired. Session persistence failures fail closed as `503 COMMON.SERVICE_UNAVAILABLE` rather than
  accepting an unverifiable credential.
- Revoked and expired rows are retained for a bounded operational window and pruned by the shipped
  operator command. The session row deliberately stores no IP address, user agent, device label, or
  provider payload; downstream apps may add reviewed device policy at their own boundary.

Session policy is restart-scoped typed configuration:

```dotenv
AUTH_SESSION_TTL=720h
AUTH_SESSION_IDLE_TIMEOUT=168h
AUTH_SESSION_TOUCH_INTERVAL=5m
AUTH_SESSION_RETENTION=720h
```

`JWT_SECRET` and `JWT_EXPIRE_DAYS` no longer configure user authentication. This is an intentional
security-breaking migration: deploying the session migration invalidates previously issued
stateless JWTs, so existing users sign in again. Luas does not silently keep the old seven-day
unrevocable path beside the new authority.

The API user uses numeric identity and backend fields such as `username`, `nickname`, and
`status`. It does not directly emit the Web shell's `name` view.

### Public Failure Semantics

- Unknown identifiers, wrong passwords, and disabled accounts all return HTTP `401` with
  `AUTH.INVALID_CREDENTIALS`. The service performs a bcrypt comparison against a fixed dummy hash
  when no account exists, so the missing-account path does not skip the dominant password work.
- Password-reset requests return the same success body for known and unknown email addresses.
  Account-specific token-storage and delivery failures are logged internally and do not change the
  public response.
- Registration currently preserves `USER.USERNAME_ALREADY_EXISTS` and
  `USER.EMAIL_ALREADY_EXISTS` for starter UX. This is a deliberate usability tradeoff: products
  with a stricter identity-enumeration threat model should replace those conflicts with one generic
  response or add an out-of-band registration flow.

### Abuse Protection

Production enables endpoint-specific authentication limits by default. Login and password-reset
flows use independent per-IP and normalized/hashed per-subject buckets; registration and reset
confirmation have their own route quotas. An auth limit always returns HTTP `429` with
`COMMON.RATE_LIMITED`, and does not reveal which bucket fired or expose quota counters.

IP identity is accepted from forwarding headers only when the direct upstream matches
`SERVER_TRUSTED_PROXIES`. The default trusts no proxies, so a client cannot rotate a spoofed
`X-Forwarded-For` value to evade quotas.

The built-in stores are process-local. Multi-replica production deployments must provide
equivalent distributed enforcement at the gateway/WAF or through a shared limiter store. These
quotas are a starter baseline, not a substitute for MFA, breached-credential checks, adaptive bot
controls, or product-specific account recovery policy.

The starter email adapter is currently synchronous. Products that require strict response-time
uniformity for password recovery should enqueue token delivery through a bounded, observable,
durable worker and keep the HTTP response independent of delivery completion.

## Production API Adapter

Luas ships a same-origin Web adapter with explicit route handlers for the Go `user` starter and
enabled optional starter features. Set browser requests to the same-origin `/api` route, enable the
adapter on the Web server, and configure its private Go API base URL:

```dotenv
NEXT_PUBLIC_API_URL=/api
API_ADAPTER_ENABLED=true
API_UPSTREAM_URL=http://api:8025/v1
API_UPSTREAM_TIMEOUT_MS=5000
API_UPSTREAM_MAX_RESPONSE_BYTES=1048576
API_CLIENT_IP_HEADER=x-real-ip
```

`API_UPSTREAM_URL` and `API_CLIENT_IP_HEADER` are server-only. The adapter calls only fixed paths
owned by checked-in Route Handlers; it is not an arbitrary proxy. Production startup rejects an
enabled adapter without a same-origin browser target, private upstream configuration, or an
explicit ingress-owned client-IP header. Upstream response bodies are rejected above
`API_UPSTREAM_MAX_RESPONSE_BYTES` before JSON parsing.

The former auth-only environment names remain accepted as deprecated aliases for one migration
window. Setting a canonical and deprecated name to different values fails startup. New deployments
and documentation must use the `API_*` names because the adapter now serves more than auth.

### Operation Mapping

| Browser operation | Adapter behavior |
|---|---|
| Login | Sends `{ username: email, password }` to `POST /v1/login`, validates the envelope/opaque credential/user, then sets the API session cookie. |
| Register | Sends a non-identifying generated username plus `{ email, nickname: name, password }` to `POST /v1/register`, then logs in by email and sets the cookie. |
| Current session | Reads the server-only opaque session cookie, sends `Authorization: Bearer ...` to `GET /v1/users/profile`, and maps the API user to `{ id, email, name }`. |
| Logout | Sends the opaque credential to `POST /v1/logout`, then expires the exact-scope cookie. An upstream `401` is idempotent success because the server session is already absent; an availability failure is reported while local credential custody is still removed. |

The generated username has the form `user_<random-id>` and exists only to satisfy the API starter's
current account model. Browser identity remains email-based. Registration accepts a 2-50 character
name, an email up to 100 characters, and an 8-50 character password so the adapter cannot accept a
payload that the Go DTO rejects.

Registration and automatic login are two upstream calls, not one transaction. If registration
succeeds and login then times out or becomes unavailable, the account remains created; the user can
recover through the normal login flow. Products requiring atomic account creation plus session
issuance should add that capability to the API contract instead of hiding compensation in Web.

### Session And Security Semantics

- Production stores the opaque bearer token in `__Host-luas_auth`; non-production uses `luas_auth`.
  The cookie is HttpOnly, `Secure` in production, `SameSite=Lax`, `Path=/`, has no `Domain`, and
  expires no later than the API-owned `expires_in` lifetime. Browser code never receives the token.
- The Web adapter validates only the bounded opaque credential shape and positive `expires_in`
  value before accepting login success. It does not decode identity or authorization claims from
  the credential; the Go API resolves current session and account state from persistence.
- Login, registration, and logout require exact same-origin browser mutations. The adapter forwards
  no incoming cookies, authorization headers, or arbitrary paths to Go. The allowed browser origin
  comes from validated `NEXT_PUBLIC_APP_URL`, not a proxy-normalized internal request URL.
- `API_CLIENT_IP_HEADER` names the ingress-owned header from which one validated IP is forwarded as
  `X-Forwarded-For`; missing, malformed, or comma-separated values are not forwarded. The Go API
  must trust only the Web adapter network through
  `SERVER_TRUSTED_PROXIES`; this preserves endpoint-specific source-IP and subject quotas without
  trusting a browser-supplied forwarding chain. Configure the ingress to overwrite a single-value
  header such as `X-Real-IP`; do not pass an appended forwarding chain into this setting.
- Canonical upstream status, `error_code`, field ownership, `request_id`, and rate-limit headers are
  preserved. Backend messages and field messages are replaced with adapter-owned generic text;
  `nickname` validation ownership maps to the browser `name` field and generated `username` errors
  are not exposed as user input.
- Every `/api/auth/*` success, validation failure, same-origin rejection, authentication failure,
  and backend-availability response sets `Cache-Control: private, no-store` and varies on `Cookie`.
  The shared response boundary merges existing `Vary`, request-ID, and rate-limit headers instead
  of replacing them. Authenticated browser responses must never rely on framework dynamism alone
  as a cache policy.
- Timeout, network, malformed-envelope, malformed-user, and malformed-token failures become
  `503 COMMON.TIMEOUT` or `503 COMMON.SERVICE_UNAVAILABLE`, never a false unauthenticated session.
- A missing, revoked, idle-expired, absolute-expired, or rejected token becomes
  `401 AUTH.UNAUTHORIZED`. A disabled API user becomes
  `403 AUTH.ACCOUNT_DISABLED`. Availability failures remain retryable and do not redirect to login.

# Authentication Contracts

Luas currently ships two authentication boundaries. They share the global HTTP envelope and
`error_code` vocabulary, but their paths, credentials, user DTOs, and session ownership are not
interchangeable.

## Browser Auth Contract

The Web feature service and development mock BFF use a browser-oriented, same-origin contract:

| Operation | Browser endpoint | Request | Successful `data` |
|---|---|---|---|
| Login | `POST /api/auth/login` | `{ email, password }` | `{ user: { id, email, name, role } }` |
| Register | `POST /api/auth/register` | `{ name, email, password }` | `{ user: { id, email, name, role } }` |
| Current session | `GET /api/auth/me` | none | `{ user: { id, email, name, role } }` |
| Logout | `POST /api/auth/logout` | none | `{ success: true }` |

The development implementation owns an HttpOnly signed mock cookie. Unsafe mock BFF operations
must reject cross-origin browser requests before reading a body or mutating state.

## Go API User Starter Contract

The Go `user` starter exposes JWT-oriented endpoints under `/v1`:

| Operation | API endpoint | Request | Successful `data` |
|---|---|---|---|
| Login | `POST /v1/login` | `{ username, password }` | `{ access_token, user }` |
| Register | `POST /v1/register` | `{ username, password, email, nickname?, phone? }` | API user |
| Request password reset | `POST /v1/password/reset` | `{ email }` | Generic message |
| Confirm password reset | `POST /v1/password/reset/confirm` | `{ token, new_password }` | Generic message |
| Profile | `GET /v1/users/profile` | Bearer token | API user |

The API user uses numeric identity and backend fields such as `username`, `nickname`, and
`status`. It does not currently emit the Web shell's `name` and `role` view.

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

## Production Integration Status

Luas does not yet ship a production adapter between these contracts. Pointing
`NEXT_PUBLIC_API_URL` directly at the Go API does not make the default Web auth flow work because
the endpoint paths, request DTOs, response DTOs, and credential ownership differ.

The required P1 integration is a same-origin production auth adapter that:

1. maps browser login, registration, current-session, and logout operations to the Go API;
2. keeps API access tokens out of browser-readable storage;
3. defines token expiry, cookie attributes, logout, and revocation behavior;
4. maps the API user to one documented browser-safe user view without inventing permissions;
5. preserves canonical errors while suppressing backend detail from UI copy;
6. enforces same-origin mutation protection and endpoint-specific abuse limits; and
7. proves the flow with API, Web, and browser tests.

Until that adapter exists, the Go auth starter is ready for API consumers and the Web mock flow is
ready for local scaffold development, but the combined production auth flow is not ready-to-use.

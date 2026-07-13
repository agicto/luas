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
| Profile | `GET /v1/users/profile` | Bearer token | API user |

The API user uses numeric identity and backend fields such as `username`, `nickname`, and
`status`. It does not currently emit the Web shell's `name` and `role` view.

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

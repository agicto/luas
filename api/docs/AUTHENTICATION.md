# Authentication Sessions

The default `user` starter owns browser-user authentication for the Go API. Its HTTP contract is
[`../../contracts/AUTHENTICATION.md`](../../contracts/AUTHENTICATION.md); this document owns the API
implementation, operations, and replacement boundary.

## Credential Model

A successful login creates one authentication session and returns a 32-byte CSPRNG credential as
unpadded base64url. The plaintext exists only in the login response and caller custody. Persistence
stores its SHA-256 hash, a random non-semantic session ID, user ownership, absolute and idle expiry,
last activity, and terminal revocation metadata.

The credential is opaque. It contains no identity, organization, role, permission, or expiry claim.
Every protected request resolves current session and account state from persistence through
`domain.SessionAuthenticator`. API keys remain the machine-to-machine credential; do not use user
sessions as API key scopes or service credentials.

## Lifecycle

- Login creates a separate session without invalidating other devices.
- Authentication fails closed with `503 COMMON.SERVICE_UNAVAILABLE` when session state cannot be
  verified. Unknown, revoked, idle-expired, and absolute-expired credentials return
  `401 AUTH.UNAUTHORIZED`.
- A disabled account returns `403 AUTH.ACCOUNT_DISABLED` and revokes all current sessions. Account
  deletion also revokes sessions inside the account transaction.
- Password change and successful password reset update the password hash and revoke every existing
  session in the same database transaction. The caller signs in again.
- `POST /v1/logout` revokes only the presented session before returning success.
- Activity extends idle expiry at the configured touch interval. This keeps authentication to one
  indexed read on every request while avoiding one write per request.

The session row deliberately stores no IP address, user agent, device label, browser fingerprint,
plaintext credential, or provider payload. Downstream products may add reviewed device policy, but
must define retention, user visibility, proxy trust, and privacy ownership first.

## Policy And Cleanup

```dotenv
AUTH_SESSION_TTL=720h
AUTH_SESSION_IDLE_TIMEOUT=168h
AUTH_SESSION_TOUCH_INTERVAL=5m
AUTH_SESSION_RETENTION=720h
```

Run cleanup repeatedly from a scheduler or one-shot job:

```bash
/app/luas auth-session:prune --batch=500
```

The command removes only revoked or expired rows beyond the configured retention cutoff, processes
at most 10,000 rows per invocation, and prints only a count. It never prints session IDs, token
hashes, users, or revocation details. Zero retention permits cleanup after a session becomes
terminal; it does not remove active sessions.

## Deployment And Upgrade

Apply `2026_04_27_000003_create_authentication_sessions_table` before deploying code that issues
opaque sessions. The table is additive, so the prior application can run during migration. The new
application intentionally rejects `JWT_SECRET` and `JWT_EXPIRE_DAYS` and does not accept previously
issued JWTs. Users sign in again after the upgrade.

Rolling the application back does not make old JWTs recoverable. A rollback requires the prior
binary and its prior secret configuration; newly issued opaque credentials are unknown to that
binary. The additive session table may remain during an application rollback and should be dropped
only after the rollback window and retention decision are closed.

## Replacement Boundary

High-scale or identity-provider-backed products may replace `domain.SessionAuthenticator` and
`domain.AuthenticationSessionMaintainer` while preserving public error semantics and server-only
browser custody. A replacement must retain immediate revocation, current account-state checks,
bounded credentials, fail-closed availability behavior, and password-event invalidation. Do not
restore long-lived stateless user JWTs as an undocumented compatibility path.

## Verification

```bash
go test ./internal/modules/user ./database/migrations
go test -race ./internal/modules/user
go run ./cmd/luas auth-session:prune --batch=1
```

Cross-boundary verification also runs the Web adapter tests listed in
[`../../web/docs/AUTHENTICATION.md`](../../web/docs/AUTHENTICATION.md).

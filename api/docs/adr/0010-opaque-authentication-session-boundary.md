# ADR 0010: Opaque Authentication Session Boundary

## Status

Accepted

## Context

The default user starter previously issued seven-day stateless JWTs. Web logout deleted its
HttpOnly cookie but could not revoke the bearer credential; password change, password reset, account
disablement, and account deletion could not immediately invalidate already issued tokens. Claims
also risked becoming stale authorization evidence as organizations and permissions grew.

A short access JWT plus rotating refresh token would reduce access-token lifetime, but adds two
credentials, rotation/reuse concurrency, browser refresh coordination, and more ambiguous failure
states. Luas already provides API keys for machine access and does not need stateless user tokens to
cross service boundaries by default.

## Decision

Use one opaque server-side authentication session for signed-in users. Issue at least 256 bits of
CSPRNG entropy, persist only SHA-256, and resolve the session together with current user state on
every protected request. Enforce absolute and idle expiry, throttle activity writes, revoke the
presented session on logout, and revoke all sessions transactionally for password and account
security events.

Expose authentication through `domain.SessionAuthenticator` and retention through
`domain.AuthenticationSessionMaintainer`. Keep the Web credential in a same-origin HttpOnly cookie;
the production adapter forwards it only to fixed Go API paths. Keep API keys as the machine
credential and resolve organization/permission policy from current persistence rather than session
claims.

Remove JWT runtime code and dependency. Reject non-empty legacy JWT configuration so deployments do
not believe old tokens remain valid. The upgrade intentionally signs every user out once.

## Consequences

- Logout and security events gain immediate server-side revocation.
- Protected requests add one indexed session/user read; activity writes occur only after the touch
  interval. Database availability is now part of authentication availability and fails closed.
- Session persistence adds retention and cleanup ownership, served by a bounded operator command.
- The stored row remains privacy-minimized and cannot support a device-management UI without an
  explicit downstream schema and product contract.
- Multi-region applications must place session authority in a consistent shared store or replace the
  domain seams; accepting replica lag as delayed revocation requires an explicit risk decision.
- Prior JWTs and new opaque credentials are mutually incompatible across application rollback.

## Verification Impact

- service and repository tests cover entropy shape, hash-only persistence, expiry, touch throttling,
  revocation, disabled users, password events, retention, and bounded pruning;
- middleware and handler tests cover public 401/403/503 semantics and logout revocation;
- migration tests execute Up and Down and assert that no plaintext/fingerprint columns exist;
- Web tests cover strict opaque-credential parsing, bounded cookie lifetime, remote logout, 401
  idempotency, outage reporting, and server-only custody;
- root governance blocks stale JWT runtime/configuration guidance and keeps API, Web, contracts,
  commands, and docs aligned.

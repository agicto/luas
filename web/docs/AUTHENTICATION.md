# Authentication Runtime Guide

Luas ships one authentication contract with two resolution modes. The mode says who can
authoritatively resolve the current browser session; it does not change the `/auth/*` HTTP
contract.

## Resolution Modes

| Mode | Selected when | Initial protected render |
|---|---|---|
| `mock-session` | The mock BFF is available and `NEXT_PUBLIC_API_URL` targets the same-origin `/api` route | The protected Server Component verifies the signed Luas cookie and passes a definitive user or unauthenticated result to the client store. |
| `client-session` | A real API, a production same-origin proxy, or any non-mock target owns authentication | The client store resolves `/auth/me` once with browser credentials. |

`src/config/mock-bff.ts` owns mock BFF availability, while
`src/features/auth/server/auth-runtime.ts` combines that fact with the API target to select the
authentication mode. Do not infer either decision separately in middleware, layouts, or feature
code.

## Protected Route Flow

In `mock-session` mode:

```text
request -> middleware verifies mock cookie -> protected Server Component reads session
        -> AuthProvider creates an isolated ready store -> AuthGuard renders immediately
```

In `client-session` mode:

```text
request -> middleware passes through -> AuthProvider creates an isolated idle store
        -> one credentialed /auth/me request -> AuthGuard renders or redirects
```

The store uses one status value (`idle`, `loading`, `authenticated`, or `unauthenticated`) instead
of several independently mutable booleans. Each `AuthProvider` creates its own Zustand store;
never hydrate a module-level auth singleton from Server Component data because that can leak one
request's user into another request.

## Security Boundary

- `middleware.ts` verifies only the Luas mock session. It deliberately passes through in
  `client-session` mode because Luas does not own or understand a downstream API's credentials.
- `AuthGuard` is navigation and rendering UX, not an authorization boundary.
- Every real API endpoint, Route Handler, and Server Action must authenticate and authorize the
  operation independently.
- Luas does not forward arbitrary incoming cookies to an external API from the Next.js server.
  Downstream apps may replace `resolveAuthBootstrap()` with a server adapter only when credential
  ownership, forwarding policy, cache behavior, and failure handling are explicitly defined.
- Only serializable, client-safe user fields belong in `AuthBootstrap`. Never pass access tokens,
  session secrets, or raw cookies through provider props.

## Downstream Adaptation

1. Keep `client-session` when the browser owns a cross-origin cookie or token exchange.
2. For a same-origin BFF with a server-readable session, replace the mock resolver with a real
   server adapter and return the existing `AuthBootstrap` union.
3. Update middleware only if that adapter can verify the real session locally. Otherwise keep
   middleware permissive and rely on the API plus `AuthGuard`.
4. Preserve the provider-owned store so request isolation and initialization deduplication remain
   intact.
5. Verify authenticated, unauthenticated, expired-session, API-unavailable, and logout flows in a
   real browser before deployment.

## Verification

```bash
pnpm exec vitest run src/test/auth-runtime-mode.test.ts src/test/auth-store.test.ts src/test/auth-runtime-boundary.test.ts
pnpm type-check
pnpm lint
pnpm build
```

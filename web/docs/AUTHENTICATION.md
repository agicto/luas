# Authentication Runtime Guide

The Web shell owns one browser authentication contract with three resolution modes. The mode says
who can authoritatively resolve the current browser session; it does not change the Web `/auth/*`
contract.

The Go `user` starter owns a separate JWT contract. Luas ships an explicit same-origin production
adapter between them; pointing the browser directly at Go remains incorrect because paths, DTOs,
user views, and credential ownership still differ. See
[`../../contracts/AUTHENTICATION.md`](../../contracts/AUTHENTICATION.md) for the canonical mapping.

## Resolution Modes

| Mode | Selected when | Initial protected render |
|---|---|---|
| `mock-session` | The mock BFF is available and `NEXT_PUBLIC_API_URL` targets the same-origin `/api` route | The protected Server Component verifies the signed Luas cookie and passes a definitive user or unauthenticated result to the client store. |
| `api-session` | `API_ADAPTER_ENABLED=true` and `NEXT_PUBLIC_API_URL` targets same-origin `/api` | The protected Server Component reads the HttpOnly API cookie, resolves `/v1/users/profile` through the fixed adapter, and passes authenticated, unauthenticated, forbidden, or unavailable state to the client store. |
| `client-session` | A real API, a production same-origin proxy, or any non-mock target owns authentication | The client store resolves `/auth/me` once with browser credentials. |

`src/config/mock-bff.ts` owns mock BFF availability, `src/config/server-env.ts` owns production
adapter availability, and `src/features/auth/server/auth-runtime.ts` combines those facts with the
browser API target. Do not infer the mode separately in middleware, layouts, or feature code.

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

In `api-session` mode:

```text
request -> protected Server Component reads API cookie -> fixed Go profile request
        -> AuthProvider creates an isolated ready/retryable store -> AuthGuard renders outcome
```

The store uses one status value instead of several independently mutable booleans. Each
`AuthProvider` creates its own Zustand store; never hydrate a module-level auth singleton from
Server Component data because that can leak one request's user into another request.

## Resolution Outcomes

| Store status | Evidence | Guard behavior |
|---|---|---|
| `idle` | Client-owned session has not been checked | Block content until resolution starts. |
| `loading` | One `/auth/me` request is in flight | Show the protected loading state. |
| `authenticated` | The server bootstrap or `/auth/me` returned a user | Render protected content. |
| `unauthenticated` | HTTP `401`, `AUTH.UNAUTHORIZED`, or invalid credentials | Redirect to login with `returnUrl`. |
| `forbidden` | HTTP `403`, `AUTH.FORBIDDEN`, or `AUTH.ACCOUNT_DISABLED` | Keep content blocked and show a non-redirecting access-denied state. |
| `unavailable` | Network, timeout, rate limit, `5xx`, malformed, or unknown failure | Preserve the session as unknown and offer an explicit retry. |

Do not branch on backend message text. A failed availability check must not mutate an unknown
session into `unauthenticated`; doing so creates false logout events and retrying login cannot fix
the underlying dependency failure. Retries from `forbidden` and `unavailable` reuse the same
deduplicated initializer. Successful `/auth/me` payloads must also pass the shared runtime
`isAuthResponse()` guard; TypeScript types alone do not validate external JSON.

## Authentication Mutations

`login`, `register`, `me`, and `logout` services receive external JSON as `unknown` and validate
their successful payloads through `src/features/auth/utils/auth-response.ts`. A malformed `2xx`
response becomes `CLIENT.INVALID_RESPONSE`; it must not authenticate, redirect, or report a
successful logout.

Login and registration use manual error presentation because their forms own the recovery
workflow:

- Mark the React Query mutation with `errorHandling: 'local'` and disable retries. The HTTP client
  only normalizes errors; it never owns user-facing presentation.
- Select copy from normalized `error_code` and HTTP status through `resolveAuthErrorKey()`; never
  display a backend `message` directly.
- Treat backend `errors` only as field ownership. Render the local, reviewed translation for each
  matching field so backend implementation detail cannot become user-facing copy.
- Announce the form-level error with an alert, associate field feedback through
  `aria-describedby`, and clear stale mutation errors when the user edits the form.
- Do not redirect on network, timeout, malformed success, validation, or authentication failures.

Logout is idempotent from the user's perspective. Success and `401` / `AUTH.UNAUTHORIZED` both
clear local auth state and navigate to the logged-out route. Availability failures and malformed
success payloads preserve local state and show localized failure feedback because the server-side
session outcome is unknown.

## Security Boundary

- Unsafe mock BFF routes call `guardSameOriginMutation()` after the production availability guard
  and before reading bodies or mutating state. The guard rejects cross-site and same-site sibling
  browser writes using `Sec-Fetch-Site` and exact `Origin` comparison; clients without browser
  fetch metadata remain usable for tests and automation.
- The signed mock session cookie is HttpOnly and `SameSite=Lax`. Production uses a Secure
  `__Host-luas_session` cookie with `Path=/`, no Domain attribute, and an exact-scope expiry on
  logout.
- The production adapter uses a separate Secure `__Host-luas_auth` cookie. It stores the Go bearer
  token HttpOnly, caps persistence at 30 days, and expires no later than the JWT `exp`. Web does not
  trust token claims for authorization; Go validates the token on profile resolution.
- Adapter routes call only fixed Go auth paths, reject redirects, never forward incoming cookies or
  authorization headers, and replace backend messages with generic local text. Configured
  `API_CLIENT_IP_HEADER` input is parsed as one IP before becoming `X-Forwarded-For`.
- Every auth Route Handler finalizes success and failure responses with
  `Cache-Control: private, no-store` and `Vary: Cookie`. This includes malformed input,
  cross-origin rejection, missing sessions, upstream failures, and disabled-backend responses.
  `src/server/http/private-response.ts` merges existing `Vary`, request-ID, and rate-limit headers;
  do not replace this with an assumption that a cookie read automatically protects intermediary
  caches.
- The API must set `SERVER_TRUSTED_PROXIES` to the Web adapter network, never a trust-all range, so
  Go's public-auth source and subject quotas remain meaningful behind the adapter.
- Demo credentials live in `src/features/auth/server/mock-identity.ts`. The login Server Component
  passes the preset to the Client Component only in `mock-session` mode. Production adapter and
  client-owned modes stay blank; an explicit production mock opt-in is visibly demo-only.
- `middleware.ts` verifies only the Luas mock session. It deliberately passes through in
  `api-session` and `client-session` modes because only the protected Server Component or downstream
  identity provider can authoritatively resolve those credentials.
- `AuthGuard` is navigation and rendering UX, not an authorization boundary.
- Every real API endpoint, Route Handler, and Server Action must authenticate and authorize the
  operation independently.
- Luas does not forward arbitrary incoming cookies to an external API from the Next.js server.
  Downstream apps may replace `resolveAuthBootstrap()` with a server adapter only when credential
  ownership, forwarding policy, cache behavior, and failure handling are explicitly defined.
- Only serializable, client-safe user fields belong in `AuthBootstrap`. Never pass access tokens,
  session secrets, or raw cookies through provider props.
- `AuthUser` contains only `id`, `email`, and `name`. Do not add a mock `role`; authorization fields
  belong to a real permission contract when that starter exists.
- `src/features/auth/utils/auth-user.ts` owns the shared runtime user predicate for mock cookies
  and client session responses. `src/features/auth/utils/auth-response.ts` composes that predicate
  into endpoint success contracts. Keep required-field semantics aligned there.

## Downstream Adaptation

1. Read `contracts/AUTHENTICATION.md`; do not assume the Go JWT endpoints implement the Web DTOs.
2. Use the shipped `api-session` adapter for the Luas Go starter; configure the private API URL,
   timeout, ingress client-IP header, and API trusted proxies together.
3. Keep `client-session` when another identity provider owns a cross-origin cookie or token exchange.
4. Replace `auth-adapter-route.ts` only when the downstream server owns a different authoritative
   session contract and can return the existing `AuthBootstrap` union.
5. Update middleware only if that adapter can verify the real session locally. Otherwise keep
   middleware permissive and rely on the API plus `AuthGuard`.
6. Preserve the provider-owned store so request isolation and initialization deduplication remain
   intact.
7. Verify authenticated, unauthenticated, forbidden, API-unavailable, retry recovery, expired-session,
   and logout flows in a real browser before deployment.

## Verification

```bash
pnpm exec vitest run src/test/auth-adapter-route.test.ts src/test/auth-session-cookie.test.ts src/test/go-api-auth-adapter.test.ts src/test/auth-route-backend.test.ts src/test/auth-runtime-mode.test.ts src/test/auth-store.test.ts src/test/auth-guard-recovery.test.tsx src/test/auth-service-contract.test.ts src/test/auth-form-errors.test.tsx src/test/auth-logout.test.tsx
pnpm type-check
pnpm lint
pnpm build
```

# Mock BFF Replacement Guide

The Web mock BFF is the development-only set of Next.js route handlers under `src/app/api/**`.
It lets the web shell run before a real backend is available. It must preserve the shared HTTP
contract, but it is not the production API.

Use this guide when turning Luas into a downstream app.

## Runtime Modes

| Mode | `NEXT_PUBLIC_API_URL` | `MOCK_BFF_ENABLED` | Use |
|---|---|---|---|
| Local scaffold development | `/api` | unset or `false` | Uses `src/app/api/**` mock route handlers. |
| Demo deployment without a backend | `/api` | `true` | Demo-only. Also set a strong `SESSION_SECRET`. |
| Production downstream app | Production endpoint or same-origin adapter | unset or `false` | Uses production contracts and keeps mock routes disabled. |

If production uses the same `/api` path, route that path to the real API at the deployment layer.
If production calls a cross-origin API, the API must allow credentialed browser requests from the
web origin because the default HTTP client sends credentials.
Production runtime fails fast when `MOCK_BFF_ENABLED=true` without a strong `SESSION_SECRET`;
normal downstream production runtime and production builds can omit this mock-only secret.

Authentication is an explicit exception to generic base-URL replacement: the default Web
`/auth/*` cookie/session contract and Go `/v1/*` JWT contract have different paths, DTOs, and
credential ownership. Use [`../../contracts/AUTHENTICATION.md`](../../contracts/AUTHENTICATION.md)
to build the production adapter instead of pointing the existing auth service directly at Go.

Authentication mode is derived from both mock availability and the API target. Local development
or an explicit demo using same-origin `/api` selects `mock-session`, so the protected Server
Component can verify the Luas cookie without a client `/auth/me` waterfall. External APIs and
production same-origin proxies select `client-session`; middleware must not interpret their
credentials as a Luas mock cookie. See [`AUTHENTICATION.md`](AUTHENTICATION.md).

## Replacement Checklist

1. Confirm the real API follows [`../../contracts/README.md`](../../contracts/README.md), especially
   response envelopes, `error_code`, `request_id`, and validation error shapes.
2. Set `NEXT_PUBLIC_API_URL` to the production base URL or same-origin adapter path whose browser
   contract is documented.
3. Keep feature services only when their endpoint paths and DTOs match that browser contract. For
   auth, implement the adapter contract before retaining `src/features/auth/services/auth-service.ts`.
4. Replace mock server state:
   - `src/app/api/**/route.ts`
   - `src/app/api/_shared/*`
   - `src/features/auth/server/session.ts`
   - `src/features/example/server/mock-example-store.ts`
   - `src/features/auth/server/mock-identity.ts`
   - `src/config/mock-session.ts`
   Keep `AuthBootstrap` and the provider-owned store. Replace `resolveAuthBootstrap()` only if the
   downstream server can safely and authoritatively resolve the real session.
5. Leave `MOCK_BFF_ENABLED=false` for production unless the deployment is explicitly demo-only.
6. When all mock route handlers are removed, remove or adapt `src/test/mock-bff-route-contract.test.ts`
   because that test is a scaffold guardrail for existing mock routes.

## Keeping a Local Mock

Some downstream apps keep mock routes for local or preview development. In that case:

- Every `src/app/api/**/route.ts` file must call `guardMockBffRoute()` before reading request bodies
  or touching mock state.
- Every `POST`, `PUT`, `PATCH`, or `DELETE` handler must then call
  `guardSameOriginMutation(request)` before reading request bodies or touching mock state.
- Mock success responses must use `apiSuccessResponse()` so they emit `{ code: 0, message: "success", data }`.
- Mock error responses must use `{ code, error_code, message, request_id? }`.
- Use `ApiErrorCode` for mock responses and `ClientErrorCode` only for frontend-only fallback errors.
- Keep demo credentials in a `server-only` module. Do not import them into shared config, client
  hooks, translations, or Client Components. Normal production remains blank; an explicitly
  enabled `mock-session` demo receives the preset from its Login Server Component.
- Run `src/test/mock-bff-route-contract.test.ts` and `src/test/error-code-vocabulary.test.ts` after
  adding or deleting mock route handlers.

## Verification

Run the Web verification tier after replacing or deleting the mock BFF:

```bash
pnpm type-check
pnpm lint
pnpm exec vitest run
pnpm build
```

For production integration, also run a browser or curl check that proves adapter mappings, auth,
CORS, credentials, CSRF protection, and error responses match the owning contracts.

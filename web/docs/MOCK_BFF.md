# Mock BFF Replacement Guide

The Web mock BFF is the development-only behavior behind Next.js route handlers under
`src/app/api/**`. It lets the web shell run before a real backend is available. Auth, default API
key, optional organization, permission, notification, and asset Route Handlers are hybrid entry points: they select either this mock
behavior or the shipped production API adapter. A route location under `/api` does not by itself
make behavior mock or production.

Use this guide when turning Luas into a downstream app.

## Runtime Modes

| Mode                              | Browser target                                      | Server switches                                  | Use                                                                                                            |
| --------------------------------- | --------------------------------------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- |
| Local scaffold development        | `/api`                                              | adapter off; mock unset/false                    | Uses mock route behavior.                                                                                      |
| Demo deployment without a backend | `/api`                                              | `MOCK_BFF_ENABLED=true`                          | Demo-only. Also set a strong `SESSION_SECRET`.                                                                 |
| Production Luas Go API            | `/api`                                              | `API_ADAPTER_ENABLED=true` plus adapter settings | Auth and explicitly enabled starter routes use fixed production handlers; unrelated mock routes stay disabled. |
| Other production backend          | Contract-compatible endpoint or replacement adapter | adapter/mock off                                 | Uses the downstream production contract and keeps Luas mock behavior disabled.                                 |

If production uses the same `/api` path, route that path to the real API at the deployment layer.
If production calls a cross-origin API, the API must allow credentialed browser requests from the
web origin because the default HTTP client sends credentials.
Production runtime fails fast when `MOCK_BFF_ENABLED=true` without a strong `SESSION_SECRET`;
normal downstream production runtime and production builds can omit this mock-only secret.

Authentication is an explicit exception to generic base-URL replacement: the default Web
`/auth/*` cookie/session contract and Go `/v1/*` JWT contract have different paths, DTOs, and
credential ownership. Use the production adapter documented in
[`../../contracts/AUTHENTICATION.md`](../../contracts/AUTHENTICATION.md) instead of pointing the
existing auth service directly at Go.

Authentication mode is derived from adapter ownership, mock availability, and the browser target.
The production adapter selects `api-session`; local/demo mock behavior selects `mock-session`;
external identity systems select `client-session`. Middleware interprets only the signed mock
cookie. See [`AUTHENTICATION.md`](AUTHENTICATION.md).

## Replacement Checklist

1. Confirm the real API follows [`../../contracts/README.md`](../../contracts/README.md), especially
   response envelopes, `error_code`, `request_id`, and validation error shapes.
2. Set `NEXT_PUBLIC_API_URL` to the production base URL or same-origin adapter path whose browser
   contract is documented.
3. Keep feature services only when their endpoint paths and DTOs match that browser contract. For
   the Luas Go auth starter, retain the browser service and configure the shipped adapter.
4. Replace mock server state:
   - `src/app/api/**/route.ts`
   - `src/app/api/_shared/*`
   - `src/features/auth/server/session.ts`
   - `src/features/auth/server/auth-adapter-route.ts`
   - `src/features/auth/server/go-api-auth-adapter.ts`
   - `src/features/api-key/server/mock-api-key-store.ts`
   - `src/features/api-key/server/api-key-route.ts`
   - `src/features/organization/server/mock-organization-store.ts`
   - `src/features/organization/server/organization-route.ts`
   - `src/features/organization/server/organization-lifecycle-route.ts`
   - `src/features/permission/server/mock-permission-store.ts`
   - `src/features/permission/server/permission-route.ts`
   - `src/features/notification/server/mock-notification-store.ts`
   - `src/features/notification/server/notification-route.ts`
   - `src/features/asset/server/mock-asset-store.ts`
   - `src/features/asset/server/asset-route.ts`
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

- Mock-only `src/app/api/**/route.ts` files call `guardMockBffRoute()` before reading request bodies
  or touching mock state. Hybrid auth routes call `resolveAuthRoute()` first and mutate mock state
  only when it selects the mock BFF. Hybrid organization routes call `resolveOrganizationRoute()`
  and require `NEXT_PUBLIC_OPTIONAL_FEATURES=organization` before touching their mock store.
  Hybrid permission routes call `resolvePermissionRoute()`, require the explicit
  `organization,permission` dependency selection, and preserve owner bypass plus delegated
  dominance checks in the mock store.
  Hybrid notification routes call `resolveNotificationRoute()`, require the explicit `notification`
  selection, isolate state by authenticated user, and preserve read high-water and preference
  semantics.
  Hybrid asset routes call `resolveAssetRoute()`, require the explicit `asset` selection, isolate
  metadata and bytes by authenticated user, and preserve idempotency, inspection, lifecycle,
  attachment download, and ownership non-disclosure. Transfer grants remain short-lived and are not
  persisted in browser state.
  Hybrid API key routes call `resolveApiKeyRoute()` and follow the one-time secret contract in
  [`API_KEYS.md`](API_KEYS.md).
- Every `POST`, `PUT`, `PATCH`, or `DELETE` handler must then call
  `guardSameOriginMutation(request)` before reading request bodies or touching mock state.
- The guard compares browser `Origin` with validated `NEXT_PUBLIC_APP_URL`, so a public host remains
  authoritative when Next.js receives an internal reverse-proxy URL.
- Mock success responses must use `apiSuccessResponse()` so they emit `{ code: 0, message: "success", data }`.
- JSON request bodies must use `readJsonBody()` and map `too_large` to
  `413 COMMON.REQUEST_TOO_LARGE`; do not call `request.json()` directly in a Route Handler.
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

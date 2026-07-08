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
| Production downstream app | Real API origin or same-origin proxy | unset or `false` | Uses the real API and keeps mock routes disabled. |

If production uses the same `/api` path, route that path to the real API at the deployment layer.
If production calls a cross-origin API, the API must allow credentialed browser requests from the
web origin because the default HTTP client sends credentials.
Production runtime fails fast when mock auth can run without a strong `SESSION_SECRET`; production
builds may omit runtime-only secrets during `phase-production-build`.

## Replacement Checklist

1. Confirm the real API follows [`../../contracts/README.md`](../../contracts/README.md), especially
   response envelopes, `error_code`, `request_id`, and validation error shapes.
2. Set `NEXT_PUBLIC_API_URL` to the real API base URL or to a same-origin proxy path.
3. Keep feature services such as `src/features/auth/services/auth-service.ts` when the endpoint paths
   and DTOs still match. Update service methods when the real API uses different paths.
4. Replace mock server state:
   - `src/app/api/**/route.ts`
   - `src/app/api/_shared/*`
   - `src/features/auth/server/session.ts`
   - `src/features/example/server/mock-example-store.ts`
   - `authConfig.demoUser` in `src/config/auth.ts`
5. Leave `MOCK_BFF_ENABLED=false` for production unless the deployment is explicitly demo-only.
6. When all mock route handlers are removed, remove or adapt `src/test/mock-bff-route-contract.test.ts`
   because that test is a scaffold guardrail for existing mock routes.

## Keeping a Local Mock

Some downstream apps keep mock routes for local or preview development. In that case:

- Every `src/app/api/**/route.ts` file must call `guardMockBffRoute()` before reading request bodies
  or touching mock state.
- Mock success responses must use `apiSuccessResponse()` so they emit `{ code: 0, message: "success", data }`.
- Mock error responses must use `{ code, error_code, message, request_id? }`.
- Use `ApiErrorCode` for mock responses and `ClientErrorCode` only for frontend-only fallback errors.
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

For cross-origin production APIs, also run a browser or curl check that proves auth, CORS,
credentials, and error responses match the shared contract.

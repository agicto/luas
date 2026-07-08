# Adding a Web Feature

Use this checklist for feature-first additions under `web/`.

## Placement

- Put feature-owned code in `src/features/<feature>/`.
- Put route entry points in `src/app/...`.
- Put generic reusable UI in `src/components/common/`.
- Do not edit `src/components/ui/` unless intentionally updating shadcn-style primitives.

Recommended feature layout:

```text
src/features/<feature>/
├── components/
├── hooks/
├── services/
├── server/
├── types.ts
└── index.ts
```

## HTTP

- Use `src/http/request.ts` for API calls.
- Treat backend `error_code` as the stable error branch key.
- Keep DTO fields aligned with `contracts/README.md`.
- If you add a development mock BFF route under `src/app/api/`, call `guardMockBffRoute()` before reading the request body or touching mock state. Return successful payloads with `apiSuccessResponse()` so mock routes preserve `{ code: 0, message: "success", data }`. `src/test/mock-bff-route-contract.test.ts` fails if a route misses the guard, bypasses shared response helpers, or emits legacy underscore-style error codes.
- Do not rely on mock BFF routes for downstream production apps; point `NEXT_PUBLIC_API_URL` at the real API or explicitly document a demo-only production opt-in. See `docs/MOCK_BFF.md` for replacement and deletion steps.
- Read runtime configuration through `@/config/env`; `src/test/env-contract.test.ts` fails if production source reads `process.env` directly.

## State

- Use React Query for server state.
- Use Zustand only for shared UI/session state.
- Keep services stateless; hooks own query and mutation behavior.
- Place `QueryProvider` at the nearest route group that needs React Query.
- Use `AuthenticatedProviders` only for protected route groups that need session initialization.
- Keep public site pages free of auth-store subscriptions unless the feature intentionally becomes authenticated.
- `src/test/public-route-boundary.test.ts` fails if public `(site)` routes pull in auth, query, HTTP, mock BFF, mock session, or Zustand runtime dependencies.

## UI

- Prefer Server Components by default.
- Keep Client Components small and leaf-level.
- Use existing shadcn-style primitives and semantic Tailwind tokens.
- Add i18n keys for user-facing text.
- When adding a locale, update `src/i18n/locales.ts`, every translation module, and the i18n config tests/docs.
- Do not duplicate locale detection logic; update `src/i18n/locale-resolution.ts` and its tests when changing cookie/header/default behavior.

## Verification

Run from `web/`:

```bash
pnpm type-check
pnpm lint
pnpm test -- --run
pnpm build
```

For repo-wide verification, run `make check` at the root.

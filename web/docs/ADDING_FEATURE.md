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
- Use `getEnvelope()` only when a caller owns pagination `meta` / `links`; normal calls continue to
  receive extracted `data`. Validate security- or state-sensitive successful JSON before caching.
- Treat backend `error_code` as the stable error branch key.
- Keep DTO fields aligned with `contracts/README.md`.
- If you add a development mock BFF route under `src/app/api/`, call `guardMockBffRoute()` before reading the request body or touching mock state. Unsafe handlers (`POST`, `PUT`, `PATCH`, `DELETE`) must then call `guardSameOriginMutation(request)` before parsing or mutation. Read JSON through bounded `readJsonBody()` and map an oversized body to `413 COMMON.REQUEST_TOO_LARGE`. Return successful payloads with `apiSuccessResponse()` so mock routes preserve `{ code: 0, message: "success", data }`. `src/test/mock-bff-route-contract.test.ts` fails if a route misses either required guard, bypasses shared response helpers, or emits legacy underscore-style error codes.
- A hybrid production/mock feature may use a named route resolver and delegate to
  `src/server/api-adapter/`, but every browser path remains an explicit Route Handler. Never accept
  a caller-controlled upstream path or build a catch-all authenticated proxy.
- Do not rely on mock BFF routes for downstream production apps; replace them with contract-compatible production endpoints or a documented adapter. See `docs/MOCK_BFF.md` for replacement and deletion steps.
- Read browser-safe runtime configuration through `@/config/env` and server-only values through `@/config/server-env`; `src/test/env-contract.test.ts` guards direct `process.env` access and client/server leakage.

## State

- Use React Query for server state.
- Use Zustand only for shared UI/session state. Auth state is provider-owned and request-isolated;
  do not create a module-level store containing server-bootstrapped users.
- Keep services stateless; hooks own query and mutation behavior.
- For a scaffold-optional browser workflow, register its canonical name in
  `src/config/optional-features.ts`, gate server pages and Route Handlers, and lazy-load any shell
  integration so the default build performs no optional feature request.
- Place `QueryProvider` at the nearest route group that needs React Query.
- Use `AuthenticatedProviders` only for protected route groups and pass the result of
  `resolveAuthBootstrap()` from their Server Component boundary.
- Keep public site pages free of auth-store subscriptions unless the feature intentionally becomes authenticated.
- `src/test/public-route-boundary.test.ts` fails if public `(site)` routes pull in auth, query, HTTP, mock BFF, mock session, toast, or Zustand runtime dependencies.
- Mount `Toaster` at the nearest route group that emits toast feedback; do not add it to the root layout unless every route genuinely needs that runtime.

## UI

- Prefer Server Components by default.
- Keep Client Components small and leaf-level.
- Use existing shadcn-style primitives and semantic Tailwind tokens.
- Add i18n keys for user-facing text.
- Keep formal site, auth, and console copy within the boundary enforced by `pnpm lint:i18n-copy`; exact brands and technical identifiers are the only normal literal exceptions.
- If a Client Component calls `useT` with a namespace that its route does not already own, add that namespace to `src/i18n/client-message-namespaces.ts` and provide it from the nearest route layout with `RouteMessagesProvider`; never pass the full message tree from root.
- When adding a locale, update `src/i18n/locales.ts`, every translation module, and the i18n config tests/docs.
- Do not duplicate locale detection logic; update `src/i18n/locale-resolution.ts` and its tests when changing cookie/header/default behavior.

## Verification

Run from `web/`:

```bash
pnpm type-check
pnpm lint
pnpm lint:i18n-copy
pnpm test -- --run
pnpm build
```

For repo-wide verification, run `make check` at the root.

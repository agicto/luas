# AGENTS.md - Luas Web SPA

Rules for the static React application under `web-spa/`.

## Scope

The SPA uses Vite, React 19, TypeScript, TanStack Router, TanStack Query,
Zustand, Tailwind 4, shadcn/ui, i18next, Zod, and Vitest. It
builds browser-only assets under `dist/`; no Node.js runtime or server function
may be required in production.

Inspect the owning route, feature, service, and tests first. Use root Luas
skills only when their trigger matches; this surface deliberately does not
duplicate the Next-specific skills under `web/.agents/skills/`.

## Structure

```text
src/app/                 provider, query client, and router composition
src/routes/              thin TanStack file-route entries
src/features/<feature>/  components, hooks, services, store, and types
src/components/ui/       shadcn/ui primitives owned by this deployable unit
src/components/layout/   shared route layouts
src/config/              validated public environment and feature catalog
src/http/                transport, envelopes, errors, and response bounds
src/i18n/                locale resources and browser locale lifecycle
src/lib/                 generic pure helpers
src/styles/              tokens and global styles
src/test/                shared test setup
scripts/                 deterministic build-output and budget checks
docs/                    SPA-specific architecture, security, and deployment
```

## Architecture Rules

- Keep route files declarative and thin. Route entries select a feature page,
  layout, validation, or redirect; business behavior stays in the feature.
- Use the flow `route/component -> feature hook -> feature service ->
src/http/client.ts -> HTTP API`.
- Feature services own fixed endpoint paths and response schemas. Components
  do not call `fetch` directly.
- TanStack Query owns remote/server state. Zustand owns shared browser UI
  state. Local interaction state stays in the component.
- Validate security- or state-sensitive API data with Zod before putting it in
  Query cache.
- Put reusable controls in `components/ui` only when they are genuinely
  feature-neutral. Search before adding a primitive, hook, or utility.
- Do not import source from `web/` or `api/`. Equivalent behavior is
  implemented locally against the same contracts.

## Routing

- Routes are generated from `src/routes/` into `src/routeTree.gen.ts`.
- Never edit `routeTree.gen.ts` manually.
- Preserve TanStack Router type-safe links, params, and search values; do not
  cast around a route mismatch.
- Keep automatic route code splitting enabled.
- Use `import.meta.env.BASE_URL` through the router configuration so builds
  work at `/` and reviewed subpaths.
- Production CDN configuration must rewrite unknown application paths to
  `index.html`; client routing cannot repair a missing edge fallback.

## HTTP And Errors

- Shared success envelope: `{ code: 0, message: "success", data }`.
- Business failures branch on canonical dotted backend `error_code`.
- Browser-owned network, timeout, cancellation, and invalid-response failures
  use `ClientErrorCode`.
- Preserve `request_id` for diagnostics.
- Keep fixed root-relative endpoint paths; do not turn the client into an
  arbitrary proxy.
- Requests use bounded timeouts and response bodies. Do not add automatic
  mutation retries.
- User-facing error copy belongs to i18n mappings, not upstream messages.

Update the owning file under `../contracts/` when public request, response,
status, error, session, or pagination behavior changes.

## Authentication And Security

This is a static browser client, not a BFF:

- Never place bearer tokens, refresh tokens, API keys, or secrets in
  `localStorage`, `sessionStorage`, Zustand persistence, URLs, or `VITE_*`.
- Prefer a same-origin `/api` gateway or Go browser adapter that owns HttpOnly
  cookies, Origin/CSRF enforcement, and fixed upstream mappings.
- The existing Go bearer-login response is not permission to persist its token
  in browser JavaScript.
- Route guards are UX only. The API remains the authorization authority.
- All `VITE_*` values are public build-time data. Validate them in
  `src/config/env.ts`.
- CDN-owned CSP, HSTS, frame, MIME, and referrer headers must follow
  [docs/SECURITY.md](docs/SECURITY.md).

## UI And Internationalization

- Use the configured shadcn/ui New York primitives and semantic tokens before
  adding a new visual system.
- Keep the Luas brand accent Rhine blue. Green is reserved for explicit
  success state and must not become the logo, navigation, selection, or focus
  color.
- Compose the console shell from `SidebarProvider`, `Sidebar`, and
  `SidebarInset`. Keep its open state in the display-preference store so the
  static client does not emit framework-owned UI cookies.
- Keep the console quiet, dense, responsive, and work-focused.
- Preserve visible focus, keyboard operation, accessible names, and stable
  loading/error/empty layouts.
- Use Lucide icons for familiar actions. Icon-only buttons require an
  accessible label and tooltip/title.
- Keep standard button icons and labels on one horizontal line. Use a distinct
  composed control when an interaction genuinely requires a vertical layout.
- Keep cards for repeated records, metrics, and genuinely framed tools.
- All formal user-facing copy uses i18next resources. `en-US` defines the
  source schema; every locale preserves its structure and interpolation names.
- Display preferences may use browser storage; credentials and server-owned
  state may not.

## Authority Map

| Concern                             | Read                                                         |
| ----------------------------------- | ------------------------------------------------------------ |
| Architecture and stack decision     | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)                 |
| Adding a feature                    | [docs/ADDING_FEATURE.md](docs/ADDING_FEATURE.md)             |
| OSS/CDN deployment                  | [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)                     |
| Browser security and authentication | [docs/SECURITY.md](docs/SECURITY.md)                         |
| Internationalization                | [docs/INTERNATIONALIZATION.md](docs/INTERNATIONALIZATION.md) |
| Bundle and runtime performance      | [docs/PERFORMANCE.md](docs/PERFORMANCE.md)                   |
| Shared HTTP behavior                | The matching file under `../contracts/`                      |

## Verification

Use focused proof while iterating:

```bash
corepack pnpm vitest run <test-file>
corepack pnpm type-check
corepack pnpm lint
corepack pnpm test -- --run
corepack pnpm build
```

`pnpm build` verifies that output is static, source maps are absent, and bundle
budgets pass. Use browser verification for changed workflows and responsive
layout. The repository release gate is `cd .. && make check`; run it once after
focused checks pass.

## Do Not

- Add TanStack Start, SSR, server functions, API routes, or a Node production
  server to this surface.
- Copy Next.js Route Handlers, Server Components, or server-only modules from
  `web/`.
- Hide production auth gaps behind a development token store.
- Add product-specific behavior to the scaffold default.
- Commit `dist/`, local environment files, credentials, or generated reports.

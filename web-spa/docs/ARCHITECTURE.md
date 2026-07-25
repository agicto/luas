# Web SPA Architecture

## Decision

`web-spa/` is a pure client-rendered application built with Vite and TanStack
Router. Production consists only of static HTML, JavaScript, CSS, and related
assets. It requires no Node.js process, edge function, server function, or
framework runtime.

The downstream application normally chooses one browser shell:

- `web/` when it needs SSR, server components, same-origin Route Handlers,
  metadata rendering, or the shipped Next.js production adapter;
- `web-spa/` when authenticated/product workflows can use a reviewed API
  gateway and static OSS/CDN delivery is the better operational fit.

Keeping both in Luas demonstrates supported architecture alternatives. A
downstream product should remove the one it does not intend to maintain.

## Why TanStack Router, Not TanStack Start

TanStack Start adds full-document SSR, streaming, server functions, API routes,
middleware, and client/server builds. Those are valuable when they are active
requirements, but they overlap the reason `web/` already exists.

TanStack Router provides the parts this surface needs:

- inferred route and navigation types;
- validated search parameters;
- nested layouts and error boundaries;
- intent preloading;
- generated file routes;
- automatic route code splitting;
- direct integration with TanStack Query.

Using Router directly preserves a single browser build and a static `dist/`
contract.

`pnpm architecture-check` keeps that decision executable: it rejects server
framework dependencies, direct feature-level Fetch calls, cross-deployable
source imports, route-layer data ownership, and browser persistence outside
the display-preference allowlist.

## Ownership

```text
src/routes/
  owns URL composition, route validation, layouts, and redirects

src/features/<feature>/
  owns user workflow components, hooks, services, state, DTO schemas, and tests

src/http/
  owns transport mechanics, bounded responses, shared envelope extraction,
  ClientErrorCode, backend error_code preservation, and request_id

src/app/
  owns application-wide provider and router composition

src/components/ui/
  owns the locally generated shadcn/ui primitives and variants

src/components/layout/
  owns the shared shadcn/ui Sidebar and console-header composition

src/config/
  owns validated browser-visible build configuration
```

Route files import feature pages. Features may import shared core modules.
Shared core modules never import a feature. No SPA source imports `web/` or
`api/` source.

## Data Flow

```text
route/component
  -> TanStack Query hook
  -> stateless feature service
  -> bounded native Fetch client
  -> same-origin gateway or Go API
```

TanStack Query is authoritative for remote cache state. Zustand is restricted
to shared browser UI state such as display preferences. Server records,
permissions, sessions, quotas, and remote resources do not become Zustand
persistence.

## UI Shell

The console uses the official shadcn/ui New York theme and Sidebar composition:

- `SidebarProvider` owns responsive desktop and mobile behavior;
- `Sidebar` owns navigation, grouped actions, collapse mode, and the mobile
  Sheet;
- `SidebarInset` owns the route header and page content surface;
- semantic theme variables in `src/styles/globals.css` are the visual contract;
- the brand, selection, and focus accent is Luas Rhine blue, while green is
  reserved for explicit success state;
- Zustand persists only the selected theme, locale, and desktop sidebar state.

The generated Sidebar is adapted for a pure static client. Its controlled open
state is persisted with the existing display preferences instead of writing
the framework example cookie on every toggle. Authentication and server state
remain outside browser persistence.

## Contract Parity

Architecture parity with `web/` means the same vocabulary and behavior:

- feature-first ownership;
- documented HTTP DTOs;
- `{ code, message, data }` success envelopes;
- dotted `error_code` branches;
- `request_id` diagnostics;
- explicit optional-starter activation;
- i18n, accessibility, tests, and performance budgets.

It does not mean copying Next.js server folders or pretending that the static
browser can own HttpOnly credentials. Each starter UI is added only when its
browser security and deployment adapter are implemented.

# Luas Architecture

Luas is a multi-deployable scaffold:

- `api/` is the Go backend. It owns persistence, domain rules, HTTP routes, migrations, seeders, and operational integrations.
- `web/` is the customer-facing Next.js application. It owns public and account
  journeys, user workspace UI, SSR, client state, mock BFF route handlers, and
  server-owned browser adapters.
- `admin/` is the project management application. It uses Vite and TanStack
  Router and builds to static OSS/CDN assets with no production Node.js runtime.

The units are independently deployable. A downstream application may deploy
`web/`, `admin/`, or both. All integration uses HTTP contracts, not shared source.

## Boundaries

- API and browser-shell code must not import each other's source.
- Both browser shells talk to API behavior over HTTP only.
- Contracts are documented under `contracts/`.
- The user-facing brand is `Luas`; lowercase `luas` is reserved for identifiers, packages, binaries, and env keys.

## API Shape

The API uses DDD-flavored starter modules under `api/internal/modules/`.

Default starter modules:

- `user` - account, login, profile, password flows.
- `apikey` - API key lifecycle and middleware registration.
- `audit` - write-request audit trail and user-facing audit history.

Optional starter modules are compiled but activated additively. `organization` owns tenancy and
membership, `permission` adds organization-scoped exact grants, and `notification` owns user-scoped
records, preferences, read state, and a durable channel delivery ledger. `asset` owns private
user-upload metadata, inspection, lifecycle, and deletion while opaque bytes remain behind the
storage capability. `setting` depends on `organization` and owns a finite code-defined catalog of
typed app, organization, and user overrides with versioned reset history; it is not a dynamic
configuration, secret, permission, entitlement, or notification-preference store. `usage` also
depends on `organization` and owns finite user/organization metrics, retained event idempotency, UTC
counters, and atomic hard-quota decisions; it is not telemetry, billing, or a public event API.

Typical flow:

```text
Handler -> Service -> Repository -> Database
DTO -> Domain -> PO
```

Infrastructure capabilities live under `api/internal/infra/` and `api/internal/capabilities/`.
For the starter readiness matrix and next reusable business starters, see
[`STARTER_BUSINESS_ROADMAP.md`](STARTER_BUSINESS_ROADMAP.md).

### Cross-Starter Transactions

Most starter operations own their transaction inside one repository. When an invariant spans two
active starters on the same database, the transaction owner may bind its GORM transaction to a
callback-only context through `api/internal/infra/database/transaction_context.go`. Cooperating
repositories resolve that context before falling back to their configured database.

- The transaction context must not escape its callback or be stored for later work.
- The owner controls commit and rollback; participants return errors and never commit independently.
- Every participating repository must define one lock order. Account deletion and organization
  membership creation use `user -> membership`, preventing a soft-deleted user from gaining a
  concurrent membership.
- Do not use this seam across deployable services or background jobs. Those require an explicit
  event, outbox, or workflow contract instead of an in-process database transaction.

### Active Organization Boundary

When the optional `organization` starter is enabled, tenant-scoped routes use the named
`organization_context` middleware after `auth`. The client sends one `Organization-Id` selection,
but the middleware treats that value only as a lookup key: it resolves the authenticated user's
current membership and organization in one indexed query before the handler runs.

```go
r.Group("", func(scoped *router.Router) {
	scoped.WithMiddleware("auth", "organization_context")
	scoped.GET("/projects", handler.List)
})
```

Handlers and services read the verified value with
`domain.OrganizationContextFromContext(ctx)`. They must not parse the transport header again,
persist a process-wide current organization, or trust an organization ID copied from a request
body. Every request therefore observes current membership and role state, including removal,
leave, and ownership transfer. See [`../contracts/ORGANIZATIONS.md`](../contracts/ORGANIZATIONS.md)
for the public header, cache, CORS, and non-disclosure contract.

## Customer Web Shape

The web app uses Next.js App Router and feature-first folders under `web/src/features/`.

Typical flow:

```text
Page or component -> Feature hook -> Feature service -> src/http/request.ts -> HTTP API
```

Mock BFF route handlers under `web/src/app/api/` are part of the scaffold so the Next.js shell can
run without a backend during development. They are replaceable development flows, not the
production API; see `web/docs/MOCK_BFF.md` for downstream replacement steps.

Production browser calls that use the Luas HttpOnly API session follow a fixed same-origin adapter
flow:

```text
Browser feature -> /api allowlist Route Handler -> bounded server-only API client -> Go /v1 route
```

`web/src/server/api-adapter/` owns credentials, trusted client-IP parsing, timeout, bounded upstream
JSON, safe response headers, and generic display copy. It accepts code-owned relative paths only;
do not replace the allowlist with a catch-all proxy. The optional organization feature is selected
with `NEXT_PUBLIC_OPTIONAL_FEATURES=organization`, derives the active organization from
`/console/organizations/:id`, and forwards `Organization-Id` to context-scoped operations. Explicit
member, invitation, ownership-transfer, and token-acceptance handlers map to fixed upstream paths;
the invitation token exists only in a same-origin POST body. The feature does not add selected
organization or invitation-secret state to Zustand, browser storage, cookies, or URLs.

The dependent `permission` starter is selected with `organization,permission` in the API and
Next.js Web shell. Its
access roles group exact code-owned dotted permission keys and attach to organization memberships.
Organization membership roles continue to own lifecycle and ownership. Effective permission checks
refresh current persistence, owner bypass is explicit, delegated mutations require dominance over
both current and requested grants, and resource-instance ownership remains in product policies.

The independent `notification` starter is selected with `notification` in the API and Next.js Web
shell. Downstream
API modules publish immutable user events through `domain.NotificationPublisher`; no public publish
endpoint exists. The API commits notification and delivery records atomically, while independently
deployed `luas notification:work` processes email deliveries through bounded database leases and a
stable provider idempotency key. The Web feature owns strict same-origin adapters, development mock
parity, unread state, preferences, and a lazy notification center; it never receives provider
responses or recipient routing data.

The independent `asset` starter is selected with `asset` in the API and Next.js Web shell. API
routes own the private
metadata lifecycle and issue short-lived upload/download grants through a provider-neutral object
store. Uploads land on random staging keys and become downloadable only after authoritative metadata
and bounded content inspection promote bytes to an immutable final key. The Web feature owns strict
fixed management adapters, ephemeral transfer execution, bounded mock parity, and console state; it
never receives object keys, local paths, checksums, or durable provider URLs. Production requires an
explicit R2 adapter while local storage remains a rooted development implementation. See
[`../contracts/ASSETS.md`](../contracts/ASSETS.md).

The organization-dependent `setting` starter is selected with `organization,setting` in the API
and Next.js Web shell. The API owns the finite typed catalog, effective default/override resolution, monotonic
versions, compare-and-swap writes, public app caching, private scope isolation, audit minimization,
and account cleanup. The Web feature accepts only the five shipped definitions, uses fixed
same-origin routes, and exposes real user and organization preferences. Unknown definitions fail
closed instead of becoming an arbitrary settings editor. See
[`../contracts/SETTINGS.md`](../contracts/SETTINGS.md).

The organization-dependent `usage` starter is selected with `organization,usage` in the API and
Next.js Web shell.
Trusted application services use framework-free record/consume seams; only the finite current-period
summary is browser-readable. Exact retained idempotency, safe integers, UTC periods, row locking,
quota CAS, denied-decision persistence, retention, and account cleanup stay in the API. The Next.js
Web shell uses two fixed private adapters and strict catalog validation; it cannot ingest events or
mutate quota.
See [`../contracts/USAGE.md`](../contracts/USAGE.md).

## Admin Console Shape

The static app uses Vite, TanStack Router, and feature-first folders under
`admin/src/features/`.

Typical flow:

```text
TanStack route -> Feature component -> Query hook -> Feature service
  -> src/http/client.ts -> HTTP API
```

Routes under `admin/src/routes/` are generated into one type-safe route tree and split
automatically. TanStack Query owns remote state, Zustand owns shared browser-only UI state, Zod
validates important responses, and i18next owns formal user-facing copy.

`admin/` emits only `dist/` static assets. It has no Route Handlers, Server Components, server
functions, private runtime environment, or mock BFF. Production routing serves existing hashed
assets, routes an allowlisted `/api/*` prefix to a reviewed backend when needed, and rewrites other
application paths to `index.html`.

Static delivery does not make bearer-token storage safe. The API provides an optional same-origin
Go browser-session adapter that owns an HttpOnly cookie, exact Origin enforcement, and fixed auth
operations; an equivalent reviewed gateway remains valid. Client route guards remain UX only, and
the browser session proves identity rather than system-operator authority. The initial static shell
includes system/readiness and browser preference features; protected starter UI is ported only after
its authentication and authorization contracts exist. See
[`../admin/docs/ARCHITECTURE.md`](../admin/docs/ARCHITECTURE.md) and
[`../admin/docs/SECURITY.md`](../admin/docs/SECURITY.md).

For the full list of scaffold surfaces and downstream keep/delete/replace rules, see
[`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md).

## Change Flow

For a vertical feature:

1. Update or add the HTTP contract under `contracts/`.
2. Implement the API module or endpoint in `api/`.
3. Add service, hook, UI, and tests in each browser shell the downstream app supports.
4. Run verification in each changed deployable unit, or `make check` at the repo root.

For cross-cutting changes, prefer small, explicit changes on each side over a shared source package.

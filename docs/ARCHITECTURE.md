# Luas Architecture

Luas is a two-part scaffold:

- `api/` is the Go backend. It owns persistence, domain rules, HTTP routes, migrations, seeders, and operational integrations.
- `web/` is the Next.js frontend. It owns UI, route groups, client state, mock BFF route handlers, and browser-facing workflows.

The two halves are independent deployable units. They share HTTP contracts, not source code.

## Boundaries

- API code must not import from `web/`.
- Web code talks to API behavior over HTTP only.
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
records, preferences, read state, and a durable channel delivery ledger.

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

## Web Shape

The web app uses Next.js App Router and feature-first folders under `web/src/features/`.

Typical flow:

```text
Page or component -> Feature hook -> Feature service -> src/http/request.ts -> HTTP API
```

Mock BFF route handlers under `web/src/app/api/` are part of the scaffold so the web half can run without a backend during development. They are replaceable development flows, not the production API; see `web/docs/MOCK_BFF.md` for downstream replacement steps.

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

The dependent `permission` starter is selected with `organization,permission` in both halves. Its
access roles group exact code-owned dotted permission keys and attach to organization memberships.
Organization membership roles continue to own lifecycle and ownership. Effective permission checks
refresh current persistence, owner bypass is explicit, delegated mutations require dominance over
both current and requested grants, and resource-instance ownership remains in product policies.

The independent `notification` starter is selected with `notification` in both halves. Downstream
API modules publish immutable user events through `domain.NotificationPublisher`; no public publish
endpoint exists. The API commits notification and delivery records atomically, while independently
deployed `luas notification:work` processes email deliveries through bounded database leases and a
stable provider idempotency key. The Web feature owns strict same-origin adapters, development mock
parity, unread state, preferences, and a lazy notification center; it never receives provider
responses or recipient routing data.

For the full list of scaffold surfaces and downstream keep/delete/replace rules, see
[`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md).

## Change Flow

For a vertical feature:

1. Update or add the HTTP contract under `contracts/`.
2. Implement the API module or endpoint in `api/`.
3. Add Web service, hook, UI, and tests in `web/`.
4. Run API and Web verification from the changed half, or `make check` at the repo root.

For cross-cutting changes, prefer small, explicit changes on each side over a shared source package.

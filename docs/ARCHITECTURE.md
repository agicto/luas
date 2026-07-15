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

## Web Shape

The web app uses Next.js App Router and feature-first folders under `web/src/features/`.

Typical flow:

```text
Page or component -> Feature hook -> Feature service -> src/http/request.ts -> HTTP API
```

Mock BFF route handlers under `web/src/app/api/` are part of the scaffold so the web half can run without a backend during development. They are replaceable development flows, not the production API; see `web/docs/MOCK_BFF.md` for downstream replacement steps.

For the full list of scaffold surfaces and downstream keep/delete/replace rules, see
[`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md).

## Change Flow

For a vertical feature:

1. Update or add the HTTP contract under `contracts/`.
2. Implement the API module or endpoint in `api/`.
3. Add Web service, hook, UI, and tests in `web/`.
4. Run API and Web verification from the changed half, or `make check` at the repo root.

For cross-cutting changes, prefer small, explicit changes on each side over a shared source package.

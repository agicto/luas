# ADR-0002: Default Starters

## Status

Accepted

## Context

Luas is a scaffold, so a new project needs useful business-ready starters on day one. At the same time, not every route-owning module belongs in the default app.

## Decision

The default scaffold ships with three starters:

- `user`
- `apikey`
- `audit`

`organization`; its dependent `permission`, `setting`, and `usage` starters; and the independent
`notification` and `asset` starters are runnable optional entries. Organization is enabled with
`OPTIONAL_STARTERS=organization`; dependent starters require the full dependency selection, for
example `OPTIONAL_STARTERS=organization,usage`; notification and asset can be enabled alone. They
are business-ready across their documented API, migrations, Web, mock, UI, test, governance, and
extraction surfaces while remaining absent from the default runtime.

## Consequences

- New projects get auth, machine access, and write-side audit logging without extra setup.
- Default starters should prefer self-service surfaces over admin/control-plane APIs.
- Organization-scoped RBAC is available when selected, but it does not increase default complexity for every app.
- User-scoped notification records and durable email delivery are available when selected without requiring tenancy.
- Private assets are available without tenancy; typed settings and trusted usage accounting remain
  explicit organization-dependent choices.
- Default starters cannot be subtracted through optional configuration.
- Routes, migrations, seeders, and runtime hooks are derived from default plus selected optional
  manifests through the starter registry.

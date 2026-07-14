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

The `permission` starter remains planned optional starter behavior, not part of the default scaffold.
It is not runnable until its module, migrations, contracts, Web feature, and tests exist.

The `organization` backend ownership kernel is the first runnable optional entry. It remains
disabled unless `OPTIONAL_STARTERS=organization` and is not yet business-ready across every
surface; invitation, membership, ownership-transfer, active-context, Web, and extraction work is
tracked in the starter roadmap and organization contract.

## Consequences

- New projects get auth, machine access, and write-side audit logging without extra setup.
- Default starters should prefer self-service surfaces over admin/control-plane APIs.
- RBAC stays a planned optional starter direction, but it does not increase default complexity for every app.
- Default starters cannot be subtracted through optional configuration.
- Routes, migrations, seeders, and runtime hooks are derived from default plus selected optional
  manifests through the starter registry.

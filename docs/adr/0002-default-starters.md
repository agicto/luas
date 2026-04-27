# ADR-0002: Default Starters

## Status

Accepted

## Context

ZGO is a scaffold, so a new project needs useful business-ready modules on day one. At the same time, not every module belongs in the default app.

## Decision

The default scaffold ships with three starters:

- `user`
- `apikey`
- `audit`

The `permission` module remains an `optional starter`, not part of the default scaffold.

## Consequences

- New projects get auth, machine access, and write-side audit logging without extra setup.
- Default starters should prefer self-service surfaces over admin/control-plane APIs.
- RBAC stays available, but it does not increase default complexity for every app.
- Default routes, migrations, and seeders should be derived from these starter decisions.

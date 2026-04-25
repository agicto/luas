# ADR-0002: Default Starters

## Status

Accepted

## Context

ZGO is a scaffold, so a new project needs useful business-ready modules on day one. At the same time, not every module belongs in the default app.

## Decision

The default scaffold ships with two starters:

- `user`
- `apikey`

The `permission` module remains an `optional starter`, not part of the default scaffold.

## Consequences

- New projects get auth and machine access without extra setup.
- RBAC stays available, but it does not increase default complexity for every app.
- Default routes, migrations, and seeders should be derived from these starter decisions.

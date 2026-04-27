# ADR-0003: Starter Registry

## Status

Accepted

## Context

The default starter list was spread across `internal/app`, `routes`, migration bootstrap, seed bootstrap, and Wire assembly. Adding or removing a starter required editing multiple seams, which reduced locality and made the scaffold harder to extend.

## Decision

Introduce a `starter registry` module as the single assembly point for:

- active starter modules
- default starter migrations
- default starter seeders

The application, route setup, migration bootstrap, and seed bootstrap should consume the registry instead of maintaining their own starter lists.

## Consequences

- Starter assembly moves behind one seam.
- Changing the default scaffold no longer requires edits across unrelated files.
- Future work can deepen this seam further by supporting optional starter selection without editing default assembly code.

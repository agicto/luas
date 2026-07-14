# ADR-0003: Starter Registry

## Status

Accepted

## Context

The default starter list was spread across `internal/app`, `routes`, migration bootstrap, seed bootstrap, and Wire assembly. Adding or removing a starter required editing multiple seams, which reduced locality and made the scaffold harder to extend.

## Decision

Introduce a `starter registry` module as the single assembly point for:

- active starter modules
- active starter migrations
- active starter seeders
- selected-starter runtime hooks

The application, route setup, migration bootstrap, and seed bootstrap should consume the registry instead of maintaining their own starter lists.

Starter assembly interfaces live under `internal/starter/assembly`, not a top-level starter
contract package. The term `contract` remains reserved for documented HTTP behavior under the
repository-level `contracts/` directory.

The `audit`, `apikey`, and `user` starters are immutable defaults. Optional starters are compiled
into the application provider graph but remain inactive unless their canonical lowercase name is
listed in `OPTIONAL_STARTERS`. Selection is additive: an optional list cannot remove a default,
and listing a default, unknown name, duplicate name, or non-canonical name fails startup.

HTTP assembly, the application migrator, standalone migration commands, and seeder commands all
resolve the same typed `Config.Starters.Optional` snapshot. Offline migration/seeder resolution
uses manifests without runtime Handler instances; typed-nil modules must therefore be omitted by
the assembly layer. Every replica and pre-deploy job must use an identical selection.

The first available optional entry is `organization`. Its activation hook installs account
ownership protection only when selected, proving that an inactive optional starter contributes no
routes, migrations, seeders, middleware, events, or runtime policy.

## Consequences

- Starter assembly moves behind one seam.
- Changing the default scaffold no longer requires edits across unrelated files.
- Adding an optional starter requires one provider contribution and one catalog manifest, not edits
  to global route or migration lists.
- Disabled optional starter code remains compiled into the binary so runtime activation stays a
  deployment configuration operation; binary-size impact must be measured when adding one.
- `OPTIONAL_STARTERS` is restart-scoped configuration, not a runtime feature flag.

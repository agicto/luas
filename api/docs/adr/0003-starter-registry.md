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
and listing a default, unknown name, duplicate name, non-canonical name, missing dependency, or
cyclic dependency fails startup. Selected optional manifests are assembled in dependency order.

HTTP assembly, the application migrator, standalone migration commands, and seeder commands all
resolve the same typed `Config.Starters.Optional` snapshot. Offline migration/seeder resolution
uses manifests without runtime Handler instances; typed-nil modules must therefore be omitted by
the assembly layer. Every replica and pre-deploy job must use an identical selection.

Available optional entries are `organization`; `permission`, `setting`, `usage`, and `webhook`,
which depend on `organization`; and `notification` and `asset`, which depend only on the default
`user` and `audit` starters and may be selected alone. Organization's activation hook installs
account ownership protection only when selected. Together they prove that inactive optional
starters contribute no routes, migrations, seeders, middleware, events, or runtime policy and that
partial dependency selection fails before infrastructure work.

Update (2026-09-30):

- Starter names have one source: the `config.Starter*` constants in
  `internal/infra/config/starters.go`. Manifests, module names, selection-dependent validation, and
  starter-owned commands reference them and call `Config.Starters.Selected(name)`; they do not
  compare string literals. A test keeps the constant set equal to the available catalog.
- Runtime handlers reach assembly through one `starter.Handlers` struct injected by
  `wire.Struct`, not positional parameters. Metadata-only callers (validation, catalog listing,
  offline migration and seeder resolution) pass no handlers at all.

## Consequences

- Starter assembly moves behind one seam.
- Changing the default scaffold no longer requires edits across unrelated files.
- Adding an optional starter requires one provider contribution and one catalog manifest, not edits
  to global route or migration lists.
- Disabled optional starter code remains compiled into the binary so runtime activation stays a
  deployment configuration operation; binary-size impact must be measured when adding one.
- `OPTIONAL_STARTERS` is restart-scoped configuration, not a runtime feature flag.

# API Assembly Hardening Plan

Reduced implementation plan (see
[BUSINESS_IMPLEMENTATION_PLAN_STANDARD.md](BUSINESS_IMPLEMENTATION_PLAN_STANDARD.md) §1)
for hardening how the Go API assembles starters, owns schema history, and
guides coding agents. It changes no public HTTP contract, adds no business
state, and touches only `api/` plus governance docs.

## 1. Summary

**Problem.** Adding a starter module touches several global lists (provider
set, positional handler parameters, `Application` fields, config validation,
generated `wire_gen.go`). A review against Kratos, go-zero, uber-go/fx,
Goravel, GoFrame, Encore, and Grafana concluded the compile-time Wire +
manifest + `Catalog.Select` design is sound and should be kept. Grafana runs
Wire at far larger scale, also constructs disabled services, and uses 50+
positional parameters in one provider. The review did find four higher-value
problems than Wire graph size, listed below.

**Decision.** No DI or architecture rewrite. Fix correctness and agent-safety
issues first, then reduce assembly touch points incrementally.

**Non-goals.** Migrating to fx or a runtime container, lazy or build-tag
module construction, goctl-style whole-project generation, moving modules out
of the single binary, changing `OPTIONAL_STARTERS` semantics.

**Success signal.** Fresh databases produce a schema that is pinned by a
golden test and independent of current persistence structs. Generated Wire code
cannot drift silently. Starter names have one source. Static starter metadata
no longer requires constructing or nil-passing handlers.

## 2. Evidence Baseline (verified 2026-09-28)

| Finding | Evidence |
|---|---|
| Versioned migrations call `AutoMigrate` on live module POs | 16 files under `api/database/migrations/` import `internal/modules/*`, e.g. `2026_04_27_000002_add_business_fields_to_audit_logs.go` runs `AutoMigrate(&audit.AuditLogPO{})`. Releases up to `v0.9.0` are tagged. |
| Wire upstream is archived | `github.com/google/wire` archived 2025-08-25 ("no longer maintained"); repo pins `v0.7.0` and runs it via `go tool wire`. |
| No generated-code drift check | `.github/workflows/ci.yml` never runs Wire; only `make build` regenerates. |
| Starter selection is string-matched in 16 production sites | `slices.Contains(cfg.Starters.Optional, "<name>")` in `config.go`, five module services/constructors, two console commands, four operator commands. |
| Metadata paths pass `nil ×10` handlers | `ValidateConfig`, `ConfiguredMigrations`, `ConfiguredSeeders`, `AvailableCatalog` in `api/internal/starter/defaults.go`. |
| Unused, misleading packages | `internal/infra/{container,lifecycle,pipeline,breaker}` have zero importers outside themselves; `internal/domain/events.go` exposes an unused global dispatcher (`Subscribe`/`Dispatch`/`DispatchAsync`). `container` is a service locator that bypasses Wire. |
| Stale ADR | `api/docs/adr/0003-starter-registry.md` lists three optional starters; seven exist. |
| Cost is not the bottleneck | Warm `make wire` 1.3 s, warm `go build ./...` 5.5 s; the webhook starter commit spent ~130 of ~7,400 lines on assembly. |

## 2a. Status (2026-09-30)

| Slice | State |
|---|---|
| P0-A step 1 (golden schema + reset round-trip tests) | Done. The golden file was generated from the unchanged tree and is identical on PostgreSQL 15, 16, and 17. CI covers 18. |
| P0-A steps 2–4 (frozen migrations) | Done. All 17 schema migrations are frozen SQL executed through `execStatements`. The SQL was generated from a `pg_dump` of the golden schema, and three hand-written migrations stay idempotent for older installations (username index, admin seed, audit business columns). The golden file is unchanged, byte for byte, on PostgreSQL 15, 16, and 17. `TestMigrationsDoNotDependOnLiveModelPackages` forbids module and capability imports, and the eight boundary scripts now pin table-level SQL instead of PO names. |
| P0-B (remove dead packages) | Done. Deleted `internal/infra/{container,lifecycle,pipeline,breaker}` and the unused parallel event system in `internal/domain/{aggregate,events}.go`. `api/AGENTS.md` forbids service locators and global dispatchers. |
| P1-A | Done: `make wire-check`, CI step, ADR-0014, `.gitattributes`. |
| P1-B | Done: `config.Starter*` constants, `StarterConfig.Selected`, 16 selection sites, manifest/module names, vocabulary sync test, governance scripts updated. |
| P1-C | Done in reduced form: one `starter.Handlers` struct (`wire.Struct`) replaced the 10 positional handler parameters and every `nil ×10` call. The module-level `NewStarterManifest(handler)` stayed because governance scripts pin its shape; splitting it into a separate spec type no longer removes any global touch point. |

New finding: `notification_preferences.user_id` carries an unused `bigserial` sequence default
(a GORM artifact on a foreign-key primary key). It is frozen as-is in the golden schema. Fixing it
needs its own forward migration.

### Design Principles Applied

This plan follows the engineering-fundamentals stance of
[mattpocock/skills](https://github.com/mattpocock/skills):

- **Feedback loops before change:** each slice starts with an executable check (golden schema,
  reset round-trip, `wire-check`, vocabulary sync test) so agents fail fast instead of relying on
  review.
- **Deep modules at clean seams:** starter assembly exposes one small interface
  (`ConfiguredManifests(cfg, *Handlers)`) instead of positional wiring spread across functions.
- **Shared language:** starter names have one vocabulary source, as domain terms already do in
  `CONTEXT.md`.
- **Fighting architectural entropy:** misleading code paths (service locator, parallel event
  dispatcher) are named as hazards in agent guidance, and their removal is tracked explicitly.
- **Decisions stay with the owner:** irreversible history edits and deletions are separate,
  approvable steps.

## 3. Delivery Slices

Each slice is one PR, merged in order. Slices are independent enough to stop
after any of them.

### P0-A — Freeze Migration Schema History

**Owner:** `api/database/migrations`. **Risk:** highest; affects every
downstream database.

1. Add a PostgreSQL golden-schema test. Migrate a disposable database with all
   starters selected, dump a normalized schema (tables, columns, types,
   nullability, defaults, indexes, constraints from `information_schema` /
   `pg_catalog`), and compare it to
   `database/migrations/testdata/schema.golden.sql`. Generate the golden file
   from the current tree before any other change.
2. Replace each `AutoMigrate(&module.XPO{})` with a migration-local snapshot
   struct or explicit SQL that reproduces the golden schema byte-for-byte.
   Migration names stay unchanged, so recorded migration rows remain valid and
   existing databases need no action.
3. Add a guard test: `go list -deps ./database/migrations` must not contain
   `github.com/zgiai/luas/api/internal/modules/`.
4. Update `api/docs/DATABASE.md` and the `sql-migration-review` skill: versioned
   migrations never reference live POs; new tables use snapshot structs or SQL.

**Acceptance:** golden test passes before and after step 2 without editing the
golden file; guard test fails if a module import is reintroduced.
**Verification:** targeted `go test ./database/migrations/...` against
disposable PostgreSQL, then the API tier that includes migrations.
**Rollback:** revert the PR; schema output is identical by construction.

### P0-B — Remove Misleading Dead Code

**Owner:** `api/internal/infra`, `api/internal/domain`.

1. Delete `internal/infra/container`, `lifecycle`, `pipeline`, and `breaker`,
   plus the unused global dispatcher functions in `internal/domain/events.go`
   (keep types that have live callers).
2. Remove doc references to these packages.
3. Add to `api/AGENTS.md`: dependencies flow through Wire providers; do not add
   global service locators, registries, or dispatchers.

**Acceptance:** `go build ./...` and `go vet ./...` pass, and nothing refers to
the deleted packages. Release notes flag the removal for forks that may have
imported them.
**Verification:** `go build ./... && go test ./internal/...` (targeted), `make agent-check-changed`.

### P1-A — Wire Governance

**Owner:** `api/Makefile`, CI, ADR.

1. Add `make wire-check`: regenerate, then `git diff --exit-code internal/wiring`.
2. Run it in the API CI job; document it in `docs/CI.md`. Mark `wire_gen.go`
   as `linguist-generated` in `.gitattributes` and document "resolve conflicts by
   re-running `make wire`, never by hand".
3. Write ADR-0014 "Wire maintenance posture": stay on `v0.7.0`. Wire only
   generates plain Go; at runtime the binary depends on nothing beyond the tiny
   `wire` marker package. If a future Go release breaks the generator, switch
   to a maintained fork or vendor the generator. As a last resort, keep
   `wire_gen.go` as hand-maintained code.

**Acceptance:** a PR that edits a `ProviderSet` without regenerating fails CI.
**Verification:** run `make wire-check` locally on a clean tree and on a
deliberately stale tree.

### P1-B — Single Source for Starter Names

**Owner:** `api/internal/starter`, `api/internal/infra/config`.

1. Add a dependency-free leaf package (e.g. `internal/starter/names`) with one
   constant per starter. `config` cannot import modules, so the leaf package
   avoids import cycles.
2. Add `func (s StartersConfig) Selected(name string) bool`.
3. Replace all 16 string-matching sites and the manifest `Name`/dependency
   literals with the constants.
4. Test: the set of catalog manifest names equals the set of constants.

**Acceptance:** no string literal starter names remain in production code
outside the names package.
**Verification:** `go test ./internal/starter/... ./internal/infra/config/...`, affected module tests.

### P1-C — Split Static Starter Spec From Runtime Binding

**Owner:** `api/internal/starter`, `api/internal/starter/assembly`, each module's `provider.go`.

1. Introduce `assembly.StarterSpec` (name, dependencies, migration names, seeder
   names). Each module exports its spec and no longer needs a handler to
   describe itself.
2. Build `Catalog` and selection from specs only. `ValidateConfig`,
   `ConfiguredMigrations`, `ConfiguredSeeders`, `AvailableCatalog`, and the
   default variants stop taking handlers.
3. Runtime binding receives one `Handlers` struct injected with
   `wire.Struct(new(Handlers), "*")` and maps selected specs to modules.
4. Update ADR-0003 (including the stale optional starter list) and
   `api/docs/ADDING_MODULE.md`.

**Acceptance:** no function passes `nil` handlers to read metadata. Adding a
starter touches its module, one spec list entry, one `Handlers` field, and one
`ProviderSet` line. Route catalog and migration parity are unchanged with starters
disabled and enabled.
**Verification:** `go test ./internal/starter/...`, `make wire-check`,
`make route-catalog-check`, optional-starter assembly tests. Estimated 200–400
changed lines.

## 4. Trigger-Gated Work (Do Not Start Yet)

| Item | Trigger |
|---|---|
| `assembly.CommandModule` so modules own operator commands and workers; shrink `app.Application`'s 18 port fields | Operator commands exceed ~25, or a downstream fork reports conflicts in `operatorcommands/manifest.go` or `app.go` |
| Module-owned config sections and validation via `StarterSpec` | `config.go` grows past ~1,500 lines or a fourth starter needs selection-dependent validation |
| Downstream Wire extension point (Grafana `wireexts` pattern) | First downstream fork needs to add providers without editing `internal/wiring` |
| Module import boundary check against declared manifest dependencies | More than ~15 modules or the first undeclared cross-module import |
| Re-evaluate DI approach | Warm `make wire` exceeds 10 s, or Wire breaks on a supported Go release |

## 5. Open Questions

- Are there known downstream forks importing the packages removed in P0-B?
  If so, P0-B ships with a deprecation release first.
- Should P0-A be followed by a patch release so downstreams pick up frozen
  migrations before they add their own?

## 6. Follow-Up

After review, register P0-A through P1-C in
[FRAMEWORK_QUALITY_ROADMAP.md](FRAMEWORK_QUALITY_ROADMAP.md) via the
`luas-framework-review` skill, per that roadmap's ranking rule.

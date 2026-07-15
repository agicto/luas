# Typed Settings

The optional `setting` starter provides durable overrides for a finite code-owned catalog. Read the
cross-deployable contract in [`../../contracts/SETTINGS.md`](../../contracts/SETTINGS.md) and the
architecture decision in [`adr/0007-typed-setting-boundary.md`](adr/0007-typed-setting-boundary.md)
before extending it.

## Activation And Ownership

```dotenv
OPTIONAL_STARTERS=organization,setting
```

`setting` depends on `user`, `audit`, and `organization`. Selection activates routes, migration
`2026_07_15_040000_create_settings_table`, and the account-deletion cleaner from the same startup
snapshot. API replicas, migration jobs, and operator commands must use the same selection.

The module lives at `internal/modules/setting`. Its layers are:

- `catalog.go`: finite definitions and scalar normalization;
- `model.go` / `repository.go`: scoped owner constraints, row locks, and compare-and-swap history;
- `service.go`: defaults, override decoding, public filtering, audit metadata, and downstream seams;
- `handler.go` / `routes.go`: cache, precondition, auth, and organization-role behavior;
- `provider.go`: optional starter assembly and migration ownership.

Business-aware operator commands live under `internal/bootstrap/operatorcommands`, where assembly
code may combine the setting and audit seams. Generic command registration/output remains under
`internal/infra/console`; infrastructure does not import setting or domain packages.

The app container exposes `domain.SettingReader` and `domain.AppSettingWriter`. Downstream modules use
those seams; they do not import the setting repository or decode `value_json` themselves.

## Extending The Catalog

Replace or compose the `NewDefaultCatalog` provider at assembly time using the exported definition
constructors. A definition must use one stable lowercase semantic key and one scalar kind: string,
boolean, integer, enum, or timezone. Keys are namespaced, dot-separated, and use letter-led
snake_case within each segment. Public visibility is valid only for app scope. Defaults are
validated during startup; duplicates, invalid defaults, invalid enum options, and catalogs above 64
definitions fail startup.

Browser-facing additions require a matching contract and strict Web schema/UI change. A database row
does not make an unknown key part of the contract. Renaming/removing a definition is a data migration
decision because old rows remain intentionally invisible until migrated or removed.

Do not put passwords, API tokens, provider credentials, private keys, or connection strings in the
catalog. Setting audit/log minimization is defense in depth, not secret-storage authorization.

## Concurrency And Storage

The unique identity is `(scope, subject_id, key)`. User and organization writes lock the owner row,
then lock or create the setting row and compare its version. App writes use the same row-level CAS.
Exactly one writer succeeds for an expected version.

Reset writes a tombstone (`is_overridden=false`) and increments version. It does not delete the row,
so a stale writer cannot become current after reset. No-op set/reset operations do not increment.
Stored JSON must decode to the definition's canonical scalar representation; corrupt values fail
with service unavailable rather than silently reverting to defaults.

All definitions are queried in one bounded `IN` request. Collections intentionally do not paginate
because the catalog limit is lower and more meaningful than a client-controlled page size.

## Account Deletion

The user module runs every `AccountDeletionGuard` before any `AccountDeletionCleaner`. Cleaners then
run in deterministic order inside the locked user transaction before soft delete. The setting
cleaner physically removes user-scope rows. A guard failure runs no cleaner; any cleaner failure
rolls back earlier cleanup and account deletion.

Organization rows remain until Luas defines organization deletion. Their foreign key uses physical
cascade as a future last line of integrity; no current route physically deletes an organization.

## Audit And Operations

Setting mutation audit metadata contains operation, scope, key, subject ID, before/after version,
and source. It never includes request values, stored JSON, defaults, or enum options.

App-scope operations are intentionally CLI/internal-only:

```bash
go run ./cmd/luas setting:list
go run ./cmd/luas setting:set \
  --key=branding.display_name \
  --value='"Luas Cloud"' \
  --expected-version=0
go run ./cmd/luas setting:reset \
  --key=branding.display_name \
  --expected-version=1
```

Commands require `setting` selection, an available database, and explicit expected versions. Output
contains metadata rather than values. Set/reset persist a system-actor audit entry with only scope,
key, versions, source, and operation. Audit persistence remains best-effort after the setting commit,
matching HTTP audit behavior; a failure produces an operator warning rather than pretending the
setting write rolled back. Secrets and startup configuration continue through the typed configuration
authority documented in [`CONFIGURATION.md`](CONFIGURATION.md).

## Verification

```bash
go test ./internal/modules/setting ./internal/modules/user
go test ./internal/infra/console/commands ./internal/starter ./internal/bootstrap
go run ./cmd/luas route:list
python3 ../.agents/skills/luas-framework-review/scripts/check-setting-boundary.py
```

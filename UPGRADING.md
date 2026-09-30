# Upgrading Luas

This guide lists changes that a downstream fork must review when it pulls a newer Luas. Each entry
has an impact level:

- **High**: the fork fails to build, or behavior changes, until it acts.
- **Medium**: builds and runs, but a documented rule changed; code that follows the old rule should
  be updated.
- **Low**: cleanup that only matters if the fork used the affected code.

Newest changes come first. Merge the upstream branch, run `make check`, then work through the
entries in order.

## 2026-10-01 — Platform operator starter

### Low — New optional `operator` starter

Opt-in only; nothing changes unless `operator` is added to `OPTIONAL_STARTERS`.

- Selecting it requires `OPERATOR_ALLOWED_ORIGINS` (exact Admin Console origins) and adds the
  `platform_operators` migration, `/v1/operator/*` routes, and `operator:grant`, `operator:revoke`,
  and `operator:list` commands. Grant the first operator with `luas operator:grant <email>`.
- To require operator sign-in in the Admin Console, build it with `VITE_OPTIONAL_FEATURES=operator`
  and route the Admin origin's `/api/*` to the Go API on the same origin. Without the flag the
  console is unchanged.
- New error codes: `OPERATOR.FORBIDDEN`, `OPERATOR.ORIGIN_REJECTED`, `OPERATOR.CSRF_REJECTED`,
  `OPERATOR.TARGET_PROTECTED`. Clients that switch exhaustively on `error_code` should add them.
- `domain.CredentialSignIn` and `domain.SessionRevoker` are new user-starter seams; the public login
  now shares its credential check with operator sign-in without behavior change.
- `domain.UserAdministrator` is a new user-starter seam used by `/v1/operator/users`. Disabling an
  account through it revokes the account's sessions in the same transaction (reason
  `account_disabled`); ending sessions uses the new revocation reason `operator`.
- `domain.AuditLogRepository` gains `FindAll`, and `domain.AuditLogFilter` gains `UserID`, `From`, and
  `To`. Custom audit repositories must implement `FindAll` (newest first, `From` inclusive, `To`
  exclusive).

## 2026-09-30 — Dead code and dialect cleanup

### High — Unused API packages removed

Removed with no production callers: `internal/infra/{queue,schedule,retry,http,lang,types,contracts}`,
`internal/infra/events/dispatcher.go` (global `SimpleDispatcher`), `pkg/{hash,utils,request,
resource,encryption,validation,support,events}`, and the `api/tests/unit` directory.

- Queue and scheduler code: import `internal/capabilities/workflow` directly. `queue.New()` is
  `workflow.NewQueueManager()`, `schedule.New()` is `workflow.NewScheduler()`, and
  `schedule.Global()` is `workflow.GlobalScheduler()`; other names are unchanged.
- `pkg/encryption`: use `internal/capabilities/crypto`.
- `pkg/events.Event`: use `events.BasicEvent` from `internal/infra/events`. The embedded field of
  `events.WrappedEvent` is now `BasicEvent`.
- Tests formerly in `tests/unit` now live beside their packages as `*_external_test.go`; CI targets
  (`test-race-critical`, `benchmark-workflow`) point at `./internal/capabilities/workflow`.
- `make governance` now rejects `support`, `utils`, `common`, and `helpers` packages under `pkg/`
  and `internal/infra/`.

### Medium — Schema builder is PostgreSQL-only

The MySQL grammar and the MySQL-only `ColumnDefinition.After` and `First` methods were removed.
`schema.NewGrammar` always returns the PostgreSQL grammar.

### Low — Deprecated registries and unused Web utilities removed

- `migrations.Default()`, `seeders.Default()`, and `seeders.RunDefault()` are gone; use
  `starter.DefaultMigrations()`, `starter.DefaultSeeders()`, or the configured variants.
- Web `src/utils` keeps only `cn`. The unused `string`, `date`, `object`, `array`, `validation`, and
  `event-bus` utilities and the `useEventBus` hook were removed.

### Scheduled — `ROLE.NOT_FOUND` error code

`ROLE.NOT_FOUND` is a deprecated legacy value in the public error-code enum and is no longer emitted
by any starter; access roles use `PERMISSION.ROLE_NOT_FOUND`. It will be removed from
`contracts/openapi.yaml` in a later release. Stop matching on it now.

## 2026-09-30 — API assembly hardening

### High — Released migrations frozen as SQL

Versioned migrations no longer call `AutoMigrate` on module persistence structs. Existing databases
need no action: migration names are unchanged and the schema is byte-identical.

- Write new migrations as SQL through `execStatements` or with the schema builder; never import
  `internal/modules` or `internal/capabilities` from `database/migrations`.
- `TestMigrationsProduceGoldenSchema` compares a fresh database with
  `database/migrations/testdata/schema.golden.sql`. When you add a migration, regenerate it with
  `LUAS_UPDATE_GOLDEN_SCHEMA=1` and review the diff.

### High — Starter handlers injected as one struct

`starter.NewConfiguredRegistry`, `ConfiguredManifests`, `DefaultManifests`, and `OptionalManifests`
take a `*starter.Handlers` instead of positional handler parameters. Metadata-only callers pass
`nil`. A fork that added a starter adds one `Handlers` field and one manifest line, then runs
`make wire`.

### Medium — Starter names come from constants

Use `config.Starter*` constants and `cfg.Starters.Selected(name)` instead of string literals.
Add a constant and a `StarterNames()` entry for each new starter; a catalog test enforces the match.

### Medium — Removed domain event system and service locator

`internal/domain/{aggregate,events}.go` and `internal/infra/{container,lifecycle,pipeline,breaker}`
were removed. Publish events through the injected `events.EventBus`.

### Low — Wire drift check

CI runs `make wire-check`. Resolve `wire_gen.go` merge conflicts by running `make wire`, never by
hand. See `api/docs/adr/0014-wire-maintenance-posture.md`.

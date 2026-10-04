# Upgrading Luas

This guide lists changes that a downstream fork must review when it pulls a newer Luas. Each entry
has an impact level:

- **High**: the fork fails to build, or behavior changes, until it acts.
- **Medium**: builds and runs, but a documented rule changed; code that follows the old rule should
  be updated.
- **Low**: cleanup that only matters if the fork used the affected code.

Newest changes come first. Merge the upstream branch, run `make check`, then work through the
entries in order.

## 2026-10-03 — Security review fixes

### Medium — User writes are column-scoped

`user` repository `Update` now writes only `nickname`, `avatar`, `phone`, and `bio` and returns
`USER.NOT_FOUND` for a deleted account; sign-in records `last_login` through the new `RecordLogin`.
Before, both wrote the whole row they had read, so a sign-in or profile update racing a password
change, operator disable, or account deletion could restore the old password hash, re-enable the
account, or undelete it. A fork that relied on `Update` to persist other columns must add a
dedicated repository command for them.

### Medium — Usernames cannot contain `@`, and email sign-in resolves to the email owner

Registration rejects a username that contains `@` (`422`). A login identifier that contains `@`
matches the email first. Before, the documented username precedence was not applied at all (the
lookup returned the lowest ID), so an account registered with another account's email as its
username could capture that account's sign-in. Existing usernames are not changed.

### Low — Operator routes reject foreign origins on reads

Every `/v1/operator` request that carries an `Origin` outside `OPERATOR_ALLOWED_ORIGINS` now fails
with `403 OPERATOR.ORIGIN_REJECTED`, not only unsafe methods. Same-origin reads, which send no
`Origin`, are unaffected. The operator cookie is `Secure` whenever every allowed origin is `https`.

### Low — Webhook targets in IPv6 transition ranges are rejected

IPv4-compatible (`::/96`), NAT64 (`64:ff9b::/96`, `64:ff9b:1::/48`), Teredo (`2001::/32`), and 6to4
(`2002::/16`) destinations fail with `WEBHOOK.INVALID_TARGET` unless private targets are explicitly
allowed.

## 2026-10-03 — Dependency security updates

### Medium — Toolchain and framework minimums raised

All open advisories are closed. A fork must pick up the same minimums:

- Go `1.25.13` (seven standard-library advisories in `1.25.12`); the API builder image is pinned to
  the matching digest in `api/Dockerfile`, `api/scripts/verify-container.sh`, and the container
  governance check.
- Next.js `16.3.8` and `eslint-config-next` `16.3.8` (three remote-code-execution advisories in
  `16.2.x`), axios `1.20.0`, vitest `4.1.11`.
- `google.golang.org/grpc` `1.83.2`, `golang.org/x/crypto` `0.55.0`, and OpenTelemetry `1.45.0`.
- The Web base image is Node `22.23.3` on Alpine 3.24 (OpenSSL fix), pinned by digest in
  `web/Dockerfile`, `web/scripts/verify-container.sh`, and the container governance check.
- pnpm overrides in all three `pnpm-workspace.yaml` files now force patched `sharp`, `undici`,
  `js-yaml`, `browserslist`, `baseline-browser-mapping`, `brace-expansion`, and `@humanfs/node`.

axios `1.20` changed its method return types; `web/src/http/request.ts` narrows the unwrapped
payload in one helper. A fork that calls `axios` instances with two explicit type arguments
(`get<T, T>`) should do the same.

## 2026-10-02 — Admin Console starter administration

### Low — Operator routes and screens for organizations, webhooks, and notifications

When `operator` is selected together with `organization`, `webhook`, or `notification`, the API now
serves read-mostly operator routes for that starter under `/v1/operator`
([`contracts/OPERATORS.md`](contracts/OPERATORS.md)): an organization directory with members,
secret-free webhook endpoints, deliveries, and attempts with delivery replay, and the notification
delivery ledger without content. A deployment that selects those starters but must not expose them
to operators has to remove the mounts in `api/internal/modules/operator/routes.go`.

The Admin Console shows the matching screens only when `VITE_OPTIONAL_FEATURES` lists
`organization`, `webhook`, or `notification` in addition to `operator`. No migration.

### Low — `operator.NewHandler` takes a `Surfaces` struct

`operator.NewHandler(service, guard, cfg, settings)` became
`operator.NewHandler(service, guard, cfg, operator.Surfaces{...})`. Wire injects `Surfaces`; only a
fork that constructs the handler by hand must change the call.

## 2026-10-02 — Schema cleanup

### Low — `notification_preferences.user_id` loses its sequence default

Migration `2026_10_02_000000_drop_notification_preferences_user_id_sequence` drops the
`notification_preferences_user_id_seq` default left by the original `bigserial` definition. The
column is always the owning user's ID, so no application write used the sequence. Run migrations
with the `notification` starter selected; a fork that inserted preference rows without `user_id`
must now supply it.

## 2026-10-01 — Operations hardening

### Medium — CLI plugins no longer load from the current directory

`plugin:list` and plugin dispatch search only absolute `PATH` entries, and plugin names must match
`[a-z][a-z0-9-]*`. A plugin that lived only in the working directory must move onto `PATH`.

### Low — Durable task operations

New commands `workflow:tasks`, `workflow:retry`, `workflow:cancel`, and `workflow:prune` require
`QUEUE_DRIVER=postgres`. Finished tasks were never deleted before; schedule `workflow:prune`
(default retention 720h) to bound the `workflow_tasks` table.

## 2026-10-01 — Module ownership and generator

### Medium — Starters own their error mappings

`internal/bootstrap/domain_error_mappings.go` now maps only shared errors (not found, conflict,
invalid input, unavailable, authentication required, permission denied, legacy role). Each starter
maps its own errors in `error_mappings.go` through the new `assembly.ErrorModule` capability. A fork
that added mappings to the central file should move them into its starter's
`RegisterErrorMappings`; `registerDomainErrorMappings` now also takes the starter registry.

### Low — `make:module` generates a wired optional starter

The generator adds a starter manifest, error-mapping hook, `auth`-protected routes, and a frozen SQL
migration, and registers the starter constant and catalog entry. Generated packages drop underscores
(`blog_post` directory, `blogpost` package).

### Low — Test doubles and removed middleware

- `internal/infra/testing.FakeMailer` records email for any starter mail seam.
- The unused `internal/infra/middleware` CORS implementation was removed; the kernel uses
  `gin-contrib/cors`.

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
- The setting starter's `Handler` exposes `AppList`, `AppSet`, and `AppReset` for app-scoped settings.
  They perform no authorization; mount them only behind an authorizing starter such as `operator`.

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

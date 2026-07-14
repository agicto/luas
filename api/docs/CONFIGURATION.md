# Configuration

Luas uses one typed configuration authority: [`internal/infra/config.Config`](../internal/infra/config/config.go).
Environment files are inputs to that startup snapshot; they are not a second configuration API.

## Authority And Lifecycle

1. A process calls `config.Load()` during bootstrap.
2. `pkg/env` resolves process variables and optional environment files.
3. `config.Load()` converts those strings into typed fields and validates cross-field invariants.
4. Bootstrap gives that same `*config.Config` pointer to logging and the Wire dependency graph.
5. The snapshot remains fixed for the lifetime of that process.

New runtime code should receive `*config.Config` or a smaller capability-owned typed config through
dependency injection. `config.GlobalConfig` remains a read-only compatibility surface for existing
middleware and the welcome route; do not make new features depend on it.

`config.Use()` exists for tests and alternate bootstraps that already hold a fully constructed
configuration. `config.LoadFresh()` exists for tests and one-shot diagnostics. Neither function
rebuilds already constructed services.

`config.LoadAIConfig()` is the capability-scoped loader for `ai:chat`; it returns the same typed
`AIConfig` used by the full snapshot without requiring unrelated database or JWT settings. Runtime
packages still do not read environment variables directly.

## Precedence

From highest to lowest:

1. Process environment variables supplied by the shell, container runtime, or orchestrator.
2. The file explicitly selected by process variable `LUAS_ENV_FILE`.
3. `.env.<APP_ENV>.local`.
4. `.env.local`.
5. `.env.<APP_ENV>`.
6. `.env`.
7. Defaults in `config.Load()`.

An explicitly present process value, including an empty string, wins over every file value.
`APP_ENV` file selection is resolved from the process environment first, then `LUAS_ENV_FILE`, then
the base `.env`. `GO_ENV` and `GIN_MODE` are accepted as fallback inputs and normalized to the same
application environment. Environment-specific and local files cannot change the identity after file
selection, so the loaded file set and production defaults cannot disagree.

`LoadFresh()` removes values injected by the previous file snapshot before resolving the layers
again. If application or test code changed one of those values at runtime, the changed process value
is preserved as the new highest-priority input. Missing optional files are normal, while malformed
files and a missing explicitly selected `LUAS_ENV_FILE` fail typed configuration loading.

`production`, `prod`, and `release` are equivalent production aliases. Without explicit overrides,
they disable application debug output and local file logging, select Gin release mode, emit JSON logs
to stdout, enable the scaffold rate-limit guardrails, and keep metrics disabled.

## Restart, Not Hot Reload

Configuration changes require a process restart. Watching `.env` and mutating a global key/value map
would leave database pools, HTTP servers, clients, middleware, and workers on different snapshots.
Luas therefore does not expose runtime configuration watching. `make air` rebuilds and restarts the
development process, which creates a complete new dependency graph.

## Optional Starter Activation

`OPTIONAL_STARTERS` is a comma-separated, additive list of canonical starter names. The default is
empty; `audit`, `apikey`, and `user` remain active without being named. The current available value
is `organization`:

```dotenv
OPTIONAL_STARTERS=organization
```

Selection is resolved through `internal/starter` from the same typed configuration snapshot used by
Wire. Unknown names, duplicates, non-lowercase names, and default starter names fail before server
startup or CLI database work. HTTP routes, runtime hooks, migrations, and seeders consume the same
selection; do not maintain a second starter list in a command or deployment script.

All API replicas, migration jobs, and seeder jobs for one environment must receive the same value.
Changing it requires a deployment and, when enabling a persistence-owning starter, its pre-deploy
migration. It is not a per-request flag and must not be toggled independently across replicas.

## Email Provider Configuration

The optional email capability reads `MAIL_FROM`, `RESEND_API_KEY`, and `MAIL_REQUEST_TIMEOUT` from
the same typed snapshot. Sender and API key are an all-or-none pair; partial configuration fails
startup instead of silently skipping delivery. When configured, the sender must be a valid mailbox
and the timeout must be positive. The default provider request timeout is 10 seconds.

The timeout is a duration value such as `3s` or `500ms`, not a bare millisecond count. It bounds each
provider call independently and composes with caller cancellation. See [`EMAIL.md`](EMAIL.md) for the
64 KiB response cap, privacy rules, best-effort semantics, and downstream adapter boundary.

## Secrets And Deployment

Luas does not serialize a configuration cache. A cache would duplicate `JWT_SECRET`, database
credentials, provider keys, and other secrets on disk while bypassing the deployment platform's
secret lifecycle. Production images also do not contain `.env` or `.env.example`; inject required
values through the runtime or a mounted secret source.

Keep `.env.example` as the schema of available keys and safe local examples. It is not a claim that
every deployment must set every key. Required and conditional values belong in typed validation.

Local file logging is intentionally simple. Production should use structured stdout and let the
runtime platform own collection, retention, rotation, and compression; Luas does not advertise
in-process rotation or a storage sink that does not actually persist records.

## Adding A Key

1. Add the typed field to `Config` or the owning nested config.
2. Read it once in `config.Load()` with an explicit default.
3. Add validation for unsafe or contradictory combinations.
4. Document it in `.env.example` without a real secret.
5. Pass a smaller typed configuration into the owning capability when practical.
6. Add loading and validation tests, including production behavior when relevant.

Do not add dot-notation string accessors, parallel dynamic repositories, or package-specific calls to
`os.Getenv` for runtime settings.

## Doctor

Run the diagnostic from the API directory:

```bash
cd api && go run ./cmd/luas doctor
```

Use `--env-example=/path/to/.env.example` for a downstream schema. Doctor rejects malformed or
duplicate declarations, runs the same typed configuration validation as bootstrap, and reports
security-sensitive combinations. Provider model IDs are treated as provider-owned opaque values;
Luas only checks whether the selected provider has a built-in adapter and its required connection
settings.

## Verification

```bash
go test ./tests/unit -run '^TestEnv_'
go test ./internal/infra/config ./internal/infra/email ./internal/infra/console/commands
go test -race ./tests/unit ./internal/infra/config ./internal/infra/email ./internal/infra/console/commands
```

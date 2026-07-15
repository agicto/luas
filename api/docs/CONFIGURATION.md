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
packages still do not read environment variables directly. It also applies the same enablement,
endpoint, timeout, and byte-limit validation as the full application loader.

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
empty; `audit`, `apikey`, and `user` remain active without being named. Available values are
`organization`, `permission`, `notification`, `asset`, `setting`, `usage`, and `webhook`;
permission, setting, usage, and webhook explicitly depend on organization, while notification and
asset can be selected independently:

```dotenv
OPTIONAL_STARTERS=organization,permission
ORGANIZATION_INVITATION_TTL=168h
# or: OPTIONAL_STARTERS=notification
# or: OPTIONAL_STARTERS=asset
# or: OPTIONAL_STARTERS=organization,setting
# or: OPTIONAL_STARTERS=organization,usage
# or: OPTIONAL_STARTERS=organization,webhook
```

Selection is resolved through `internal/starter` from the same typed configuration snapshot used by
Wire. Unknown names, duplicates, non-lowercase names, default starter names, missing dependencies,
and dependency cycles fail before server startup or CLI database work. Dependency order is resolved
deterministically. HTTP routes, runtime hooks, migrations, and seeders consume the same selection;
do not maintain a second starter list in a command or deployment script.

All API replicas, migration jobs, and seeder jobs for one environment must receive the same value.
Changing it requires a deployment and, when enabling a persistence-owning starter, its pre-deploy
migration. It is not a per-request flag and must not be toggled independently across replicas.

`ORGANIZATION_INVITATION_TTL` controls the lifetime of one-time invitation tokens and defaults to
`168h` (7 days). It must be positive when `organization` is selected. Invitation expiry is evaluated
against this immutable startup policy; changing the value affects newly created invitations only and
requires a process restart like every other configuration change.

## Outbound Webhook Configuration

Selecting `webhook` requires `WEBHOOK_ENCRYPTION_KEY` with at least 32 characters. It protects
endpoint signing secrets at rest and must be supplied from the deployment secret store. Keep the
same key available to HTTP replicas, workers, replay/prune jobs, and migration-safe rollback windows;
changing it without an explicit data-key migration makes existing endpoints unreadable.

`WEBHOOK_REQUEST_TIMEOUT` bounds each receiver call and defaults to `15s` with a hard maximum of
`30s`. `WEBHOOK_MAX_RESPONSE_BYTES` bounds response draining and defaults to 65,536 bytes. Responses
are never persisted. `WEBHOOK_SECRET_OVERLAP` controls the previous-secret signing window and
defaults to `24h`; `WEBHOOK_EVENT_RETENTION` controls terminal replay history and defaults to `720h`
(30 days). Startup rejects values outside the documented safety bounds.

`WEBHOOK_ALLOW_INSECURE_HTTP` and `WEBHOOK_ALLOW_PRIVATE_TARGETS` exist only for local verification.
Production rejects either override. Local Compose enables them so its isolated verifier can dispatch
to the API container itself; downstream production deployments should also enforce outbound network
policy independently of application validation. See [`WEBHOOKS.md`](WEBHOOKS.md).

## AI Capability Configuration

`AI_ENABLED` defaults to `false`. Enabling it requires an explicit `AI_DEFAULT_MODEL` and the secret
for the selected registered provider; Luas intentionally does not choose a model whose behavior and
availability can change independently of the scaffold. The built-in provider requires
`OPENAI_API_KEY`. `OPENAI_BASE_URL` must be an absolute HTTP(S) URL without credentials, query, or
fragment, and production requires HTTPS.

`AI_REQUEST_TIMEOUT` defaults to `120s` and bounds both a one-shot request and the complete lifetime
of a streaming session. `AI_MAX_INPUT_BYTES`, `AI_MAX_RESPONSE_BYTES`, and
`AI_MAX_STREAM_EVENT_BYTES` default to 1 MiB, 4 MiB, and 1 MiB respectively. Typed startup validation
keeps each value inside its documented hard range and ensures the stream-event cap cannot exceed the
response cap. See [`AI.md`](AI.md) for transport, privacy, error, retry, and product-boundary rules.

## Object Storage And Asset Configuration

`OBJECT_STORAGE_DRIVER` selects the provider-neutral object adapter. It defaults to `disabled`,
except that selecting `asset` outside production defaults it to `local`. A production process with
the asset starter fails validation unless the driver is explicitly `r2`; container-local storage is
never a production fallback.

The local driver uses `OBJECT_STORAGE_LOCAL_ROOT` and is intended for private development data only.
The R2 driver requires the all-or-none secret group `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`,
`R2_ACCESS_KEY_SECRET`, `R2_BUCKET`, and `R2_ENDPOINT`. Production requires an HTTPS endpoint.
`OBJECT_STORAGE_REQUEST_TIMEOUT` bounds provider operations and defaults to `30s`.

Asset policy is independently typed through `ASSET_MAX_BYTES`, `ASSET_UPLOAD_GRANT_TTL`,
`ASSET_DOWNLOAD_GRANT_TTL`, and `ASSET_PENDING_TTL`. Size and lifetime bounds fail startup rather
than weakening upload or cleanup behavior. See [`ASSETS.md`](ASSETS.md) for the storage boundary,
CORS/lifecycle deployment responsibilities, and privacy rules.

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
go test ./internal/infra/config ./internal/infra/email ./internal/infra/storage ./internal/infra/console/commands
go test -race ./tests/unit ./internal/infra/config ./internal/infra/email ./internal/infra/storage ./internal/infra/console/commands
```

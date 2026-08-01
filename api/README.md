# Luas API

> A product-neutral scaffold for modular Go APIs

Luas provides a stable backend project structure, dependency injection, explicit module boundaries,
shared HTTP responses, pagination, migrations, testing utilities, and replaceable infrastructure.
It is a framework and starter template, not a product application. The default assembly contains
only the minimum useful authentication, API key, and audit starters; additional business starters
must be enabled explicitly.

## Role And Boundaries

`api/` is the authoritative backend for both Luas browser applications and for
other HTTP clients. It owns domain rules, PostgreSQL persistence, migrations,
public HTTP routes, background workers, and provider integrations.

It does not own customer pages or administrative screens. Those belong to
`web/` and `admin/` respectively, which integrate with this service only
through the contracts under `../contracts/`. The API must remain deployable
without either browser application.

## Core Capabilities

- DDD-friendly modular package structure and layered request flow.
- Gin HTTP kernel with centralized route registration.
- Wire dependency injection.
- GORM persistence and versioned migrations.
- Default `user`, `apikey`, and `audit` starters.
- Provider-neutral AI capability with the `ai:chat` operator command.
- Cancellable workflow queue capability with graceful worker shutdown.
- Canonical API envelopes, dotted error codes, and request IDs.
- Bounded pagination, validation, logging, revocable sessions, and middleware.
- Test helpers and integration-test foundations.
- Optional Redis cache adapter, email, OpenTelemetry, R2, and Sentry integrations.

## Quick Start

### 1. Prerequisites

- Go 1.25.12 or later
- PostgreSQL 15, 16, 17, or 18 on a current minor release
- Redis only when a downstream application explicitly assembles the cache adapter or other shared
  infrastructure

### 2. Configure The Process

```bash
cp .env.example .env
```

Review at least these values:

```bash
APP_NAME=Luas
APP_ENV=development
SERVER_HOST=127.0.0.1
SERVER_PORT=8025

DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_NAME=luas
```

`user`, `apikey`, and `audit` are enabled by default. Organization, permission, notification,
asset, setting, usage, and webhook are optional starters. Every process that owns routes,
migrations, seeders, or workers must use the same complete dependency set.

Organization and permission:

```bash
OPTIONAL_STARTERS=organization,permission
```

The organization starter owns invitations, membership, ownership transfer, and active context.
The permission starter adds organization-scoped access roles, exact grants, transactional
assignments, and a replaceable authorizer. See
[`../contracts/ORGANIZATIONS.md`](../contracts/ORGANIZATIONS.md) and
[`../contracts/PERMISSIONS.md`](../contracts/PERMISSIONS.md).

Notifications:

```bash
OPTIONAL_STARTERS=notification
go run ./cmd/luas notification:work --batch=25 --poll=2s
```

The notification starter owns idempotent publication, user preferences, in-app read state, and
lease-driven email delivery. The HTTP process, migrations, and worker must use the same starter and
email configuration. See [`docs/NOTIFICATIONS.md`](docs/NOTIFICATIONS.md) and
[`../contracts/NOTIFICATIONS.md`](../contracts/NOTIFICATIONS.md).

Private assets:

```bash
OPTIONAL_STARTERS=asset
ASSET_TRANSFER_SIGNING_KEY=replace-with-openssl-rand-hex-32
go run ./cmd/luas asset:prune --batch=100
```

Development uses root-confined local storage. Production deployments that enable assets must
configure R2 explicitly and never fall back to the container filesystem. See
[`docs/ASSETS.md`](docs/ASSETS.md) and [`../contracts/ASSETS.md`](../contracts/ASSETS.md).

Settings and usage:

```bash
OPTIONAL_STARTERS=organization,setting
# Or: OPTIONAL_STARTERS=organization,usage
```

Settings provide a finite, code-defined catalog of app, organization, and user scalar overrides.
Usage provides trusted events, UTC counters, atomic quota decisions, and read-only summaries.
Usage events enter through a Go domain interface or operator command; no public browser ingestion,
billing, plan, or pricing semantics are included. See [`docs/SETTINGS.md`](docs/SETTINGS.md),
[`docs/USAGE.md`](docs/USAGE.md), and [`../contracts/USAGE.md`](../contracts/USAGE.md).

### 3. Generate Dependency Injection

```bash
make wire
```

### 4. Start The HTTP Server

```bash
go run ./cmd/server
```

The default loopback listener is `127.0.0.1:8025`:

- Home: `http://127.0.0.1:8025/`
- Health: `http://127.0.0.1:8025/v1/health`
- Readiness: `http://127.0.0.1:8025/health/ready`
- Prometheus metrics: `http://127.0.0.1:8025/metrics` in development. Production requires
  `METRICS_ENABLED=true` and network-level access controls.

### 5. Use The Operator CLI

```bash
go run ./cmd/luas version
go run ./cmd/luas starter:list
go run ./cmd/luas starter:list --format=json
go run ./cmd/luas starter:enable permission --dry-run
go run ./cmd/luas starter:enable permission
go run ./cmd/luas starter:check
go run ./cmd/luas starter:disable organization --cascade
DB_ENABLED=false go run ./cmd/luas route:list
go run ./cmd/luas migrate
go run ./cmd/luas seed
go run ./cmd/luas auth-session:prune --batch=500
go run ./cmd/luas audit:prune --before=2026-04-01T00:00:00Z --batch=500
go run ./cmd/luas ai:chat "Summarize this scaffold in one sentence"
```

Starter commands edit only the selected runtime `.env` file and never `.env.example`. Enabling a
starter adds transitive dependencies in deterministic order. Disabling a required dependency fails
unless `--cascade` is explicit. `--env-file` selects another runtime file, and updates use an atomic
same-directory replacement.

## Common Commands

```bash
make build
make test
make test-race-critical
make benchmark-http
make benchmark-workflow
make container-check
make compose-check
make lint
make wire
make vuln
make air
```

## Default HTTP Guardrails

The API kernel enables these core protections by default:

- `ListenAddress`: binds to `127.0.0.1`; the container explicitly sets `SERVER_HOST=0.0.0.0`.
- `TransportTimeouts`: 10-second header read, 60-second request read, 190-second response write,
  and 120-second keep-alive idle timeout.
- `HeaderLimit`: reads at most 64 KiB of request headers.
- `RequestID`: returns `X-Request-ID` and the canonical `request_id` envelope field.
- `Helmet`: emits baseline security response headers.
- `BodyLimit`: defaults to 10 MB and returns `413 COMMON.REQUEST_TOO_LARGE` when exceeded.
- `Timeout`: uses a 180-second cooperative request deadline and returns `503 COMMON.TIMEOUT` when
  the handler respects its context and has not started a response.
- `RateLimit`: enabled by default in production at `600/min` per client IP and returns
  `429 COMMON.RATE_LIMITED` when exceeded.
- `AuthAbuseGuard`: production login and password-reset paths use separate per-IP and per-subject
  budgets.
- `TrustedProxies`: forwarding headers are ignored unless the upstream appears in
  `SERVER_TRUSTED_PROXIES`.
- `CORS`: permits local browser shells by default; production must configure trusted origins.

Override the budgets through `.env`:

```bash
SERVER_HOST=127.0.0.1
SERVER_READ_TIMEOUT=60
SERVER_READ_HEADER_TIMEOUT=10
SERVER_WRITE_TIMEOUT=190
SERVER_IDLE_TIMEOUT=120
SERVER_MAX_HEADER_BYTES=65536
MIDDLEWARE_REQUEST_TIMEOUT=180
MIDDLEWARE_BODY_LIMIT_MB=10
MIDDLEWARE_RATE_LIMIT_ENABLED=true
MIDDLEWARE_RATE_LIMIT_MAX=600
MIDDLEWARE_RATE_LIMIT_WINDOW=1m
MIDDLEWARE_RATE_LIMIT_MAX_BUCKETS=10000
AUTH_RATE_LIMIT_ENABLED=true
AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE=10000
AUTH_RATE_LIMIT_LOGIN_IP_MAX=20
AUTH_RATE_LIMIT_LOGIN_SUBJECT_MAX=10
SERVER_TRUSTED_PROXIES=10.20.0.0/16
CORS_ALLOW_ORIGINS=https://app.example.com
```

`SERVER_HOST` is the socket bind address, not banner text. `SERVER_WRITE_TIMEOUT` must exceed
`MIDDLEWARE_REQUEST_TIMEOUT` so a cooperative timeout can write the standard error envelope. Set a
write timeout of `0` only when a gateway or streaming endpoint explicitly owns that deadline.
Negative transport budgets and contradictory timeout relationships fail startup validation.

Request timeouts do not preempt Gin handlers in another goroutine. They propagate a context
deadline so database, HTTP client, and AI provider calls can cancel safely. Global and auth rate
limits use bounded in-process fixed-window stores. Multi-instance deployments must enforce shared
limits at a gateway, WAF, or explicit shared adapter and must not silently fall back to independent
instance counters when that dependency fails. Auth limits do not expose bucket details and do not
replace MFA, risk detection, or progressive challenges. Compression belongs to the deployment/CDN
layer or an explicitly assembled middleware. See [`docs/MIDDLEWARE.md`](docs/MIDDLEWARE.md).

## Database-Disabled Mode

`DB_ENABLED=false` starts the API without opening a database connection. This mode supports checks
of root, health, metrics, route assembly, and downstream extraction. Default starter routes remain
registered, so authentication and validation may return their own errors first; any operation that
reaches persistence returns `503 COMMON.SERVICE_UNAVAILABLE` instead of panicking on a nil GORM
connection. Audit writes remain best-effort and cannot replace an already-generated primary
response.

This mode represents dependency degradation, not starter readiness. `/health/live` remains live,
while `/health/ready` reports the database as down with `503`. Production deployments that retain
database-backed starters must enable PostgreSQL. Only a downstream application that removes every
database-backed starter may treat database-disabled execution as complete operation.

Workflow driver ownership, payload rules, shutdown behavior, and production replacement guidance
are documented in [`docs/WORKFLOW.md`](docs/WORKFLOW.md). The `memory` driver is a bounded,
in-process, non-durable queue and cannot move work between replicas.

## Project Structure

```text
luas/api/
|-- cmd/
|   |-- server/               # HTTP server entry point
|   `-- luas/                 # Operator CLI entry point
|-- internal/
|   |-- app/                  # Application aggregate
|   |-- bootstrap/            # Startup and lifecycle
|   |-- domain/               # Domain objects and errors
|   |-- infra/                # Shared infrastructure
|   |-- modules/              # Business starters
|   `-- wiring/               # Wire dependency injection
|-- pkg/                      # Public reusable packages
|-- routes/                   # Global route entry point
|-- database/
|   |-- migrations/           # Database migrations
|   `-- seeders/              # Data initialization
`-- tests/
    |-- feature/
    |-- integration/
    `-- unit/
```

## Module Conventions

Default and optional ownership boundaries:

- `internal/modules/user`: default authentication starter with routes, migrations, and seeders.
- `internal/modules/apikey`: default API key starter with routes, migrations, and the `api_key`
  middleware group.
- `internal/modules/audit`: default audit starter for global write capture and current-user history.
- `internal/modules/asset`: optional private asset starter for ownership, metadata, inspection,
  lifecycle, and deletion; object bytes remain owned by the storage capability.

A business starter normally follows this eight-file structure:

```text
internal/modules/<module>/
|-- model.go
|-- dto.go
|-- repository.go
|-- service.go
|-- handler.go
|-- routes.go
|-- provider.go
`-- service_test.go
```

Layer flow:

```text
Handler -> Service -> Repository -> Database
DTO -> Domain -> PO
```

- Handlers own parameter binding, authorization context, and response output.
- Services own business rules and error semantics.
- Repositories own conversion between persistence objects and domain objects.
- HTTP responses use `pkg/response`.
- List endpoints use the shared pagination contract.

## Configuration Lifecycle

`internal/infra/config.Config` is the API's only typed configuration authority. Startup loads and
validates one immutable snapshot in this order: process environment, `LUAS_ENV_FILE`,
environment-local file, local file, environment file, base `.env`, then code defaults.
Configuration changes require a process restart; `make air` rebuilds and restarts during local
development.

Luas does not expose a secret-bearing configuration cache or runtime `.env` reload that cannot
atomically rebuild the dependency graph. See [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) for
the complete precedence rules, extension guidance, and `doctor` diagnostics.

## Testing

```bash
make test
make test-kest
go test ./...
go test ./tests/feature/...
go test ./tests/integration/...
```

Kest flow entry points:

- `tests/kest/auth.flow.md`
- `tests/kest/api_keys.flow.md`

Run locally:

```bash
make test-kest
./tests/kest/run_local.sh tests/kest/auth.flow.md
```

## AI Capability

`internal/capabilities/ai` is a provider-neutral technical capability with an OpenAI Responses API
adapter. It is disabled by default and does not own prompts, conversations, runs, billing, or other
product semantics.

```bash
AI_ENABLED=true
AI_DEFAULT_PROVIDER=openai
AI_DEFAULT_MODEL=provider-model
OPENAI_API_KEY=replace-me
```

```bash
go run ./cmd/luas ai:chat "Write a short project summary"
go run ./cmd/luas ai:chat --system="Answer in JSON" --model=provider-model "List 3 scaffold priorities"
```

Timeouts, input and output bounds, private errors, streaming behavior, and the downstream AI
workspace boundary are documented in [`docs/AI.md`](docs/AI.md).

## API Key Starter

The default API key starter exposes:

- `GET /v1/api-keys`
- `POST /v1/api-keys`
- `DELETE /v1/api-keys/:id`

It registers the `api_key` middleware group and `key` alias:

```go
r.Group("/v1", func(api *router.Router) {
    api.WithMiddleware("api_key")
    api.GET("/inference", handler.Run)
})
```

## Optional Integrations

These packages are optional infrastructure, not the scaffold's business identity:

- Redis cache adapter: implements `cache.Store`; downstream assembly owns the client and injection.
  No `REDIS_*` configuration silently activates it, and it is not a rate-limit driver.
- Sentry.
- OpenTelemetry.
- Resend email capability: 10-second provider timeout, 64 KiB response limit, and context
  cancellation. See [`docs/EMAIL.md`](docs/EMAIL.md).
- R2 object storage capability: AWS SDK for Go v2, short-lived signed transfers, and private error
  boundaries. The optional asset starter adds business lifecycle. See
  [`docs/ASSETS.md`](docs/ASSETS.md).

Downstream applications that do not need these integrations can retain only the core HTTP,
configuration, database, routing, and module layers.

## Deployment

The repository provides a production-oriented API image, local development Compose, and dedicated
container smoke CI. See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md). `docker-compose.yml` is a local
development topology, not a production manifest:

```bash
docker compose up --build --wait
docker compose down
```

- `Dockerfile`: pins frontend, Go builder, and distroless runtime images by version and digest;
  the runtime is non-root and contains no `.env` file.
- `.dockerignore`: excludes local binaries, test binaries, logs, coverage, and development artifacts.
- `health:check`: provides a loopback liveness probe without shell or curl dependencies.
- `LOG_STDOUT=true` and `LOG_FILE_ENABLED=false`: emit JSON request logs to stdout.
- `make container-check`: verifies BuildKit materials, OCI identity, startup, probes, logs,
  environment leakage, and SIGTERM handling.
- `make compose-check`: verifies PostgreSQL, startup migrations, readiness, and selected starters.
- `.github/workflows/container.yml`: runs the smoke test and Trivy gate and retains build metadata
  and the image SBOM.

Image publication, registry ownership, secret injection, and deployment automation remain
downstream decisions. Build inputs, CycloneDX 1.7, vulnerability gates, and the downstream Cosign
boundary are documented in [`../docs/CONTAINER_SECURITY.md`](../docs/CONTAINER_SECURITY.md).

## Design Principles

- The root repository describes scaffold capabilities, not a specific product.
- Module boundaries optimize for replacement and focused testing.
- Defaults remain minimal; additional capabilities require explicit assembly.
- The framework follows the same module rules it asks downstream applications to follow.

## License

[MIT](../LICENSE)

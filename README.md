# Luas

Luas is an open-source full-stack starter kit for building secure web applications with Go and
React. It combines a modular PostgreSQL API, production-ready business starters, a Next.js console,
and a lightweight static SPA in one coherent architecture.

Start with authentication, API keys, audit history, typed configuration, migrations, testing,
containers, and CI already working. Enable organizations, permissions, notifications, private
assets, settings, usage limits, and webhooks when the product needs them.

## What You Can Build

Luas works well for SaaS products, internal platforms, developer tools, AI applications, operations
consoles, and API-first services. The included foundations remove common setup work without forcing
product-specific business rules.

| Capability | What is ready to use |
|---|---|
| Accounts and sessions | Registration, login, profile, password change and reset, account deletion, revocable opaque sessions, idle and absolute expiry, and abuse controls |
| API access | User-owned API keys, one-time plaintext display, hash-only storage, exact scopes, listing, usage tracking, and revocation |
| Audit history | Write-request auditing, user-facing history, structured change metadata, privacy controls, and bounded retention |
| Organizations | Tenant creation, membership, invitations, active organization context, roles, member management, ownership transfer, and account-integrity guards |
| Permissions | Organization-scoped access roles, code-owned permission catalogs, exact grants, default-deny checks, and transactional assignment |
| Notifications | In-app notifications, read state, user preferences, durable email delivery, idempotent publication, leasing, and retries |
| Private assets | Upload intents, metadata, bounded validation, short-lived transfer grants, R2 support, lifecycle cleanup, and private downloads |
| Settings | Typed app, organization, and user settings with defaults, optimistic concurrency, ETags, and operator commands |
| Usage and quotas | Trusted idempotent events, UTC counters, atomic quota decisions, overrides, retention, and private usage summaries |
| Outbound webhooks | Encrypted rotating secrets, Standard Webhooks signatures, SSRF-resistant destinations, durable delivery, retries, replay, and auto-disable |

The `user`, `apikey`, and `audit` starters are enabled by default. Other business starters are
optional, so a small application does not inherit tenancy, RBAC, storage, or integration complexity
until it needs those capabilities.

## Why Luas Is Convenient

- **One architecture from API to UI.** Contracts, error codes, request IDs, pagination, feature
  names, and ownership boundaries stay consistent across deployable units.
- **Schema-backed HTTP contracts.** OpenAPI 3.1 generates independent TypeScript types for both
  browser shells, verifies Go route coverage, and blocks accidental breaking changes in CI.
- **PostgreSQL everywhere.** Runtime, migrations, repositories, integration tests, and CI use
  PostgreSQL semantics. SQLite is not used as a compatibility substitute.
- **Durable tasks without extra infrastructure.** PostgreSQL-backed jobs include idempotency,
  delayed execution, fenced leases, retries, cancellation, dead-letter state, trace propagation,
  queue lag metrics, and multi-replica worker safety without requiring Redis.
- **Two frontend choices.** Use the complete Next.js console for SSR and secure server adapters, or
  ship the Vite SPA directly through OSS/S3-compatible storage and a CDN.
- **A working console.** Authentication, account settings, API keys, audit history, and optional
  starter workflows already have feature-first UI foundations.
- **Secure defaults.** Typed startup validation, HttpOnly session custody, stable public errors,
  bounded requests, rate limits, trusted-proxy policy, private assets, and secret-safe telemetry are
  built into the foundation.
- **Replaceable providers.** Email, cache, storage, tracing, error reporting, workflows, and AI
  execution are explicit infrastructure seams rather than business-layer dependencies.
- **Operational tooling included.** Migrations, seeders, workers, pruning commands, route discovery,
  health checks, readiness, metrics, graceful shutdown, and production containers are available
  from the start.
- **Fast collaboration.** Local architecture instructions, focused verification commands, and
  maintained agent context help human and AI contributors make changes with the same vocabulary.

## Technology

| Deployable unit | Stack |
|---|---|
| `api/` | Go 1.25, Gin, Wire, GORM, PostgreSQL, typed configuration, versioned migrations, structured logging, OpenTelemetry, and operator CLI |
| `web/` | Next.js 16, React 19, TypeScript, Tailwind CSS 4, shadcn/ui, TanStack Query, Zustand, next-intl, and a secure same-origin API adapter |
| `web-spa/` | Vite 8, React 19, TanStack Router, TanStack Query, Zustand, Zod, Tailwind CSS 4, shadcn/ui, and i18next |
| Delivery | Docker Compose, non-root production images, GitHub Actions, SBOM generation, dependency scanning, container scanning, and bundle budgets |

Each deployable unit is independent. Browser applications communicate with the API through
documented HTTP contracts and never import backend source code.

## Choose A Frontend

| Need | Choose `web/` | Choose `web-spa/` |
|---|---:|---:|
| Server rendering and Server Components | Yes | No |
| Same-origin HttpOnly auth adapter | Included | Requires a reviewed gateway or browser adapter |
| Development mock BFF | Included | No |
| Full optional-starter console UI | Included | Port features as needed |
| Static OSS/CDN deployment | No | Yes |
| Frontend Node.js runtime in production | Yes | No |
| TanStack type-safe file routing | No | Yes |

Most projects select one browser shell. Choose `web/` for the broadest ready-to-use product surface;
choose `web-spa/` when static delivery, a smaller runtime, and CDN hosting are more important than
SSR or frontend server functions.

## Quick Start

### Requirements

- Docker with Compose v2 for the fastest API setup
- Go 1.25.12 or newer for native API development
- Node.js 22.12 or newer with Corepack for either browser shell

### Start The API

```bash
cd api
docker compose up --build --wait
curl -fsS http://127.0.0.1:8025/health/ready
```

The Compose stack starts PostgreSQL, builds the API, applies migrations, and waits for readiness.
The API listens on `http://127.0.0.1:8025`.

### Start The Next.js Console

```bash
cd web
corepack pnpm install
cp .env.example .env.local
corepack pnpm dev
```

Open `http://localhost:3000`. The development mock BFF makes the console immediately explorable;
production requires an explicit API adapter or backend.

### Or Start The Static SPA

```bash
cd web-spa
corepack pnpm install
cp .env.example .env.local
corepack pnpm dev
```

Open `http://127.0.0.1:4173`. Build static deployment assets with:

```bash
corepack pnpm build
```

Upload `web-spa/dist/` to OSS, S3-compatible object storage, or a CDN. The output contains no
frontend server bundle or production Node.js runtime.

## Enable Business Starters

Inspect and configure optional API starters with the CLI:

```bash
cd api
go run ./cmd/luas starter:list
go run ./cmd/luas starter:enable permission
go run ./cmd/luas starter:check
```

`starter:enable` adds required dependencies, so enabling `permission` writes
`OPTIONAL_STARTERS=organization,permission` to `.env`. Use `--dry-run` to preview a change and
`starter:list --format=json` for tooling. Deployments may still inject the additive selection
directly:

```bash
OPTIONAL_STARTERS=organization,permission,notification docker compose up --build --wait
```

Enable their matching Next.js features:

```env
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,permission,notification
```

Dependencies are explicit:

- `permission`, `setting`, `usage`, and `webhook` require `organization`.
- `notification` and `asset` can be enabled independently.
- API servers, migrations, workers, and browser features should use a compatible starter selection.

Disabling a dependency is rejected while another selected starter needs it. Use an explicit
`starter:disable organization --cascade` when removing the dependency and all selected dependents
is intended.

The [starter readiness matrix](docs/STARTER_BUSINESS_ROADMAP.md) describes every starter's workflow,
dependencies, security properties, and intentional limits.

## Architecture

```text
Next.js Console                  Static SPA
SSR + server adapters            OSS/CDN assets + browser gateway
         \                       /
          documented HTTP contracts
                     |
            Go API and workers
                     |
                 PostgreSQL
                     |
       email, R2, Redis, AI, telemetry
```

- `api/` owns domain rules, persistence, migrations, routes, workers, and provider integrations.
- `web/` owns Next.js routes, browser workflows, UI state, server adapters, and development mocks.
- `web-spa/` owns static routes, browser state, validated HTTP clients, and CDN output.
- `contracts/` owns stable request, response, `error_code`, `request_id`, pagination, and
  compatibility semantics.
- `docs/` owns architecture, security, deployment, CI, and extension guidance.

The API uses a simple vertical flow:

```text
route -> handler -> service -> repository -> PostgreSQL
```

Handlers own transport concerns, services own business rules, repositories own persistence
translation, and shared domain code remains independent of Gin and GORM.

## API And Operator Tools

The API includes a `luas` operator command:

```bash
cd api
go run ./cmd/luas version
go run ./cmd/luas starter:list
go run ./cmd/luas starter:check
DB_ENABLED=false go run ./cmd/luas route:list
DB_ENABLED=false go run ./cmd/luas route:list --format=json
go run ./cmd/luas migrate
go run ./cmd/luas seed
go run ./cmd/luas auth-session:prune --batch=500
go run ./cmd/luas audit:prune --before=2026-04-01T00:00:00Z --batch=500
```

Starter-specific commands manage notification workers, asset cleanup, setting overrides, usage
retention, webhook delivery, and other operational workflows. See [api/README.md](api/README.md)
for the complete command reference.

## HTTP Contracts

Public API behavior uses a consistent envelope, stable dotted error codes such as
`COMMON.NOT_FOUND` and `AUTH.INVALID_CREDENTIALS`, and a request ID on success and failure.
Contracts under `contracts/` document the shared semantics and each starter workflow.

Reviewed OpenAPI surfaces are validated and generated with:

```bash
cd contracts && corepack pnpm install --frozen-lockfile
make contract-check
make contract-generate
```

Runtime route discovery is deterministic and machine-readable:

```bash
cd api
DB_ENABLED=false AI_ENABLED=false go run ./cmd/luas route:list --format=json
```

This makes active routes inspectable in local development, CI, and deployment tooling without
maintaining a second handwritten route inventory.

## Security And Production

Luas includes practical application-security foundations:

- hash-only authentication sessions and API keys;
- same-origin HttpOnly credential custody in the Next.js adapter;
- strict input validation and bounded response parsing;
- body, header, timeout, pagination, queue, cache, and provider limits;
- production rate limits and separate authentication abuse budgets;
- explicit trusted-proxy and CORS configuration;
- TLS-required production PostgreSQL configuration;
- SSRF-resistant webhook destinations and secret-safe delivery records;
- non-root containers, health probes, graceful shutdown, SBOMs, and vulnerability gates.

Deployments still own secrets, TLS termination, ingress policy, PostgreSQL backups, provider
credentials, scaling, monitoring, and release approvals. Start with
[api/docs/DEPLOYMENT.md](api/docs/DEPLOYMENT.md),
[web/docs/SECURITY.md](web/docs/SECURITY.md), and
[docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md).

## Development Workflow

Run focused checks while editing and the complete gate before a release:

```bash
make agent-check
make check
```

`make check` validates governance, the Go API, both browser shells, production builds, HTTP
boundaries, and bundle budgets. Each deployable unit also provides narrower test, lint, type-check,
benchmark, container, and security commands.

## Project Layout

```text
luas/
|-- api/          # Go API, starters, workers, migrations, and CLI
|-- web/          # Next.js application and complete console
|-- web-spa/      # Static Vite and TanStack browser shell
|-- contracts/    # Shared HTTP and starter contracts
|-- docs/         # Architecture, security, delivery, and extension guides
`-- Makefile      # Workspace verification commands
```

## Documentation

| Topic | Start here |
|---|---|
| API development and commands | [api/README.md](api/README.md) |
| Next.js console | [web/README.md](web/README.md) |
| Static SPA and CDN deployment | [web-spa/README.md](web-spa/README.md) |
| Architecture and ownership | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| HTTP contracts | [contracts/README.md](contracts/README.md) |
| Add an API starter or module | [api/docs/ADDING_MODULE.md](api/docs/ADDING_MODULE.md) |
| Add a Next.js feature | [web/docs/ADDING_FEATURE.md](web/docs/ADDING_FEATURE.md) |
| Add a static SPA feature | [web-spa/docs/ADDING_FEATURE.md](web-spa/docs/ADDING_FEATURE.md) |
| Starter capability matrix | [docs/STARTER_BUSINESS_ROADMAP.md](docs/STARTER_BUSINESS_ROADMAP.md) |
| CI and releases | [docs/CI.md](docs/CI.md) and [docs/BRANCHING_AND_RELEASES.md](docs/BRANCHING_AND_RELEASES.md) |
| Dependency and container security | [docs/DEPENDENCY_SECURITY.md](docs/DEPENDENCY_SECURITY.md) and [docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md) |
| AI-assisted development | [docs/AGENT_SKILL_PERFORMANCE_GUIDE.md](docs/AGENT_SKILL_PERFORMANCE_GUIDE.md) |

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md), keep public behavior documented under `contracts/`, and run
the relevant verification tier before opening a pull request.

## Security

Report suspected vulnerabilities privately according to [SECURITY.md](SECURITY.md). Do not include
sensitive details in a public issue.

## License

Luas is available under the [MIT License](LICENSE).

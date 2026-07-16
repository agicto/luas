# Luas

Luas is an open-source full-stack application starter kit for building secure, maintainable products with Go and Next.js. It provides a production-oriented foundation, reusable business starters, explicit HTTP contracts, and a development system that works well for both teams and coding agents.

Luas is a scaffold, not a finished product. Keep the capabilities your application needs, remove the examples it does not, and build product behavior inside clear ownership boundaries.

## Why Luas

- **Start with working foundations.** Authentication sessions, API keys, audit history, configuration, health checks, migrations, a Web console, and verification tooling are already assembled.
- **Add business capabilities deliberately.** Organizations, permissions, notifications, assets, settings, usage, and webhooks are optional starters with documented dependencies and removal paths.
- **Keep deployment boundaries honest.** The Go API and Next.js Web app are independent deployable units connected through HTTP contracts, not shared source code.
- **Make safety executable.** Typed startup validation, stable error codes, bounded infrastructure, browser security headers, dependency checks, container checks, and performance budgets are enforced by tests and governance scripts.
- **Give contributors and agents the same map.** Canonical vocabulary, architecture guides, local instructions, and task-specific skills reduce guesswork as the application grows.

## What Is Included

| Area | Included foundation |
|---|---|
| API core | Go 1.25, Gin, Wire dependency injection, GORM, PostgreSQL, typed configuration, migrations, health checks, metrics, structured logging, and CLI commands |
| Default starters | User accounts and opaque authentication sessions, user-owned API keys, and audit history |
| Optional starters | Organizations, exact permissions, notifications, private assets, typed settings, usage and quotas, and outbound webhooks |
| Web shell | Next.js 16, React 19, TypeScript, Tailwind CSS 4, shadcn primitives, i18n, authenticated console routes, strict API adapters, and a development mock BFF |
| Capabilities | Cache, queue and workflow primitives, email, object storage, observability, and a bounded provider-neutral AI execution seam |
| Delivery | Non-root production containers, local Compose, readiness and liveness probes, SBOM generation, vulnerability gates, and route bundle budgets |
| Engineering system | Shared contracts, architecture decisions, semantic guardrails, tiered verification, and repository-local agent skills |

## Quick Start

### Prerequisites

- Docker with Compose v2 for the quickest API setup
- Go 1.25.12 or newer for native API development
- Node.js 22 or 24 with Corepack for Web development

### 1. Start the API

The local Compose stack builds the API, starts PostgreSQL, applies migrations, and waits for readiness:

```bash
cd api
docker compose up --build --wait
curl -fsS http://127.0.0.1:8025/health/ready
```

The API is available at `http://127.0.0.1:8025`. See [api/README.md](api/README.md) for native development, CLI, database, worker, and deployment workflows.

### 2. Start the Web app

In another terminal:

```bash
cd web
corepack pnpm install
cp .env.example .env.local
corepack pnpm dev
```

Open `http://localhost:3000`. Development mode can use the bounded mock BFF, so the Web shell is explorable before a production API adapter is configured. Production does not silently enable mock behavior.

### 3. Verify the workspace

```bash
make governance
make check
```

`make governance` runs deterministic architecture, contract, security, supply-chain, and agent-guidance checks. `make check` adds the API and Web verification tiers, including the production Web build and route bundle budgets.

## Choose Your Starters

The default API always includes `user`, `apikey`, and `audit`. Optional starters are additive and disabled until selected:

```bash
cd api
OPTIONAL_STARTERS=organization,permission docker compose up --build --wait
```

Enable matching browser features explicitly:

```env
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,permission
```

`permission`, `setting`, `usage`, and `webhook` require `organization`. `notification` and `asset` can be enabled independently. API processes, migration jobs, workers, and Web features must use a compatible selection. The [starter business roadmap](docs/STARTER_BUSINESS_ROADMAP.md) records readiness, dependencies, and deliberate deferrals for every starter.

## Architecture

```text
Browser
  -> Next.js Web (UI, same-origin adapters, development mock BFF)
  -> documented HTTP contracts
  -> Go API (core, starters, capabilities)
  -> PostgreSQL and external providers
```

- `api/` owns domain rules, persistence, migrations, HTTP routes, workers, and provider integrations.
- `web/` owns browser workflows, route groups, UI state, server adapters, and mock development flows.
- `contracts/` owns stable cross-deployable HTTP semantics such as envelopes, `error_code`, `request_id`, pagination, and starter workflows.
- `docs/` owns workspace architecture, security, delivery, and framework-quality decisions.

Read [CONTEXT.md](CONTEXT.md) for the canonical vocabulary and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for ownership boundaries and vertical change flow.

## Contracts And Route Discovery

Contract Markdown under `contracts/` is the human-reviewed source of truth. The API can also emit the routes assembled by the current configuration directly from the runtime:

```bash
cd api
DB_ENABLED=false AI_ENABLED=false go run ./cmd/luas route:list
DB_ENABLED=false AI_ENABLED=false go run ./cmd/luas route:list --format=json
```

The JSON form is deterministic and validated against a versioned schema, which makes it suitable for CI and tooling. It is a route catalog, not a generated OpenAPI description. See [api/docs/ROUTE_DISCOVERY.md](api/docs/ROUTE_DISCOVERY.md).

## Production Posture

Luas keeps environment-specific policy explicit:

- configuration is loaded into one typed startup snapshot and invalid combinations fail before serving traffic;
- browser mocks, AI providers, optional starters, and production metrics require deliberate activation;
- authentication credentials are opaque and hash-only at rest, while API key scopes only reduce authority;
- public errors use stable `DOMAIN.REASON` codes without leaking provider, SQL, or internal exception details;
- request, cache, queue, rate-limit, response, and provider boundaries are bounded rather than unbounded defaults;
- API and Web images run as non-root processes and have executable health and termination contracts;
- dependency and container inventories are emitted as validated CycloneDX SBOMs.

Production deployment still owns secrets, TLS, ingress trust, persistence, scaling, backups, provider credentials, and release policy. Start with [api/docs/DEPLOYMENT.md](api/docs/DEPLOYMENT.md), [web/docs/SECURITY.md](web/docs/SECURITY.md), and [docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md).

## AI-Assisted Development

Luas treats agent context as maintained architecture rather than an informal prompt:

1. Read [CONTEXT.md](CONTEXT.md) for global terms and boundaries.
2. Read [AGENTS.md](AGENTS.md), then the matching `api/AGENTS.md` or `web/AGENTS.md` before editing.
3. Load the task-specific workflow under `.agents/skills/` for contract changes, debugging, reviews, extraction, or verification.
4. Run `make governance` to detect vocabulary, boundary, contract, and guidance drift.

This guidance is useful to human contributors too: the same terms and checks apply regardless of who writes the change.

## Create A Downstream App

1. Decide which default and optional starters belong to the product.
2. Replace the public brand, environment defaults, and example console content.
3. Remove unused examples, devtools, mock routes, starters, and capabilities using their documented removal paths.
4. Configure the production Web adapters and provider implementations required by the product.
5. Add product contracts before cross-deployable behavior, then implement API and Web changes against them.
6. Run `make check` and the relevant container or Compose checks before release.

Use [docs/SCAFFOLD_SURFACES.md](docs/SCAFFOLD_SURFACES.md) as the keep, replace, or remove catalog.

## Documentation

| Topic | Start here |
|---|---|
| Global vocabulary | [CONTEXT.md](CONTEXT.md) |
| Architecture and extension flow | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| HTTP contracts | [contracts/README.md](contracts/README.md) |
| API development | [api/README.md](api/README.md) |
| Web development | [web/README.md](web/README.md) |
| Add an API module | [api/docs/ADDING_MODULE.md](api/docs/ADDING_MODULE.md) |
| Add a Web feature | [web/docs/ADDING_FEATURE.md](web/docs/ADDING_FEATURE.md) |
| Dependency security | [docs/DEPENDENCY_SECURITY.md](docs/DEPENDENCY_SECURITY.md) |
| Container security | [docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md) |
| CI and release branches | [docs/CI.md](docs/CI.md) and [docs/BRANCHING_AND_RELEASES.md](docs/BRANCHING_AND_RELEASES.md) |
| Long-term quality roadmap | [docs/FRAMEWORK_QUALITY_ROADMAP.md](docs/FRAMEWORK_QUALITY_ROADMAP.md) |

## Contributing

Contributions should preserve Luas as a reusable scaffold rather than add product-specific behavior. Read [CONTRIBUTING.md](CONTRIBUTING.md), keep changes scoped, update contracts and documentation with behavior, and run `make check` before opening a pull request.

## Security

Do not report suspected vulnerabilities in a public issue. Follow [SECURITY.md](SECURITY.md) for the private reporting process and supported-version policy.

## License

Luas is open-source software licensed under the [MIT License](LICENSE).

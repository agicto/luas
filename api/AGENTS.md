# AGENTS.md - Luas API

Rules for the Go backend under `api/`.

## Scope

The API uses Gin, Wire, GORM, and DDD-flavored starter modules. Preserve the
module identity `github.com/zgiai/luas/api`.

For a routine change, inspect the owning package and tests first. Read only the
document or skill that owns the active concern; do not load this entire
architecture catalog plus every linked skill.

## Skill Routing

Root skills remain available. API-specific skills are:

| Skill | Select when |
|---|---|
| `architecture-principles` | Deciding an API seam, layer, or starter/capability boundary |
| `module-creation` | Creating a route-owning default or optional starter |
| `api-development` | Adding or changing HTTP handlers, routes, validation, pagination, or response behavior |
| `database-design` | Designing persistence, indexes, query shape, or table lifecycle |
| `logging-standards` | Changing structured events, request correlation, or redaction |
| `testing-strategy` | Choosing test boundaries, doubles, or integration coverage |
| `kest-flow` | Writing Markdown API flow scenarios |
| `deployment` | Building or verifying the API runtime image and deployment |
| `sql-migration-review` | Writing or reviewing a SQL/GORM migration |

Choose one primary skill. Add another only when the change genuinely crosses
its owning boundary. Root `luas-code-review` is the review workflow; API rules
below and the owning docs supply the backend-specific standard.

## Architecture

```text
cmd/                  CLI and server entry points
internal/bootstrap/   startup and lifecycle
internal/domain/      framework-free entities, errors, and seams
internal/modules/     route-owning default and optional starters
internal/capabilities technical capabilities without product ownership
internal/infra/       provider implementations
internal/starter/     starter registry and assembly
internal/wiring/      Wire provider composition
pkg/                  deliberately public libraries
routes/               global route assembly
tests/                cross-package and integration tests
```

Default route-owning starter flow:

```text
handler DTO -> service domain -> repository PO -> database
```

A starter module normally has `model.go`, `dto.go`, `repository.go`,
`service.go`, `handler.go`, `routes.go`, `provider.go`, and focused tests.
Capabilities should not gain HTTP files merely to match that template.

### Layer Rules

- `internal/domain/` stays framework-free and standard-library-only.
- Handlers own transport concerns; services own business rules; repositories
  own persistence translation.
- Business logic uses domain values, not GORM persistence objects.
- DTO/domain/PO conversion stays explicit, normally beside DTOs or persistence.
- Prefer concrete implementations and constructors. Add interfaces only at
  real replacement or test seams.
- Keep Wire provider sets near the implementation they assemble and regenerate
  Wire output when provider graphs change.
- Never introduce reverse imports from domain/core layers into modules.

### Database Dialect

- PostgreSQL is the only relational database compatibility target for runtime,
  migrations, repositories, and integration tests.
- Never add or expand SQLite drivers, dependencies, DSNs, dialect branches,
  fixtures, or tests. Existing SQLite references are frozen migration debt,
  not a pattern to copy.
- Unit tests that do not need SQL semantics use an existing repository seam or
  test double. Tests that validate SQL, constraints, transactions, locking,
  migrations, or query shape use disposable PostgreSQL.
- When changing a legacy SQLite-backed test, migrate the touched behavior to a
  test double or PostgreSQL instead of extending the SQLite path.

## Naming

- Packages: singular lowercase.
- Files and JSON fields: `snake_case`.
- Persistence models: `{Name}PO`.
- Requests/responses: `{Action}{Name}Request`, `{Name}Response`.
- Constructors: `New{Name}`.
- Implementations are unexported unless they are a deliberate public seam.
- Sensitive fields always use `json:"-"`.
- Code and comments are English.

## HTTP Contracts

- Use the response and handler helpers; do not hand-roll envelopes.
- Non-2xx responses expose stable dotted `error_code` values and, when
  available, `request_id`. Clients never branch on message text.
- Wrap sentinel errors with `%w` so central mapping remains reliable.
- Use `handler.BindJSON()` and validation tags for request bodies.
- Paginate unbounded lists. A finite code-owned catalog needs the reviewed
  `// luas:bounded-list max=<n> reason=<reason>` marker.
- Resource URLs use plural nouns, not action verbs.
- Use `200` for reads/updates, `201` for creation, and `204` for successful
  bodyless deletion.
- Update the owning file under `../contracts/` before changing a shared public
  shape or behavior.

## Security And Runtime

- Public authentication failures stay enumeration-safe. Unknown identities
  still perform password-hash work.
- Keep independent per-IP and per-subject auth limits; do not combine them
  into one `IP+subject` key.
- `SERVER_TRUSTED_PROXIES` defaults to trust-none; trust-all CIDRs are invalid.
- A disabled database returns `domain.ErrServiceUnavailable`; it must not
  panic or masquerade as not-found/invalid-credentials.
- Read configuration through the typed startup snapshot, not scattered
  environment reads.
- Bound server/provider timeouts, body sizes, response reads, pools, queues,
  caches, and identity cardinality.
- Production images contain no environment files and run with separate
  liveness/readiness semantics.

## Authority Map

| Concern | Read |
|---|---|
| Adding a module | [docs/ADDING_MODULE.md](docs/ADDING_MODULE.md) |
| Configuration | [docs/CONFIGURATION.md](docs/CONFIGURATION.md) |
| Database and profiling | [docs/DATABASE.md](docs/DATABASE.md) |
| Cache capability | [docs/CACHE.md](docs/CACHE.md) |
| Authentication | [docs/AUTHENTICATION.md](docs/AUTHENTICATION.md) |
| Observability/privacy | [docs/OBSERVABILITY.md](docs/OBSERVABILITY.md) |
| Middleware | [docs/MIDDLEWARE.md](docs/MIDDLEWARE.md) |
| Workflow lifecycle | [docs/WORKFLOW.md](docs/WORKFLOW.md) |
| Deployment | [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) |
| Route discovery | [docs/ROUTE_DISCOVERY.md](docs/ROUTE_DISCOVERY.md) |
| Capability behavior | The matching file under `docs/` and `../contracts/` |

Open the matching capability document only when changing that capability.

## Verification

Start focused, then widen once:

```bash
# Focused package proof
go test ./internal/modules/<module>/...

# Static + focused tests (scope is optional)
bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./internal/modules/<module>/...

# Generated dependency graph
make wire

# Runtime route assembly
make route-catalog-check

# Full API tests
make test

# Performance claims only
make benchmark-cache
make benchmark-database

# Wide-impact lifecycle changes
make test-race-critical
```

Run migration, benchmark, vulnerability, Compose, and container checks when
their corresponding boundary changes. The repository release gate is
`cd .. && make check`; do not run it repeatedly during local iteration.

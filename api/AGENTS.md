# AGENTS.md — Luas API

Instructions for AI coding agents working on the Luas API half.

## Project Overview

The API half is the Go backend of the Luas scaffold. It uses Gin + Wire DI + GORM, DDD-flavored starter modules, and layered architecture.

## 📖 AGENTS.md vs Skills - Positioning

### AGENTS.md (This Document) - Quick Reference Manual

**Purpose**: One-stop quick reference for most common commands, standards, and patterns.

**Content**:
- ✅ Project structure and common commands
- ✅ **Coding standards and best practices** (mandatory)
- ✅ Quick examples and common tools
- ✅ Development guidelines and notes

**Use Cases**:
- Quick lookup for commands and tools
- Verify coding standards
- Daily development reference

**Characteristics**: Concise, fast, at-a-glance

---

### Skills System - Complete Workflow Guides

**Purpose**: In-depth workflow documentation with complete steps, scripts, and examples.

**Content**:
- ✅ Complete workflows (15+ steps)
- ✅ Full code examples
- ✅ Automation scripts
- ✅ Troubleshooting guides

**Use Cases**:
- Create new route-owning starter-style modules (complete process)
- Learn best practices (deep understanding)
- Execute complex tasks (step-by-step)

**Characteristics**: Detailed, complete, executable

---

**Relationship**: Complementary, not replacement
- 📖 **AGENTS.md**: "How to use this command?" "What's this standard?"
- 🎯 **Skills**: "How to create a starter-style module from scratch?" "What's the complete workflow?"

---

## AI Agent Skills

This project includes a **Skills System** in `.agents/skills/` that provides modular workflows and best practices for AI agents.

### What are Skills?

Skills are self-contained packages of instructions, scripts, and examples that guide AI agents through complex tasks. They use a **Progressive Disclosure Architecture**:

- **Level 1 (Metadata)**: Lightweight skill descriptions loaded at startup
- **Level 2 (Instructions)**: Detailed SKILL.md content loaded when relevant
- **Level 3 (Resources)**: Scripts and examples loaded on demand

### Available Skills

| Skill | Description | When to Use |
|-------|-------------|-------------|
| [`architecture-principles`](./.agents/skills/architecture-principles/) | Shared vocabulary for seams, depth, locality, and starter boundaries | Designing or refactoring architecture |
| [`module-creation`](./.agents/skills/module-creation/) | Create starter-style DDD modules | Creating route-owning starters or optional starters |
| [`coding-standards`](./.agents/skills/coding-standards/) | Verify code follows Luas standards | Code review, PR submission |
| [`api-development`](./.agents/skills/api-development/) | API standards: pagination, errors, REST | Developing REST APIs |
| [`logging-standards`](./.agents/skills/logging-standards/) | Structured logging, levels, context | Implementing logging, debugging |
| [`code-review-guide`](./.agents/skills/code-review-guide/) | Review process, checklists, feedback | Code review, PR submission |
| [`testing-strategy`](./.agents/skills/testing-strategy/) | Test patterns (unit, integration), mocking, table-driven tests | Writing and organizing tests |
| [`database-design`](./.agents/skills/database-design/) | Schema standards, indexing, migration, SQL optimization | Designing tables and improving DB performance |
| [`deployment`](./.agents/skills/deployment/) | Deployment workflows and checklists | Shipping to staging/production |
| [`sql-migration-review`](./.agents/skills/sql-migration-review/) | Backward compat, lock duration, index, rollback review for migrations | Reviewing or writing any DB migration |

### How AI Agents Use Skills

1. **Startup**: Scan `.agents/skills/` and load metadata (name, description)
2. **Intent Analysis**: Match user request to relevant skills
3. **Dynamic Loading**: Read full SKILL.md when needed
4. **Execution**: Follow skill workflow steps
5. **Resource Access**: Load scripts/examples as required

### For Developers

```bash
# View available skills
ls .agents/skills/

# Read a skill
cat .agents/skills/module-creation/SKILL.md

# Run validation script
.agents/skills/module-creation/scripts/validate-module.sh blog
```

See [`.agents/skills/README.md`](./.agents/skills/README.md) for detailed documentation.

## Architecture References

- [`../CONTEXT.md`](../CONTEXT.md) — global Luas vocabulary and boundary terms.
- [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) — typed configuration authority, precedence, restart lifecycle, and secrets.
- [`docs/EMAIL.md`](docs/EMAIL.md) — outbound provider timeout, cancellation, privacy, and best-effort delivery semantics.
- [`docs/NOTIFICATIONS.md`](docs/NOTIFICATIONS.md) — optional notification publication, durable lease worker, retry/privacy rules, and replacement.
- [`docs/ASSETS.md`](docs/ASSETS.md) — optional asset ownership, storage capability, inspection, cleanup, and replacement.
- [`docs/SETTINGS.md`](docs/SETTINGS.md) — optional typed setting catalog, CAS history, CLI, audit privacy, and account cleanup.
- [`docs/OBSERVABILITY.md`](docs/OBSERVABILITY.md) — request-log minimization, automatic credential redaction, safe exception diagnostics, parameterized SQL, and audit privacy.
- [`docs/MIDDLEWARE.md`](docs/MIDDLEWARE.md) — default, starter-owned, opt-in, and deployment-owned HTTP middleware.
- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) — production image, local Compose, probes, logs, migrations, and container verification.
- [`docs/WORKFLOW.md`](docs/WORKFLOW.md) — queue ownership, cancellation, shutdown, and durable-driver replacement rules.
- [`../contracts/API_KEYS.md`](../contracts/API_KEYS.md) — API key lifecycle, scope grammar, one-time plaintext, and route authorization contract.
- [`../contracts/ORGANIZATIONS.md`](../contracts/ORGANIZATIONS.md) — optional organization activation, verified active context, ownership, invitation, member lifecycle, and transfer contract.
- [`../contracts/PERMISSIONS.md`](../contracts/PERMISSIONS.md) — optional access roles, exact grants, delegated-management safety, and browser contract.
- [`../contracts/NOTIFICATIONS.md`](../contracts/NOTIFICATIONS.md) — optional user notification, preference, read-state, and delivery contract.
- [`../contracts/ASSETS.md`](../contracts/ASSETS.md) — optional private asset lifecycle, transfer grant, inspection, and deletion contract.
- [`../contracts/SETTINGS.md`](../contracts/SETTINGS.md) — optional typed app/organization/user settings and conditional HTTP contract.
- [`docs/PERMISSIONS.md`](docs/PERMISSIONS.md) — permission catalog extension, transactional checks, and authorizer replacement.

## Directory Structure

```text
luas/
├── cmd/
│   ├── luas/              # CLI tool
│   └── server/            # HTTP server entry
├── internal/
│   ├── bootstrap/         # Application startup
│   ├── domain/            # Framework-free domain vocabulary and seams
│   ├── modules/           # Route-owning starter modules
│   │   └── user/          # Example: 8 files
│   │       ├── model.go       # Database entity (UserPO)
│   │       ├── dto.go         # DTO + Mapper functions
│   │       ├── repository.go  # Data access layer
│   │       ├── service.go     # Business logic layer
│   │       ├── handler.go     # HTTP handlers
│   │       ├── routes.go      # Route registration
│   │       ├── provider.go    # Wire DI
│   │       └── service_test.go
│   ├── capabilities/      # Technical capabilities (idgen, crypto)
│   ├── infra/             # Infrastructure (33+ components)
│   ├── starter/           # Starter registry and assembly seams
│   └── wiring/            # Wire dependency injection
├── pkg/                   # Public libraries
├── routes/                # Global routes
└── tests/                 # Tests
```

## Common Commands

```bash
make build         # Build CLI
make test          # Run tests
make test-race-critical # Run queue/worker lifecycle race tests required by CI
make benchmark-http # Measure the core HTTP middleware chain with metrics off/on
make container-check # Build and exercise the production image contract
make benchmark-workflow # Measure the memory queue round trip
make lint          # Code linting
make wire          # Generate DI
make vuln          # Reachable vulnerability scan with the pinned Go tool
make air           # Rebuild and restart the development server
```

## Starter-Style Module Structure

| File | Responsibility |
|------|----------------|
| `model.go` | Database entity `UserPO` (GORM) |
| `dto.go` | Request/Response DTO + `toDomain()`/`toUserPO()` mappers |
| `repository.go` | Data access, returns `domain.User` |
| `service.go` | Business logic, uses `domain.User` |
| `handler.go` | HTTP handlers |
| `routes.go` | Route registration |
| `provider.go` | Wire ProviderSet |

## Capabilities Layer

`internal/capabilities/` provides technical helpers (e.g., `idgen`, `crypto`).

Workflow queue changes must preserve the lifecycle contract in [`docs/WORKFLOW.md`](docs/WORKFLOW.md):
memory delivery is process-local and volatile, payload ownership transfers on successful dispatch,
delayed work follows its context, and `Close` must remain idempotent and race-free.

> **📚 Full Guide**: See [`testing-strategy` skill - Mocks](./.agents/skills/testing-strategy/) for dependency patterns.

```go
id := idgen.UUID()
hash, _ := crypto.HashPassword("password")
```

---

## Domain Layer

`internal/domain/` contains framework-free domain entities, value objects, errors, `error_code`
constants, and repository seams. It must stay standard-library-only. Sensitive fields MUST use
`json:"-"`.

---

## 📋 Coding Standards (Mandatory)

> **📚 Full Guide**: See [`coding-standards` skill](./.agents/skills/coding-standards/)

### 1. Naming Quick Reference

- **Packages**: `singular`, lowercase (`package user`)
- **Files**: `snake_case` (`user_handler.go`)
- **DB Entities**: `{Name}PO` (`UserPO`)
- **DTOs**: `{Action}{Name}Request` / `{Name}Response`
- **Interfaces**: explicit seam names (`UserRepository`, `AuthService`) only when justified
- **Private Impl**: lowercase (`repository`)
- **Constructor**: `New{TypeName}` returning the concrete implementation by default
- **JSON Tags**: `snake_case` (`json:"user_id"`)

### 2. Architecture Standards

#### 8-File Starter Structure (Recommended Default)

Route-owning starter modules usually include the following 8 files:

```
internal/modules/user/
├── model.go              # 1. Database entity (UserPO)
├── dto.go                # 2. DTOs + Mapper functions
├── repository.go         # 3. Data access layer
├── service.go            # 4. Business logic layer
├── handler.go            # 5. HTTP handlers
├── routes.go             # 6. Route registration
├── provider.go           # 7. Wire DI configuration
└── service_test.go       # 8. Unit tests
```

Capabilities may intentionally omit HTTP-oriented files such as `handler.go` and `routes.go`.

**Validation**:
```bash
.agents/skills/module-creation/scripts/validate-module.sh user
```

### 2. Architecture Standards

> **📚 Full Guide**: See [`coding-standards` skill - Architecture](./.agents/skills/coding-standards/)

- **Layered Flow**: `Handler` (DTO) → `Service` (Domain) → `Repository` (PO) → `Database`.
- **8-File Starter Template**: Recommended for route-owning starter modules.
  > **🚀 Create Module**: Use [`module-creation` skill](./.agents/skills/module-creation/)

---

### 3. File Organization & Coding Patterns

Detailed requirements for each file (`model.go`, `dto.go`, etc.) are now moved to the **Skills System**:

- **Model Design**: See [`database-design` skill](./.agents/skills/database-design/)
- **API & Handlers**: See [`api-development` skill](./.agents/skills/api-development/)
- **Business Logic**: See [`coding-standards` skill](./.agents/skills/coding-standards/)
- **Testing**: See [`testing-strategy` skill](./.agents/skills/testing-strategy/)

---

### 4. Error & Security Standards

- **Errors**: Use `response.HandleError`, wrap with `fmt.Errorf("%w")`, and define package-level `Err...`.
- **Error Contract**: Non-2xx responses MUST expose stable `error_code`; do not make clients branch on message text.
- **Request Correlation**: Error responses SHOULD include `request_id`, and request logs SHOULD carry the same value.
- **HTTP Transport**: `SERVER_HOST` must control the real socket bind address. Keep read-header,
  read, write, idle, and header-size defaults wired through `config.ServerConfig`; a positive server
  write timeout must outlive the cooperative middleware request timeout.
- **Container Runtime**: Production images must not embed environment files. Keep local Compose
  development-only, liveness separate from readiness, request logs on container stdout, and
  `make container-check` aligned with Dockerfile behavior.
- **Disabled Database**: Repositories that receive nil GORM because `DB_ENABLED=false` MUST return
  `domain.ErrServiceUnavailable`; they must never dereference nil or silently turn dependency failure
  into not-found/invalid-credential behavior. Audit persistence remains best-effort.
- **Security**: Hide sensitive fields (`json:"-"`), validate inputs (`binding`), and use `crypto` capability.
- **Authentication Enumeration**: Public login/recovery failures MUST stay generic. Unknown-login paths must still perform password-hash work; never reveal disabled/existing accounts through status or `error_code`.
- **Authentication Abuse**: Keep public auth quotas starter-owned and use independent per-IP and per-subject buckets. Do not key a single bucket by `IP+subject`.
- **Proxy Trust**: Client-IP security controls depend on `SERVER_TRUSTED_PROXIES`; the default must remain trust-none, and trust-all CIDRs are forbidden.

---

### 5. API Development Quick Reference

> **📚 Full Details**: See [`api-development` skill](./.agents/skills/api-development/)

- **Pagination**: REQUIRED for list endpoints.
- **Unified Errors**: REQUIRED `response.HandleError`.
- **Success**: 200 (Success), 201 (Created), 204 (NoContent).
- **URLs**: Plural nouns, NO verbs (`/api/users`).
- **Validation**: REQUIRED `handler.BindJSON()` with tags.

#### Quick Verification

Run the validation script:
```bash
.agents/skills/api-development/scripts/validate-api.sh <module_name>
```

#### Complete Example

See [`.agents/skills/api-development/examples/complete-crud-handler.go`](./.agents/skills/api-development/examples/complete-crud-handler.go)

---

## Development Guidelines

1. **DTO includes Mapper** - Mapper functions go in `dto.go`
2. **Use Domain Layer** - Business logic uses `domain.User`
3. **Private implementations** - Struct names are unexported
4. **Constructors return concrete types by default** - expose interfaces only when a real seam exists
5. **snake_case JSON** - `json:"user_id"`
6. **English comments** - All code and comments in English
7. **Use handler package** - For ParseID, GetUserID, BindJSON
8. **Domain has JSON tags** - Sensitive fields use `json:"-"`

## Testing

```bash
# Unit tests
go test ./internal/modules/user/...

# Integration tests
go test ./tests/integration/...

# All tests
make test
```

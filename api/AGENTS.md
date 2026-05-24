# AGENTS.md

Instructions for AI coding agents working on the ZGO framework.

## Project Overview

ZGO is a modern Go framework using Domain-Driven Design (DDD) + layered architecture.

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
- Create new modules (complete process)
- Learn best practices (deep understanding)
- Execute complex tasks (step-by-step)

**Characteristics**: Detailed, complete, executable

---

**Relationship**: Complementary, not replacement
- 📖 **AGENTS.md**: "How to use this command?" "What's this standard?"
- 🎯 **Skills**: "How to create a module from scratch?" "What's the complete workflow?"

---

## AI Agent Skills

This project includes a **Skills System** in `.agent/skills/` that provides modular workflows and best practices for AI agents.

### What are Skills?

Skills are self-contained packages of instructions, scripts, and examples that guide AI agents through complex tasks. They use a **Progressive Disclosure Architecture**:

- **Level 1 (Metadata)**: Lightweight skill descriptions loaded at startup
- **Level 2 (Instructions)**: Detailed SKILL.md content loaded when relevant
- **Level 3 (Resources)**: Scripts and examples loaded on demand

### Available Skills

| Skill | Description | When to Use |
|-------|-------------|-------------|
| [`architecture-principles`](./.agent/skills/architecture-principles/) | Shared vocabulary for seams, depth, locality, and starter boundaries | Designing or refactoring architecture |
| [`module-creation`](./.agent/skills/module-creation/) | Create starter-style DDD modules | Creating new business modules |
| [`coding-standards`](./.agent/skills/coding-standards/) | Verify code follows ZGO standards | Code review, PR submission |
| [`api-development`](./.agent/skills/api-development/) | API standards: pagination, errors, REST | Developing REST APIs |
| [`logging-standards`](./.agent/skills/logging-standards/) | Structured logging, levels, context | Implementing logging, debugging |
| [`code-review-guide`](./.agent/skills/code-review-guide/) | Review process, checklists, feedback | Code review, PR submission |
| [`testing-strategy`](./.agent/skills/testing-strategy/) | Test patterns (unit, integration), mocking, table-driven tests | Writing and organizing tests |
| [`database-design`](./.agent/skills/database-design/) | Schema standards, indexing, migration, SQL optimization | Designing tables and improving DB performance |

### How AI Agents Use Skills

1. **Startup**: Scan `.agent/skills/` and load metadata (name, description)
2. **Intent Analysis**: Match user request to relevant skills
3. **Dynamic Loading**: Read full SKILL.md when needed
4. **Execution**: Follow skill workflow steps
5. **Resource Access**: Load scripts/examples as required

### For Developers

```bash
# View available skills
ls .agent/skills/

# Read a skill
cat .agent/skills/module-creation/SKILL.md

# Run validation script
.agent/skills/module-creation/scripts/validate-module.sh blog
```

See [`.agent/skills/README.md`](./.agent/skills/README.md) for detailed documentation.

## Directory Structure

```text
zgo/
├── cmd/
│   ├── zgo/              # CLI tool
│   └── server/            # HTTP server entry
├── internal/
│   ├── bootstrap/         # Application startup
│   ├── domain/            # Domain entities (core business)
│   ├── modules/           # Business modules
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
│   └── wiring/            # Wire dependency injection
├── pkg/                   # Public libraries
├── routes/                # Global routes
└── tests/                 # Tests
```

## Common Commands

```bash
make build         # Build CLI
make test          # Run tests
make lint          # Code linting
make wire          # Generate DI
make air           # Hot-reload dev server
```

## Module Structure (default starter template)

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

> **📚 Full Guide**: See [`testing-strategy` skill - Mocks](./.agent/skills/testing-strategy/) for dependency patterns.

```go
id := idgen.UUID()
hash, _ := crypto.HashPassword("password")
```

---

## Domain Layer

`internal/domain/` contains core business entities. Sensitive fields MUST use `json:"-"`.

---

## 📋 Coding Standards (Mandatory)

> **📚 Full Guide**: See [`coding-standards` skill](./.agent/skills/coding-standards/)

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

`capability` modules may intentionally omit HTTP-oriented files such as `handler.go` and `routes.go`.

**Validation**:
```bash
.agent/skills/module-creation/scripts/validate-module.sh user
```

### 2. Architecture Standards

> **📚 Full Guide**: See [`coding-standards` skill - Architecture](./.agent/skills/coding-standards/)

- **Layered Flow**: `Handler` (DTO) → `Service` (Domain) → `Repository` (PO) → `Database`.
- **8-File Starter Template**: Recommended for route-owning starter modules.
  > **🚀 Create Module**: Use [`module-creation` skill](./.agent/skills/module-creation/)

---

### 3. File Organization & Coding Patterns

Detailed requirements for each file (`model.go`, `dto.go`, etc.) are now moved to the **Skills System**:

- **Model Design**: See [`database-design` skill](./.agent/skills/database-design/)
- **API & Handlers**: See [`api-development` skill](./.agent/skills/api-development/)
- **Business Logic**: See [`coding-standards` skill](./.agent/skills/coding-standards/)
- **Testing**: See [`testing-strategy` skill](./.agent/skills/testing-strategy/)

---

### 4. Error & Security Standards

- **Errors**: Use `response.HandleError`, wrap with `fmt.Errorf("%w")`, and define package-level `Err...`.
- **Error Contract**: Non-2xx responses MUST expose stable `error_code`; do not make clients branch on message text.
- **Request Correlation**: Error responses SHOULD include `request_id`, and request logs SHOULD carry the same value.
- **Security**: Hide sensitive fields (`json:"-"`), validate inputs (`binding`), and use `crypto` capability.

---

### 5. API Development Quick Reference

> **📚 Full Details**: See [`api-development` skill](./.agent/skills/api-development/)

- **Pagination**: REQUIRED for list endpoints.
- **Unified Errors**: REQUIRED `response.HandleError`.
- **Success**: 200 (Success), 201 (Created), 204 (NoContent).
- **URLs**: Plural nouns, NO verbs (`/api/users`).
- **Validation**: REQUIRED `handler.BindJSON()` with tags.

#### Quick Verification

Run the validation script:
```bash
.agent/skills/api-development/scripts/validate-api.sh <module_name>
```

#### Complete Example

See [`.agent/skills/api-development/examples/complete-crud-handler.go`](./.agent/skills/api-development/examples/complete-crud-handler.go)

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

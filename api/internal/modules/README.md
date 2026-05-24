# Modules

Business domain modules directory, following Domain-Driven Design (DDD) patterns.

## Module Overview

| Module | Description | Type |
|--------|-------------|------|
| `user` | Default auth starter (register/login/JWT/profile) | Starter |
| `apikey` | Default API key starter (create/list/revoke + middleware) | Starter |
| `audit` | Default audit starter (global write-request logging + history API) | Starter |
| `permission` | Optional RBAC example module, not wired by default | Optional |

## Standard Starter Structure (8 files)

```text
module_name/
├── model.go        # Database entity (GORM)
├── dto.go          # DTO + Mapper functions
├── repository.go   # Data access layer (interface + impl)
├── service.go      # Business logic layer (interface + impl)
├── handler.go      # HTTP handlers
├── routes.go       # Route registration
├── provider.go     # Wire dependency injection
└── service_test.go # Unit tests (optional)
```

This is the default shape for route-owning starters and optional starters. `capability` modules may intentionally use a lighter structure.

## Module Capability Interfaces

Modules expose only the capabilities they actually need:

- `contracts.Module`: identity only, via `Name()`
- `contracts.RouteModule`: add `RegisterRoutes()`
- `contracts.MiddlewareModule`: add `RegisterMiddleware()`
- `contracts.EventModule`: add `RegisterEvents()`

The starter registry dispatches these capabilities centrally, so route/bootstrap code does not need to know which optional hooks each module supports.

### File Responsibilities

| File | Responsibility | Dependencies |
|------|----------------|--------------|
| `model.go` | Define `UserPO` database persistence object | GORM |
| `dto.go` | Request/Response structs + `toDomain()`/`toUserPO()` conversion | domain |
| `repository.go` | Database CRUD, returns `domain.User` | domain, GORM |
| `service.go` | Business logic, uses `domain.User` | domain, repository |
| `handler.go` | HTTP binding, DTO ↔ Service invocation | service, dto |
| `routes.go` | `Register(router)` route registration | handler |
| `provider.go` | Wire `ProviderSet` definition | wire |

## Domain Layer

`internal/domain/` contains core business entities **shared by all modules**:

```go
// internal/domain/user.go
type User struct {
    ID        uint
    Username  string
    Email     string
    Password  string  // Internal use, not exposed in DTO
    // ...
}
```

### Data Flow

```
HTTP Request → [handler] → DTO
                   ↓
               [service] → domain.User (business logic)
                   ↓
              [repository] → UserPO (database)
                   ↓
              [mapper] → domain.User ← return
```

## Composite Module Structure

For complex domains, use sub-modules:

```text
llm/
├── provider/       # Provider sub-module
│   ├── dto.go
│   ├── repository.go
│   ├── service.go
│   └── handler.go
├── channel/        # Channel sub-module
├── model.go        # Shared entities
└── router.go       # Unified route registration
```

## Creating New Modules

```bash
# Use CLI to generate
./zgo make:module Blog

# After generation:
# 1. Refine the generated internal/domain/<module>.go file with real business fields
# 2. Decide whether it is a starter, optional starter, or example
# 3. If it is a default starter, add its starter manifest to internal/starter/defaults.go
# 4. If it needs route middleware, implement RegisterMiddleware()
# 5. Re-generate Wire output
```

For new business modules, prefer `./zgo make:module <Name>`.
The single-file generators such as `make:service` and `make:handler` are meant to fill missing files in an existing `internal/modules/<module>/` directory, not to start a module from scratch.

## Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Entity (PO) | `{Entity}PO` | `UserPO` |
| Domain Entity | `domain.{Entity}` | `domain.User` |
| Request DTO | `{Action}{Entity}Request` | `CreateUserRequest` |
| Response DTO | `{Entity}Response` | `UserResponse` |
| Interface | `{Entity}{Layer}` or narrowed use-case name | `UserRepository`, `AuthService` |

## Best Practices

1. **DTO includes Mapper** - Conversion functions in `dto.go`, no separate file
2. **Concrete first** - Default constructors return concrete types; expose interfaces only when a real seam exists
3. **Use Domain Layer** - Business logic uses `domain.User`, don't expose `UserPO`
4. **Private implementations** - Implementation struct names are unexported
5. **Wire owns binding** - Prefer `wire.Bind(...)` over constructors returning interface by default

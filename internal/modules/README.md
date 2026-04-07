# Modules

Business domain modules directory, following Domain-Driven Design (DDD) patterns.

## Module Overview

| Module | Description | Type |
|--------|-------------|------|
| `user` | Default auth starter (register/login/JWT/profile) | Starter |
| `apikey` | Default API key starter (create/list/revoke + middleware) | Starter |
| `permission` | Optional RBAC example module, not wired by default | Optional |

## Standard Module Structure (8 files)

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
# 1. Add the handler to internal/app/app.go
# 2. Add the provider set to internal/wiring/wire.go
# 3. If the module needs route middleware, implement RegisterMiddleware()
# 4. Re-generate Wire output
```

## Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Entity (PO) | `{Entity}PO` | `UserPO` |
| Domain Entity | `domain.{Entity}` | `domain.User` |
| Request DTO | `{Action}{Entity}Request` | `CreateUserRequest` |
| Response DTO | `{Entity}Response` | `UserResponse` |
| Interface | `{Entity}{Layer}` | `UserRepository`, `UserService` |

## Best Practices

1. **DTO includes Mapper** - Conversion functions in `dto.go`, no separate file
2. **Interface + Impl together** - Repository/Service interface and impl in same file
3. **Use Domain Layer** - Business logic uses `domain.User`, don't expose `UserPO`
4. **Private implementations** - Implementation struct names are unexported
5. **Constructor returns interface** - `NewService() Service`

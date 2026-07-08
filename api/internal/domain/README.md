# Domain Layer

`internal/domain/` is the framework-free vocabulary for Luas starter behavior. It names domain
entities, value objects, domain errors, domain events, repository seams, and stable API
`error_code` constants that starter modules can share.

It is not a home for HTTP handlers, GORM persistence objects, external SDK clients, runtime
configuration, response helpers, or downstream product-specific workflows.

## Responsibilities

| Responsibility | Belongs here | Does not belong here |
|---|---|---|
| Entities | `User`, `APIKey`, `AuditLog` domain structs | GORM `*PO` persistence structs |
| Value objects | `Email`, `Username`, `Password`, status values | Request DTO validation tags |
| Domain errors | `ErrInvalidCredentials`, `ErrAPIKeyRevoked` | HTTP response envelopes |
| Error codes | `COMMON.NOT_FOUND`, `AUTH.INVALID_CREDENTIALS` constants | Web-only fallback or client-only codes |
| Repository seams | Small interfaces consumed by services | SQL queries, GORM sessions, transaction wiring |
| Domain events | Event names and payloads | Event bus infrastructure or async worker runtime |

## Boundary Rules

- `internal/domain/` may import only the Go standard library.
- It must not import `pkg/`, `internal/capabilities/`, `internal/infra/`, or `internal/modules/`.
- It must not import Gin, GORM, Redis, OpenAI SDKs, HTTP clients, or response packages.
- Starter modules implement domain repository interfaces in `internal/modules/<starter>/repository.go`.
- Assembly code may adapt domain errors or `error_code` values to HTTP behavior, but the domain package
  does not write HTTP responses itself.

These rules are enforced by
[`../../docs/PACKAGE_BOUNDARIES.md`](../../docs/PACKAGE_BOUNDARIES.md) and
`.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`.

## Interface Rules

Repository interfaces are useful when they name a real seam between domain behavior and a starter
module implementation. Keep them narrow and close to the domain concept:

```go
type UserRepository interface {
    FindByID(ctx context.Context, id uint) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}
```

Avoid broad service-like interfaces that collect unrelated use cases. Most request-driven workflow
logic belongs in `internal/modules/<starter>/service.go`, not in a generic domain service.

## Data Flow

```text
HTTP handler -> DTO -> starter service -> domain entity/interface -> repository -> PO/database
```

Typical ownership:

| Step | Owner |
|---|---|
| HTTP binding and response envelope | `internal/modules/<starter>/handler.go` |
| Request workflow and authorization decisions | `internal/modules/<starter>/service.go` |
| Domain entity, value object, error, or repository seam | `internal/domain/` |
| Persistence object and query implementation | `internal/modules/<starter>/model.go` and `repository.go` |
| Database, HTTP, logging, event bus, and middleware runtime | `internal/infra/` |

## Adding Domain Concepts

1. Start in the starter module unless the concept needs to be shared by more than one file or starter.
2. Move only framework-free vocabulary into `internal/domain/`.
3. Keep sensitive fields hidden with `json:"-"`.
4. Add or update `error_code` constants only when API behavior needs a stable machine-readable branch.
5. Run the vocabulary and package boundary checks after changing this package.

## Related Documentation

- [Module Development Guide](../modules/README.md)
- [Adding a Backend Module](../../docs/ADDING_MODULE.md)
- [API Package Boundaries](../../docs/PACKAGE_BOUNDARIES.md)

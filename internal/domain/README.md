# Domain Layer

The **Domain Layer** is the heart of ZGO framework, containing the core business logic that is independent of any infrastructure concerns.

## 📋 Responsibilities

### 1. Entity Definitions
Core business entities that represent the fundamental concepts of your application.

```go
// user.go
type User struct {
    ID        uint
    Username  string
    Email     string
    Password  string  // Hidden in JSON via `json:"-"`
    Status    int
    CreatedAt time.Time
}
```

### 2. Repository Interfaces (Contracts)
Define **what** data operations are needed, not **how** they are implemented.

```go
// user.go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id uint) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    // ...
}
```

**Why interfaces here?**
- Modules depend on domain interfaces, not concrete implementations
- This enables dependency inversion and prevents circular dependencies
- Easy to swap implementations (e.g., PostgreSQL → MongoDB)

### 3. Value Objects
Immutable objects that represent concepts with no identity.

```go
// value_objects.go
type Email struct {
    value string
}

func NewEmail(s string) (Email, error) {
    if !isValidEmail(s) {
        return Email{}, ErrInvalidEmail
    }
    return Email{value: s}, nil
}
```

### 4. Domain Events
Events that represent something significant that happened in the domain.

```go
// events.go
type UserCreatedEvent struct {
    BaseEvent
    UserID   uint
    Username string
    Email    string
}

type OrderCompletedEvent struct {
    BaseEvent
    OrderID uint
    UserID  uint
    Amount  float64
}
```

### 5. Domain Errors
Business-specific errors that are meaningful to the domain.

```go
// errors.go
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrEmailAlreadyExists = errors.New("email already exists")
)
```

---

## 📁 File Structure

```
domain/
├── README.md           # This file
├── user.go             # User entity + UserRepository interface
├── permission.go       # Permission entities + interfaces
├── value_objects.go    # Value objects (Email, Username, etc.)
├── events.go           # Domain events
├── errors.go           # Domain-specific errors
└── aggregate.go        # Aggregate roots (if using DDD aggregates)
```

---

## 🔑 Key Principles

### 1. No Infrastructure Dependencies
The domain layer should **never** import:
- Database packages (gorm, sqlx)
- HTTP frameworks (gin, echo)
- External services (redis, kafka)

### 2. Interface Segregation
Define small, focused interfaces:
```go
// ✅ Good: Small, focused interface
type UserFinder interface {
    FindByID(ctx context.Context, id uint) (*User, error)
}

// ❌ Avoid: Large, monolithic interface with unrelated methods
type UserEverything interface {
    Create, Update, Delete, FindByID, FindByEmail, SendEmail, GenerateReport...
}
```

### 3. Dependency Inversion
Modules implement domain interfaces, not the other way around:

```
┌─────────────────────┐
│      Domain         │  ← Defines interfaces (UserRepository)
└─────────────────────┘
          ▲
          │ implements
          │
┌─────────────────────┐
│   modules/user      │  ← Implements UserRepository
└─────────────────────┘
```

### 4. Breaking Circular Dependencies
If Module A needs to call Module B:

```go
// ❌ Wrong: Direct import causes circular dependency
import "modules/permission"

// ✅ Correct: Depend on domain interface
type RoleAssigner interface {
    AssignDefaultRole(ctx context.Context, userID uint) error
}
```

---

## Notes

- In ZGO, most request-driven business logic belongs in `internal/modules/*/service.go`.
- Add a dedicated domain service only when logic is shared across modules or does not fit an entity/aggregate cleanly.

## 🔄 Data Flow

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  ← Uses DTO (Request/Response)
└─────────────┘
     │
     ▼
┌─────────────┐
│   Service   │  ← Uses domain.User
└─────────────┘
     │
     ▼
┌─────────────┐
│ Repository  │  ← Implements domain.UserRepository
└─────────────┘
     │
     ▼
┌─────────────┐
│  Database   │  ← Uses internal PO (Persistence Object)
└─────────────┘
```

---

## 📖 Related Documentation

- [Module Development Guide](../../modules/README.md)
- [Wire Dependency Injection](../../../docs/dependency_injection.md)
- [Event-Driven Architecture](../events/README.md)

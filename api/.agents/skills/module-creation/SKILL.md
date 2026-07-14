---
name: module-creation
description: Complete workflow for creating a new starter-style DDD module in Luas
version: 1.0.0
category: development
tags: [module, ddd, scaffolding, architecture]
author: Luas Team
updated: 2026-04-26
---

# Module Creation Skill

## 📋 Purpose

This skill guides you through creating a starter-style DDD (Domain-Driven Design) module in the Luas API half. It treats the 8-file layout as the default template for route-owning starter modules, while keeping the architecture aligned with the current `core / starter / capability / optional starter / example` split.

Use [`architecture-principles`](../architecture-principles/) first when deciding whether a new module should exist, what kind of module it is, and whether any new seam is justified.

Luas uses a layered architecture where route-owning starter modules are usually self-contained with clear separation of concerns:
- **Model Layer** (Database entities)
- **DTO Layer** (Data Transfer Objects + Mappers)
- **Repository Layer** (Data access)
- **Service Layer** (Business logic)
- **Handler Layer** (HTTP controllers)
- **Routes** (Endpoint registration)
- **Provider** (Dependency injection)
- **Tests** (Unit/integration tests)

Capabilities and examples may intentionally use lighter structures.

## 🎯 When to Use

Use this skill when:
- Creating a new route-owning starter-style module (e.g., User, Blog, Product, Order)
- Scaffolding a default or optional starter
- Ensuring consistency with Luas's DDD architecture
- Teaching or onboarding team members to Luas patterns

## ⚙️ Prerequisites

- [ ] The repository-pinned Go toolchain is available
- [ ] Luas project cloned and set up
- [ ] Wire resolves through the pinned module tool (`make wire` / `go tool wire`)
- [ ] Basic understanding of DDD concepts
- [ ] Database connection configured

## 🧭 Step 0: Classify the Module First

Before scaffolding, decide what you are building:

| Type | Typical Location | Default Shape |
|------|------------------|---------------|
| `starter` | `internal/modules/<name>` | 8-file route-owning module + starter manifest |
| `optional starter` | `internal/modules/<name>` | same as starter, added to `OptionalManifests`, disabled unless selected by `OPTIONAL_STARTERS` |
| `capability` | `internal/capabilities/<name>` | no HTTP files unless the capability truly owns routes |
| `example` | `examples/...` or docs | optimized for teaching, not default assembly |

If the answer is `capability` or `example`, do not force the 8-file scaffold just to satisfy a template.

## 🚀 Workflow Steps

### Step 1: Define Module Scope

Before writing any code, clearly define:

**Business Requirements**:
- Module name (PascalCase): `Blog`, `UserProfile`, `Product`
- Core domain entities and their relationships
- Required operations (CRUD, custom actions)

**Technical Specification**:
```markdown
Module: Blog
Domain Entity: BlogPost
Database Table: blog_posts
API Endpoints:
  - GET    /v1/blogs          → List all posts (paginated)
  - POST   /v1/blogs          → Create new post
  - GET    /v1/blogs/:id      → Get post by ID
  - PATCH  /v1/blogs/:id      → Update post
  - DELETE /v1/blogs/:id      → Delete post
  
Fields:
  - id (uint, primary key)
  - title (string, required, max 255)
  - content (text)
  - author_id (uint, foreign key)
  - status (enum: draft/published)
  - created_at, updated_at, deleted_at (GORM standard)
```

### Step 2: Generate the Default Scaffold

```bash
go run ./cmd/luas make:module Blog
```

**Expected structure**:
```
internal/modules/blog/
├── (starter-style scaffold files will be created below)
```

### Step 3: Create Database Entity (model.go)

**Purpose**: Define the database table structure using GORM.

**File**: `model.go`

```go
package blog

import (
    "time"
    "gorm.io/gorm"
)

// BlogPostPO is the persistent object for blog posts
// Naming: {Entity}PO (Persistent Object)
type BlogPostPO struct {
    ID        uint           `gorm:"primaryKey"`
    Title     string         `gorm:"size:255;not null;index"`
    Content   string         `gorm:"type:text"`
    AuthorID  uint           `gorm:"index;not null"`
    Status    string         `gorm:"size:20;default:'draft';index"` // draft, published
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName overrides the default table name
func (BlogPostPO) TableName() string {
    return "blog_posts"
}
```

**Key points**:
- Suffix `PO` for database models
- Use GORM tags for constraints
- Add soft delete (`DeletedAt`) only when the module lifecycle needs it
- Use `TableName()` for explicit table names

### Step 4: Create Domain Entity and DTOs (dto.go)

**Purpose**: Define domain objects, request/response DTOs, and mapper functions.

**File**: `dto.go`

```go
package blog

import (
    "time"
    "github.com/zgiai/luas/api/internal/domain"
)

// ========== Domain Entity (lives in internal/domain) ==========
// Note: In practice, domain.BlogPost should be in internal/domain/blog_post.go
// For this example, we define it here, but prefer domain package

// ========== Request DTOs ==========

type CreateBlogPostRequest struct {
    Title   string `json:"title" binding:"required,max=255"`
    Content string `json:"content" binding:"required"`
    Status  string `json:"status" binding:"omitempty,oneof=draft published"`
}

type UpdateBlogPostRequest struct {
    Title   *string `json:"title" binding:"omitempty,max=255"`
    Content *string `json:"content" binding:"omitempty"`
    Status  *string `json:"status" binding:"omitempty,oneof=draft published"`
}

// ========== Response DTOs ==========

type BlogPostResponse struct {
    ID        uint      `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    AuthorID  uint      `json:"author_id"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// ========== Mapper Functions ==========

// ToBlogPostPO converts domain entity to persistent object
func ToBlogPostPO(post *domain.BlogPost) *BlogPostPO {
    return &BlogPostPO{
        ID:        post.ID,
        Title:     post.Title,
        Content:   post.Content,
        AuthorID:  post.AuthorID,
        Status:    post.Status,
        CreatedAt: post.CreatedAt,
        UpdatedAt: post.UpdatedAt,
    }
}

// FromBlogPostPO converts persistent object to domain entity
func FromBlogPostPO(po *BlogPostPO) *domain.BlogPost {
    if po == nil {
        return nil
    }
    return &domain.BlogPost{
        ID:        po.ID,
        Title:     po.Title,
        Content:   po.Content,
        AuthorID:  po.AuthorID,
        Status:    po.Status,
        CreatedAt: po.CreatedAt,
        UpdatedAt: po.UpdatedAt,
    }
}

// ToResponse converts domain entity to response DTO
func ToResponse(post *domain.BlogPost) *BlogPostResponse {
    if post == nil {
        return nil
    }
    return &BlogPostResponse{
        ID:        post.ID,
        Title:     post.Title,
        Content:   post.Content,
        AuthorID:  post.AuthorID,
        Status:    post.Status,
        CreatedAt: post.CreatedAt,
        UpdatedAt: post.UpdatedAt,
    }
}
```

**Data flow**:
```
Handler (DTO) → Service (domain.BlogPost) → Repository (BlogPostPO)
                                           ← Repository (domain.BlogPost)
                ← Service (domain.BlogPost)
Handler (DTO) ←
```

### Step 5: Create Repository Layer (repository.go)

**Purpose**: Handle all database operations, return domain entities.

**File**: `repository.go`

```go
package blog

import (
    "context"
    "github.com/zgiai/luas/api/internal/domain"
    "gorm.io/gorm"
)

// repository is the private implementation
type repository struct {
    db *gorm.DB
}

// NewRepository creates a new blog repository
func NewRepository(db *gorm.DB) *repository {
    return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, post *domain.BlogPost) error {
    po := ToBlogPostPO(post)
    if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
        return err
    }
    *post = *FromBlogPostPO(po) // Update with generated ID
    return nil
}

func (r *repository) GetByID(ctx context.Context, id uint) (*domain.BlogPost, error) {
    var po BlogPostPO
    if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
        return nil, err
    }
    return FromBlogPostPO(&po), nil
}

func (r *repository) Update(ctx context.Context, post *domain.BlogPost) error {
    po := ToBlogPostPO(post)
    return r.db.WithContext(ctx).Save(po).Error
}

func (r *repository) Delete(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).Delete(&BlogPostPO{}, id).Error
}

func (r *repository) List(ctx context.Context, page, pageSize int) ([]*domain.BlogPost, int64, error) {
    var posts []BlogPostPO
    var total int64
    
    offset := (page - 1) * pageSize
    
    if err := r.db.WithContext(ctx).Model(&BlogPostPO{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    if err := r.db.WithContext(ctx).
        Offset(offset).
        Limit(pageSize).
        Find(&posts).Error; err != nil {
        return nil, 0, err
    }
    
    result := make([]*domain.BlogPost, len(posts))
    for i, po := range posts {
        result[i] = FromBlogPostPO(&po)
    }
    
    return result, total, nil
}
```

**Key patterns**:
- Concrete-first design
- Private struct implementation
- Introduce an interface only when callers or tests truly need a seam
- Always use `context.Context`
- Convert PO ↔ Domain at repository boundary

### Step 6: Create Service Layer (service.go)

**Purpose**: Implement business logic using domain entities.

**File**: `service.go`

```go
package blog

import (
    "context"
    "errors"
    "github.com/zgiai/luas/api/internal/domain"
)

var (
    ErrBlogPostNotFound     = errors.New("blog post not found")
    ErrInvalidBlogPostData  = errors.New("invalid blog post data")
    ErrUnauthorized         = errors.New("unauthorized operation")
)

// service is the private implementation
type service struct {
    repo domain.BlogPostRepository
}

// NewService creates a new blog service
func NewService(repo domain.BlogPostRepository) *service {
    return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req *CreateBlogPostRequest, authorID uint) (*domain.BlogPost, error) {
    // Business validation
    if req.Title == "" {
        return nil, ErrInvalidBlogPostData
    }
    
    // Set default status
    status := req.Status
    if status == "" {
        status = "draft"
    }
    
    post := &domain.BlogPost{
        Title:    req.Title,
        Content:  req.Content,
        AuthorID: authorID,
        Status:   status,
    }
    
    if err := s.repo.Create(ctx, post); err != nil {
        return nil, err
    }
    
    return post, nil
}

func (s *service) GetByID(ctx context.Context, id uint) (*domain.BlogPost, error) {
    post, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, ErrBlogPostNotFound
    }
    return post, nil
}

func (s *service) Update(ctx context.Context, id uint, req *UpdateBlogPostRequest) (*domain.BlogPost, error) {
    post, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, ErrBlogPostNotFound
    }
    
    // Apply partial updates
    if req.Title != nil {
        post.Title = *req.Title
    }
    if req.Content != nil {
        post.Content = *req.Content
    }
    if req.Status != nil {
        post.Status = *req.Status
    }
    
    if err := s.repo.Update(ctx, post); err != nil {
        return nil, err
    }
    
    return post, nil
}

func (s *service) Delete(ctx context.Context, id uint) error {
    if _, err := s.repo.GetByID(ctx, id); err != nil {
        return ErrBlogPostNotFound
    }
    return s.repo.Delete(ctx, id)
}

func (s *service) List(ctx context.Context, page, pageSize int) ([]*domain.BlogPost, int64, error) {
    return s.repo.List(ctx, page, pageSize)
}
```

**Business logic examples**:
- Input validation
- Default values
- Authorization checks
- Business rules enforcement

### Step 7: Create HTTP Handlers (handler.go)

**Purpose**: Handle HTTP requests, use service layer, return responses.

**File**: `handler.go`

```go
package blog

import (
    "github.com/gin-gonic/gin"
    "github.com/zgiai/luas/api/pkg/handler"
    "github.com/zgiai/luas/api/pkg/response"
)

// Handler handles blog post HTTP requests
type Handler struct {
    service Service
}

// NewHandler creates a new blog handler
func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

// Create godoc
// @Summary Create blog post
// @Tags blogs
// @Accept json
// @Produce json
// @Param body body CreateBlogPostRequest true "Blog post data"
// @Success 201 {object} BlogPostResponse
// @Router /v1/blogs [post]
func (h *Handler) Create(c *gin.Context) {
    // Get authenticated user
    userID, ok := handler.GetUserID(c)
    if !ok {
        return // 401 already sent
    }
    
    // Bind request
    var req CreateBlogPostRequest
    if !handler.BindJSON(c, &req) {
        return // 400 already sent
    }
    
    // Call service
    post, err := h.service.Create(c.Request.Context(), &req, userID)
    if err != nil {
        response.HandleError(c, "Failed to create blog post", err)
        return
    }
    
    // Return response
    response.Created(c, ToResponse(post))
}

// Get godoc
// @Summary Get blog post by ID
// @Tags blogs
// @Produce json
// @Param id path int true "Blog post ID"
// @Success 200 {object} BlogPostResponse
// @Router /v1/blogs/{id} [get]
func (h *Handler) Get(c *gin.Context) {
    id, ok := handler.ParseID(c, "id")
    if !ok {
        return
    }
    
    post, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        response.HandleError(c, "Blog post not found", err)
        return
    }
    
    response.Success(c, ToResponse(post))
}

// Update godoc
// @Summary Update blog post
// @Tags blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog post ID"
// @Param body body UpdateBlogPostRequest true "Updated data"
// @Success 200 {object} BlogPostResponse
// @Router /v1/blogs/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
    id, ok := handler.ParseID(c, "id")
    if !ok {
        return
    }
    
    var req UpdateBlogPostRequest
    if !handler.BindJSON(c, &req) {
        return
    }
    
    post, err := h.service.Update(c.Request.Context(), id, &req)
    if err != nil {
        response.HandleError(c, "Failed to update blog post", err)
        return
    }
    
    response.Success(c, ToResponse(post))
}

// Delete godoc
// @Summary Delete blog post
// @Tags blogs
// @Param id path int true "Blog post ID"
// @Success 204
// @Router /v1/blogs/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
    id, ok := handler.ParseID(c, "id")
    if !ok {
        return
    }
    
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        response.HandleError(c, "Failed to delete blog post", err)
        return
    }
    
    response.NoContent(c)
}

// List godoc
// @Summary List blog posts (paginated)
// @Tags blogs
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse
// @Router /v1/blogs [get]
func (h *Handler) List(c *gin.Context) {
    page := handler.QueryInt(c, "page", 1)
    perPage := handler.QueryInt(c, "per_page", 20)
    
    posts, total, err := h.service.List(c.Request.Context(), page, perPage)
    if err != nil {
        response.HandleError(c, "Failed to list blog posts", err)
        return
    }
    
    // Convert to responses
    responses := make([]*BlogPostResponse, len(posts))
    for i, post := range posts {
        responses[i] = ToResponse(post)
    }
    
    // Return paginated response
    // Note: Use pkg/pagination for automatic pagination
    response.Success(c, map[string]any{
        "data": responses,
        "meta": map[string]any{
            "total":       total,
            "page":        page,
            "per_page":    perPage,
            "total_pages": (total + int64(perPage) - 1) / int64(perPage),
        },
    })
}
```

**Handler best practices**:
- Use `pkg/handler` utilities
- Use `pkg/response` for consistent responses
- Use `pkg/pagination` for every list endpoint
- Keep handlers thin (delegate to service)

### Step 8: Register Routes (routes.go)

**Purpose**: Define API endpoints and HTTP methods.

**File**: `routes.go`

```go
package blog

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes contributes routes only when the owning starter is active.
func (h *Handler) RegisterRoutes(r *router.Router) {
    r.Group("", func(auth *router.Router) {
        auth.WithMiddleware("auth")
        auth.GET("/blogs", h.List).Name("blogs.index")
        auth.POST("/blogs", h.Create).Name("blogs.store")
        auth.GET("/blogs/:id", h.Get).Name("blogs.show").WhereNumber("id")
        auth.PATCH("/blogs/:id", h.Update).Name("blogs.update").WhereNumber("id")
        auth.DELETE("/blogs/:id", h.Delete).Name("blogs.destroy").WhereNumber("id")
    })
}
```

**Route patterns**:
- Implement `assembly.RouteModule` on the Handler
- Use named routes and parameter constraints
- Resolve middleware aliases through `internal/infra/router`
- Never edit `routes/api.go` for one starter

### Step 9: Wire Dependency Injection (provider.go)

**Purpose**: Configure Wire to auto-generate DI code.

**File**: `provider.go`

```go
package blog

import "github.com/google/wire"

// ProviderSet is the Wire provider set for blog module
var ProviderSet = wire.NewSet(
    NewRepository,
    wire.Bind(new(Repository), new(*repository)),
    NewService,
    wire.Bind(new(Service), new(*service)),
    NewHandler,
)
```

**Wire pattern**:
- Export `ProviderSet`
- Bind interfaces to implementations
- Constructors must match signatures

### Step 10: Create Unit Tests (service_test.go)

**Purpose**: Test business logic in isolation.

**File**: `service_test.go`

```go
package blog

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/zgiai/luas/api/internal/domain"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, post *domain.BlogPost) error {
    args := m.Called(ctx, post)
    return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uint) (*domain.BlogPost, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.BlogPost), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, post *domain.BlogPost) error {
    args := m.Called(ctx, post)
    return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uint) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, page, pageSize int) ([]*domain.BlogPost, int64, error) {
    args := m.Called(ctx, page, pageSize)
    return args.Get(0).([]*domain.BlogPost), args.Get(1).(int64), args.Error(2)
}

// TestCreate tests the Create service method
func TestCreate(t *testing.T) {
    mockRepo := new(MockRepository)
    svc := NewService(mockRepo)
    ctx := context.Background()
    
    req := &CreateBlogPostRequest{
        Title:   "Test Post",
        Content: "Test content",
        Status:  "draft",
    }
    
    mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.BlogPost")).Return(nil)
    
    post, err := svc.Create(ctx, req, 1)
    
    assert.NoError(t, err)
    assert.NotNil(t, post)
    assert.Equal(t, "Test Post", post.Title)
    assert.Equal(t, uint(1), post.AuthorID)
    mockRepo.AssertExpectations(t)
}
```

**Testing best practices**:
- Mock repository dependencies
- Test business logic, not infrastructure
- Use table-driven tests for multiple cases

### Step 11: Add One Starter Manifest And Catalog Entry

The manifest is the locality boundary for runtime modules and bootstrap assets:

```go
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
    return assembly.NewStaticStarterManifest(
        "blog",
        assembly.WithStarterModule(handler),
        assembly.WithStarterMigrationNames("2026_07_14_000001_create_blog_posts_table"),
    )
}
```

- Add a default starter manifest to `DefaultManifests`.
- Add an optional starter provider to `starter.ProviderSet` and its manifest to
  `OptionalManifests`. Do not add it to `DefaultManifests`.
- Do not create parallel route, migration, or seeder lists.
- If the starter installs an account/resource lifecycle hook, implement
  `assembly.ActivationModule`; the hook must run only when its manifest is selected.

**Generate Wire code**:

```bash
make wire
```

Expected output:
```
wire: blog: wrote /path/to/luas/api/internal/wiring/wire_gen.go
```

### Step 12: Prove Disabled And Enabled Assembly

For an optional starter, compare the same CLI with and without selection:

```bash
DB_ENABLED=false JWT_SECRET=0123456789abcdef0123456789abcdef \
  go run ./cmd/luas route:list

DB_ENABLED=false JWT_SECRET=0123456789abcdef0123456789abcdef \
  OPTIONAL_STARTERS=blog go run ./cmd/luas route:list
```

The first output must contain no blog routes. The second must contain the module's exact route set.
Add catalog tests proving that `ConfiguredMigrations` changes by the matching migration and that an
unknown, duplicate, default, or non-canonical selection fails.

### Step 13: Create Database Migration

Create a timestamped migration under `database/migrations/` using the current migration interface:

```go
package migrations

import (
    "gorm.io/gorm"

    "github.com/zgiai/luas/api/internal/infra/migration"
    "github.com/zgiai/luas/api/internal/modules/blog"
)

func init() {
    register("2026_07_14_000001_create_blog_posts_table", &createBlogPostsTable{
        BaseMigration: migration.BaseMigration{UseTransaction: true},
    })
}

type createBlogPostsTable struct {
    migration.BaseMigration
}

func (m *createBlogPostsTable) Up(db *gorm.DB) error {
    return db.AutoMigrate(&blog.BlogPostPO{})
}

func (m *createBlogPostsTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable(&blog.BlogPostPO{})
}
```

Register the exact migration name in the starter manifest, review indexes and lock behavior with
`sql-migration-review`, then run it through the selected catalog:

```bash
OPTIONAL_STARTERS=blog go run ./cmd/luas db:migrate
```

### Step 14: Create Domain Entity

Create `internal/domain/blog_post.go`:

```go
package domain

import "time"

// BlogPost represents a blog post in the domain layer
type BlogPost struct {
    ID        uint      `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    AuthorID  uint      `json:"author_id"`
    Status    string    `json:"status"` // draft, published
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### Step 15: Validation

**Automated validation script**:

Run the validation script (see `scripts/` folder):

```bash
./.agents/skills/module-creation/scripts/validate-module.sh blog
```

**Manual checklist**:

- [ ] All 8 files created and properly structured
- [ ] Wire generation successful (`make wire`)
- [ ] Handler implements `assembly.Module` and `assembly.RouteModule`
- [ ] Manifest is present in exactly one of `DefaultManifests` or `OptionalManifests`
- [ ] Disabled/enabled route comparison matches the selected manifest
- [ ] Migration is transactional where the target database supports it, reviewed, and applied
- [ ] Unit tests passing (`go test ./internal/modules/blog/...`)
- [ ] Domain entity created in `internal/domain/`
- [ ] HTTP contract added under `../contracts/`
- [ ] Handler utilities used (`pkg/handler`, `pkg/response`)

**Test the API**:

```bash
# Start server
make air

# Test endpoints (in another terminal)
# Create
curl -X POST http://localhost:8025/v1/blogs \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Post", "content": "Hello world"}'

# List
curl http://localhost:8025/v1/blogs \
  -H "Authorization: Bearer YOUR_TOKEN"

# Get by ID
curl http://localhost:8025/v1/blogs/1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# Update
curl -X PATCH http://localhost:8025/v1/blogs/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "published"}'

# Delete
curl -X DELETE http://localhost:8025/v1/blogs/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 🔍 Troubleshooting

### Common Error 1: Wire Generation Fails

**Symptom**:
```
wire: blog: wire_gen.go:XX:YY: no provider found for ...
```

**Cause**: Missing or incorrect provider bindings

**Solution**:
1. Check `provider.go` includes all constructors:
   ```go
   var ProviderSet = wire.NewSet(
       NewRepository,
       wire.Bind(new(Repository), new(*repository)),
       NewService,
       wire.Bind(new(Service), new(*service)),
       NewHandler,
   )
   ```
2. Verify interface and implementation match
3. Ensure the optional provider is present in `starter.ProviderSet`, or the default provider is
   included through the same starter assembly seam

### Common Error 2: Routes Not Working

**Symptom**: 404 Not Found for `/v1/blogs`

**Cause**: Routes not registered

**Solution**:
1. Verify the Handler is contributed by `NewStarterManifest` and implements
   `assembly.RouteModule`.
2. For an optional starter, set `OPTIONAL_STARTERS=blog`; a disabled optional route should return
   404 by design.
3. Check middleware aliases and authentication before changing route ownership.
4. Inspect the actual assembly with
   `OPTIONAL_STARTERS=blog go run ./cmd/luas route:list`.

### Common Error 3: Database Table Not Found

**Symptom**: the database reports that `blog_posts` does not exist

**Cause**: Migration not run

**Solution**:
```bash
OPTIONAL_STARTERS=blog go run ./cmd/luas db:migrate
```

### Common Error 4: JSON Binding Fails

**Symptom**: 400 Bad Request, validation errors

**Cause**: Incorrect struct tags or request body

**Solution**:
1. Check binding tags in DTO:
   ```go
   type CreateBlogPostRequest struct {
       Title string `json:"title" binding:"required,max=255"`
   }
   ```
2. Verify request JSON matches field names (snake_case)
3. Test with `curl -v` to see actual error message

## 📚 Examples

See [`examples/blog-module-example.md`](./examples/blog-module-example.md) for the documented implementation shape of a Blog module.

## 🔗 Related Skills

- [`api-development`](../api-development/): For handler patterns and pagination
- [`testing-strategy`](../testing-strategy/): For comprehensive test coverage
- [`coding-standards`](../coding-standards/): For seam and layering rules
- [`database-design`](../database-design/): For lifecycle columns, indexes, and migration decisions
- [`sql-migration-review`](../sql-migration-review/): For reviewing the module's migrations before merge
- [`verification-before-completion`](../../../../.agents/skills/verification-before-completion/): For running verify-standards.sh after creation
- [`grill-before-build`](../../../../.agents/skills/grill-before-build/): For interviewing the user before deciding the module exists

## 📖 References

- [Luas AGENTS.md](../../../AGENTS.md) - Project development guidelines
- [DDD Layered Architecture](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- [Wire User Guide](https://github.com/google/wire/blob/main/docs/guide.md)
- [GORM Documentation](https://gorm.io/docs/)
- [Gin Web Framework](https://gin-gonic.com/docs/)

---

## 🎉 Success!

You've successfully created a starter-style DDD module using Luas's default scaffold template!

**What's next?**
1. Add more business logic and validation
2. Write integration tests
3. Generate Swagger documentation
4. Deploy and test in staging environment

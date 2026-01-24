# ZGO API Development Mandatory Standards

## 📋 Overview

This document consolidates the **MANDATORY** API development standards for the ZGO project. These rules are now enforced in `AGENTS.md` Section 7.

---

## ✅ What Has Been Added

### Location
**File**: `AGENTS.md`  
**Section**: `## 📋 Coding Standards (Mandatory)` → `### 7. API Development Standards (Mandatory)`

### New Mandatory Rules

#### 1. **Pagination (REQUIRED)**

**Rule**: All list/collection endpoints MUST implement pagination.

```go
// ✅ REQUIRED
func (h *Handler) List(c *gin.Context) {
    users, paginator, err := pagination.PaginateFromContext[*domain.User](c, h.db)
    if err != nil {
        response.HandleError(c, "Failed to fetch users", err)
        return
    }
    response.Success(c, paginator)  // Auto-detects pagination
}

// ❌ FORBIDDEN
func (h *Handler) List(c *gin.Context) {
    var users []User
    h.db.Find(&users)
    response.Success(c, users)  // NO! Missing pagination
}
```

**Standards**:
- Default page size: 20 (max: 100)
- Query parameters: `?page=1&page_size=20`
- Response includes `meta` and `links`

---

#### 2. **Unified Error Responses (REQUIRED)**

**Rule**: All error responses MUST use `pkg/response` package.

```go
// ✅ REQUIRED
response.HandleError(c, "User not found", err)
response.BadRequest(c, "Invalid input", err)
response.NotFound(c, "Resource not found", err)

// ❌ FORBIDDEN
c.JSON(404, gin.H{"error": "not found"})  // NO!
c.AbortWithStatusJSON(400, ...)            // NO!
```

**Available Functions**:
- `response.HandleError()` - Auto-maps error to status code
- `response.BadRequest()` - 400
- `response.Unauthorized()` - 401
- `response.Forbidden()` - 403
- `response.NotFound()` - 404
- `response.Conflict()` - 409
- `response.UnprocessableEntity()` - 422
- `response.ValidationFailed()` - 422 with field errors
- `response.InternalServerError()` - 500

---

#### 3. **Success Responses (REQUIRED)**

```go
// ✅ REQUIRED
response.Success(c, data)              // 200 OK
response.Created(c, newResource)       // 201 Created
response.NoContent(c)                  // 204 No Content
response.Accepted(c, task)             // 202 Accepted

// ❌ FORBIDDEN
c.JSON(200, gin.H{"data": user})       // NO!
c.JSON(201, gin.H{"user": newUser})    // NO!
```

---

#### 4. **HTTP Method Standards (REQUIRED)**

| Method | Usage | Response | Example |
|--------|-------|----------|---------|
| GET | Retrieve | 200 + data | `GET /api/users/:id` |
| POST | Create | 201 + data | `POST /api/users` |
| PATCH | Update (partial) | 200 + data | `PATCH /api/users/:id` |
| PUT | Replace (full) | 200 + data | `PUT /api/users/:id` |
| DELETE | Remove | 204 (no content) | `DELETE /api/users/:id` |

**Rules**:
- ✅ Use PATCH for partial updates (most common)
- ✅ Use PUT only for full resource replacement
- ✅ DELETE returns 204 (not 200)
- ✅ POST returns 201 with created resource

---

#### 5. **RESTful Naming (REQUIRED)**

```go
// ✅ CORRECT
GET    /api/users              // List
POST   /api/users              // Create
GET    /api/users/:id          // Get
PATCH  /api/users/:id          // Update
DELETE /api/users/:id          // Delete
GET    /api/users/:id/posts    // Nested resource

// ❌ WRONG - No verbs in URLs
GET    /api/getUsers
POST   /api/createUser
POST   /api/users/delete/:id
GET    /api/user_list
```

---

#### 6. **Request Validation (REQUIRED)**

```go
// ✅ REQUIRED - Use handler.BindJSON()
var req CreateUserRequest
if !handler.BindJSON(c, &req) {
    return  // 400 with validation errors already sent
}

// DTO with validation tags
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8,max=72"`
    Age      int    `json:"age" binding:"omitempty,gte=0,lte=150"`
}
```

**Common Tags**:
- `required` - Field is required
- `email` - Valid email format
- `min=3,max=50` - String length
- `gte=0,lte=150` - Number range
- `omitempty` - Optional field
- `oneof=active inactive` - Enum values

---

## 📊 Response Format Standards

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": {...}
}
```

### Paginated Response

```json
{
  "code": 0,
  "message": "success",
  "data": [...],
  "meta": {
    "total": 150,
    "page": 2,
    "page_size": 20,
    "total_pages": 8
  },
  "links": {
    "first": "/api/users?page=1&page_size=20",
    "last": "/api/users?page=8&page_size=20",
    "prev": "/api/users?page=1&page_size=20",
    "next": "/api/users?page=3&page_size=20"
  }
}
```

### Error Response

```json
{
  "code": 404,
  "message": "User not found",
  "error": "record not found"
}
```

### Validation Error Response

```json
{
  "code": 422,
  "message": "Validation failed",
  "errors": {
    "email": ["The email field is required"],
    "password": ["The password must be at least 8 characters"]
  }
}
```

---

## 🚫 Forbidden Patterns

### ❌ NO Manual Status Codes

```go
// ❌ WRONG
c.JSON(404, gin.H{"error": "not found"})
c.AbortWithStatusJSON(400, map[string]any{"error": "bad request"})
c.Writer.WriteHeader(500)

// ✅ CORRECT
response.NotFound(c, "User not found", err)
response.BadRequest(c, "Invalid input", err)
response.InternalServerError(c, "Server error", err)
```

### ❌ NO Unpaginated Lists

```go
// ❌ WRONG - Returns all records
func (h *Handler) List(c *gin.Context) {
    var users []User
    h.db.Find(&users)
    response.Success(c, users)
}

// ✅ CORRECT - With pagination
func (h *Handler) List(c *gin.Context) {
    users, paginator, err := pagination.PaginateFromContext[*domain.User](c, h.db)
    if err != nil {
        response.HandleError(c, "Failed to fetch users", err)
        return
    }
    response.Success(c, paginator)
}
```

### ❌ NO Inconsistent Response Formats

```go
// ❌ WRONG - Different formats
c.JSON(200, gin.H{"user": user})
c.JSON(200, gin.H{"data": users, "total": 100})
c.JSON(200, user)

// ✅ CORRECT - Always use response package
response.Success(c, user)
response.Success(c, paginator)
response.Created(c, newUser)
```

---

## ✅ Verification Checklist

Before submitting a PR with API changes, verify:

### Pagination
- [ ] All list endpoints use `pagination.PaginateFromContext[T]()`
- [ ] Response includes `meta` and `links`
- [ ] Default page size is 20, max is 100
- [ ] Query parameters are `page` and `page_size`

### Error Handling
- [ ] All errors use `response.*` functions
- [ ] No manual `c.JSON()` with status codes
- [ ] Error messages are user-friendly
- [ ] No sensitive information in error responses

### Success Responses
- [ ] `response.Success()` for 200 OK
- [ ] `response.Created()` for 201 Created
- [ ] `response.NoContent()` for 204 (DELETE)
- [ ] Consistent response format across all endpoints

### HTTP Methods
- [ ] GET for retrieval
- [ ] POST for creation (returns 201)
- [ ] PATCH for partial updates
- [ ] DELETE returns 204

### RESTful URLs
- [ ] No verbs in URLs (use HTTP methods)
- [ ] Plural resource names (`/users`, not `/user`)
- [ ] Nested resources follow `/parent/:id/child` pattern
- [ ] No underscores (use hyphens if needed)

### Validation
- [ ] All DTOs have `binding` tags
- [ ] Using `handler.BindJSON()` for auto-validation
- [ ] Validation errors return 422 with field details

---

## 📚 Examples

### Complete CRUD Example

```go
package user

import (
    "github.com/gin-gonic/gin"
    "github.com/zgiai/zgo/pkg/handler"
    "github.com/zgiai/zgo/pkg/response"
    "github.com/zgiai/zgo/pkg/pagination"
)

type Handler struct {
    service Service
    db      *gorm.DB
}

// List - GET /api/users
func (h *Handler) List(c *gin.Context) {
    users, paginator, err := pagination.PaginateFromContext[*domain.User](c, h.db)
    if err != nil {
        response.HandleError(c, "Failed to fetch users", err)
        return
    }
    response.Success(c, paginator)
}

// Get - GET /api/users/:id
func (h *Handler) Get(c *gin.Context) {
    id, ok := handler.ParseID(c, "id")
    if !ok {
        return  // 400 already sent
    }
    
    user, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        response.HandleError(c, "User not found", err)
        return
    }
    
    response.Success(c, ToResponse(user))
}

// Create - POST /api/users
func (h *Handler) Create(c *gin.Context) {
    var req CreateUserRequest
    if !handler.BindJSON(c, &req) {
        return  // 400 with validation errors already sent
    }
    
    user, err := h.service.Create(c.Request.Context(), &req)
    if err != nil {
        response.HandleError(c, "Failed to create user", err)
        return
    }
    
    response.Created(c, ToResponse(user))
}

// Update - PATCH /api/users/:id
func (h *Handler) Update(c *gin.Context) {
    id, ok := handler.ParseID(c, "id")
    if !ok {
        return
    }
    
    var req UpdateUserRequest
    if !handler.BindJSON(c, &req) {
        return
    }
    
    user, err := h.service.Update(c.Request.Context(), id, &req)
    if err != nil {
        response.HandleError(c, "Failed to update user", err)
        return
    }
    
    response.Success(c, ToResponse(user))
}

// Delete - DELETE /api/users/:id
func (h *Handler) Delete(c *gin.Context) {
    id, ok := handler.ParseID(c, "id")
    if !ok {
        return
    }
    
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        response.HandleError(c, "Failed to delete user", err)
        return
    }
    
    response.NoContent(c)  // 204 No Content
}
```

---

## 🎯 Summary

### Mandatory Requirements

1. ✅ **Pagination**: ALL list endpoints MUST use pagination
2. ✅ **Unified Errors**: ALL errors MUST use `pkg/response`
3. ✅ **Success Responses**: ALL success responses MUST use `pkg/response`
4. ✅ **HTTP Methods**: Follow RESTful standards
5. ✅ **URL Naming**: No verbs, use plural resources
6. ✅ **Validation**: Use `handler.BindJSON()` with validation tags

### Quick Reference

```go
// Pagination (MUST)
users, paginator, _ := pagination.PaginateFromContext[T](c, db)
response.Success(c, paginator)

// Errors (MUST)
response.HandleError(c, "message", err)
response.NotFound(c, "message", err)
response.BadRequest(c, "message", err)

// Success (MUST)
response.Success(c, data)       // 200
response.Created(c, resource)   // 201
response.NoContent(c)           // 204

// Validation (MUST)
if !handler.BindJSON(c, &req) {
    return
}
```

---

**Status**: ✅ Enforced in AGENTS.md  
**Updated**: 2026-01-24  
**Version**: 1.0

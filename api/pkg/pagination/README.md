# Pagination Package

This package provides one consistent, bounded pagination model for API list operations.

## Core Types

### Request

```go
type Request struct {
    Page     int    `form:"page" json:"page"`           // Page number; defaults to 1
    PageSize int    `form:"page_size" json:"page_size"` // Items per page; defaults to 10, maximum 100
    Keyword  string `form:"keyword" json:"keyword"`    // Search keyword
}
```

### Result

```go
type Result struct {
    Total    int64 `json:"total"`     // Total record count
    Page     int   `json:"page"`      // Current page
    PageSize int   `json:"page_size"` // Items per page
    LastPage int   `json:"last_page"` // Last available page
    From     int   `json:"from"`      // First position on the current page
    To       int   `json:"to"`        // Last position on the current page
}
```

## Usage

### 1. Embed The Request In A DTO

```go
type ListRequest struct {
    pagination.Request
    Status string `form:"status"`
    Plan   string `form:"plan"`
}
```

### 2. Bind Parameters In A Handler

```go
func (h *Handler) List(c *gin.Context) {
    var req ListRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    page := req.GetPage()         // Applies the default page
    pageSize := req.GetPageSize() // Applies the configured bounds
    offset := req.GetOffset()     // Calculates the SQL offset
}
```

### 3. Query In A Repository

```go
func (r *repository) List(ctx context.Context, req *ListRequest) ([]*Model, int64, error) {
    var items []*Model
    var total int64

    query := r.db.WithContext(ctx).Table("models")
    if req.Keyword != "" {
        query = query.Where("name LIKE ?", "%"+req.Keyword+"%")
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := req.GetOffset()
    if err := query.Offset(offset).Limit(req.GetPageSize()).Find(&items).Error; err != nil {
        return nil, 0, err
    }

    return items, total, nil
}
```

### 4. Use The Convenience Helpers

```go
items, result, err := pagination.Paginate[Model](db, req)
if err != nil {
    return nil, err
}

items, result, err := pagination.PaginateFromContext[Model](c, db)
```

### 5. Return The Response

```go
response.SuccessWithPagination(c, items, total, req.GetPage(), req.GetPageSize())

response.Success(c, map[string]interface{}{
    "items": items,
    "meta":  result,
})
```

## Helpers

### Request Methods

- `GetPage() int`: returns the page number, defaulting to 1.
- `GetPageSize() int`: returns the bounded page size, defaulting to 10 with a maximum of 100.
- `GetOffset() int`: returns the SQL offset.

### Package Functions

- `FromQuery(query map[string][]string) *Request`: builds a request from query parameters.
- `FromContext(c *gin.Context) *Request`: extracts a request from a Gin context.
- `BuildResult(total, page, pageSize) *Result`: builds pagination metadata.
- `Paginate[T](db, req) ([]T, *Result, error)`: performs a paginated query.

## Response Shape

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

## Constraints

1. Page defaults to 1 and page size defaults to 10.
2. Page size is capped at 100 to bound query and response cost.
3. The package integrates with `pkg/response.SuccessWithPagination()`.
4. Generic helpers preserve result type safety.

## Migration From The Legacy Paginator

```go
// Before
paginator, err := pagination.Paginate[Model](db, page, pageSize)
response.Success(c, paginator.ToMap())

// Current
items, result, err := pagination.Paginate[Model](db, req)
response.SuccessWithPagination(c, items, result.Total, result.Page, result.PageSize)
```

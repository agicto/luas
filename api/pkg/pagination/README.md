# pagination

Offset and cursor pagination for list endpoints. The response shape is defined by
[`../../../contracts/README.md`](../../../contracts/README.md); this package builds it.

## Request

`FromContext(c)` reads `page` and `per_page` from the query string.

| Constant | Value |
|---|---|
| `DefaultPage` | 1 |
| `DefaultPerPage` | 15 |
| `MaxPerPage` | 100 |

Out-of-range values fall back to the defaults or are capped at `MaxPerPage`.

## Handler usage

```go
page := pagination.FromContext(c)
items, total, err := h.service.List(c.Request.Context(), page.GetPage(), page.GetPerPage())
if err != nil {
	response.HandleError(c, "Failed to list items", err)
	return
}
paginator := pagination.NewPaginator(toResponses(items), total, page.GetPage(), page.GetPerPage())
paginator.SetPath(c.Request.URL.Path).WithQuery(c.Request.URL.Query())
response.Success(c, paginator)
```

`response.Success` recognizes a paginator and writes the envelope with `data`, `meta`
(`current_page`, `per_page`, `total`, `last_page`, `from`, `to`), and `links`.

Repositories bound their own page size: validate `page >= 1` and `1 <= pageSize <= 100` and return
`domain.ErrInvalidInput` otherwise, as the starter repositories do.

## Query helpers

`Paginate[T](db, req)` and `New[T](c, db)` run the count and the page query for a GORM scope and
return the rows with a `*Paginator[T]`. `CursorPaginate` and `NewCursor` are the cursor variants for
lists that must not count.

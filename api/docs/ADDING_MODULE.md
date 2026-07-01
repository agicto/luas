# Adding an API Module

Use this path for starter-style, route-owning backend behavior.

## Steps

1. Name the module after the domain concept, not the transport action.
2. Add or update the HTTP contract in `../../contracts/` when the web app will call it.
3. Create `internal/modules/<name>/`.
4. Keep the starter-style files local to the module:

```text
model.go
dto.go
repository.go
service.go
handler.go
routes.go
provider.go
service_test.go
```

5. Wire the provider through the existing Wire setup.
6. Register routes through the module route registration path.
7. Run `make test` in `api/`, or `make check` from the repo root.

## Design Rules

- Keep handler code transport-focused: bind DTOs, call service, return responses.
- Keep business rules in `service.go`.
- Keep persistence details in `repository.go`.
- Return domain values from repositories when practical; avoid leaking GORM models upward.
- Add a test at the service seam before broad handler tests.

## When Not To Add a Module

Use `internal/capabilities/` for technical adapters that do not own application routes, such as ID generation, crypto, AI clients, queues, or storage.

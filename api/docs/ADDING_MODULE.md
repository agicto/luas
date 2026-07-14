# Adding an API Module

Use this path for starter-style, route-owning backend behavior.

## Steps

1. Name the module after the domain concept, not the transport action.
2. Add or update the HTTP contract in `../../contracts/` before exposing a route, even when Web does not consume it yet.
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

5. Add one `NewStarterManifest` that owns the module, migration names, seeder names, and any optional runtime hook.
6. For a default starter, add the manifest to `DefaultManifests`; for an optional starter, add its provider and manifest to the optional catalog without editing `routes/api.go`.
7. Generate Wire with `make wire`. Routes register through the selected manifest's module.
8. Verify both disabled and enabled assembly, including route and migration parity.
9. Run `make test` in `api/`, or `make check` from the repo root.

## Design Rules

- Keep handler code transport-focused: bind DTOs, call service, return responses.
- Keep business rules in `service.go`.
- Keep persistence details in `repository.go`.
- Return domain values from repositories when practical; avoid leaking GORM models upward.
- Add a test at the service seam before broad handler tests.
- Keep optional activation additive. Never use the optional list to subtract defaults.
- Ensure account/resource lifecycle hooks activate only with their owning starter.
- Pass the same `OPTIONAL_STARTERS` value to HTTP, migration, and seeder processes.

## Optional Starter Verification

```bash
DB_ENABLED=false JWT_SECRET=0123456789abcdef0123456789abcdef \
  go run ./cmd/luas route:list

DB_ENABLED=false JWT_SECRET=0123456789abcdef0123456789abcdef \
  OPTIONAL_STARTERS=<name> go run ./cmd/luas route:list

go test ./internal/starter/... ./internal/modules/<name>/...
```

The first command must show no optional routes. The second must show exactly the selected routes,
and `ConfiguredMigrations` tests must prove the matching migration set. Unknown, duplicate, default,
or non-canonical starter names must fail rather than being ignored.

## When Not To Add a Module

Use `internal/capabilities/` for technical adapters that do not own application routes, such as ID generation, crypto, AI clients, queues, or storage.

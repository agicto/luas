---
name: module-creation
description: Create a route-owning Luas API starter module. Use for new default or optional business starters, not technical capabilities, local helpers, or examples.
---

# Module Creation

## Purpose

Create one complete, contract-backed starter module without forcing the
starter template onto capabilities or examples.

## Classify First

Before generating files, decide:

| Type | Location | Shape |
|---|---|---|
| Default starter | `internal/modules/<name>` | Route-owning module plus default manifest |
| Optional starter | `internal/modules/<name>` | Same quality, explicit activation |
| Capability | `internal/capabilities/<name>` | Technical seam; no HTTP files unless it owns routes |
| Example | `examples/` or docs | Teaching surface; not assembled by default |

Use `architecture-principles` only if this classification or a new seam is
actually unresolved.

## Inputs

Establish:

- domain concept and owner
- first caller-visible behavior
- public routes and error cases
- persistence and migration needs
- default or optional activation
- dependencies on other starters
- focused tests that prove disabled and enabled assembly

Update the owning file under `../../../../contracts/` before exposing new HTTP
behavior.

## Workflow

1. **Generate or create the package**

   ```bash
   go run ./cmd/luas make:module <Name>
   ```

   The default route-owning shape is:

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

2. **Implement the vertical flow**
   - `model.go`: persistence objects and table behavior.
   - `dto.go`: transport types and explicit domain/PO mapping.
   - `repository.go`: persistence only; return domain values upward.
   - `service.go`: business rules and stable domain errors.
   - `handler.go`: bind, call service, return shared responses.
   - `routes.go`: plural resource routes and named middleware.
   - `provider.go`: the module's Wire provider set.
   - tests: service behavior plus public contract-sensitive paths.

3. **Add one starter manifest**
   - Own provider, module, migration, seeder, command, and runtime hook
     registration in one manifest.
   - Add default starters to `DefaultManifests`.
   - Add optional starters to the provider catalog and `OptionalManifests`.
   - Declare prerequisites with `WithStarterDependencies`.
   - Do not edit `routes/api.go` to special-case an optional starter.

4. **Add persistence safely**
   - Use the migration conventions and `sql-migration-review`.
   - Keep migration names stable and owned by the manifest.
   - Ensure disabled database behavior returns service-unavailable rather than
     panicking.

5. **Regenerate assembly**

   ```bash
   make wire
   ```

6. **Prove activation**
   - Default starter: routes and migrations appear without optional config.
   - Optional starter: routes and migrations are absent by default and appear
     only when selected.
   - Unknown, duplicate, default, non-canonical, missing-dependency, and cyclic
     optional selections fail closed.

## Focused Verification

```bash
./.agents/skills/module-creation/scripts/validate-module.sh <name>
go test ./internal/modules/<name>/... ./internal/starter/...

DB_ENABLED=false go run ./cmd/luas route:list
DB_ENABLED=false OPTIONAL_STARTERS=<name> go run ./cmd/luas route:list

make route-catalog-check
```

Run the repository `make check` once before release when contracts or both
halves changed.

## Conditional References

- Canonical checklist:
  [`../../../docs/ADDING_MODULE.md`](../../../docs/ADDING_MODULE.md)
- Generated-shape example:
  [`examples/blog-module-example.md`](examples/blog-module-example.md)
- HTTP rules: [`../api-development/`](../api-development/)
- Persistence rules: [`../database-design/`](../database-design/)
- Test boundaries: [`../testing-strategy/`](../testing-strategy/)

Read the example only when the nearby modules do not answer an implementation
question. Existing production modules are preferred evidence.

## Boundaries

- Do not create interfaces solely because a template or mock expects them.
- Do not let handlers own business or persistence rules.
- Do not expose admin/control-plane routes by default.
- Do not infer starter order from imports or silently auto-enable dependencies.
- Do not authorize organization scope from raw request IDs; use the verified
  organization context described by the contract.
- Do not add product-specific behavior to the scaffold.

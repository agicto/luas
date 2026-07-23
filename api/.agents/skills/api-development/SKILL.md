---
name: api-development
description: Implement Luas HTTP routes and handlers. Use for endpoint, validation, pagination, response, status, error_code, or route-middleware changes in the Go API.
---

# API Development

## Purpose

Implement HTTP behavior at the route-owning module seam while preserving the
shared contract, thin handlers, and centralized response semantics.

## Read First

Read only:

1. the owning module and its tests
2. `../../../../contracts/README.md` and the owning capability contract
3. `../../../AGENTS.md`
4. `examples/complete-crud-handler.go` only when no nearby production handler
   demonstrates the needed pattern

Use root `contract-evolution` when API, Web, mock BFF, or public documentation
must move together.

## Workflow

1. **Define behavior**
   - Method and plural resource path.
   - Authentication/authorization middleware.
   - Request DTO and validation.
   - Success status/body.
   - Stable domain error and dotted `error_code`.
   - Pagination or finite-list bound.

2. **Update the contract first**
   - Keep JSON fields `snake_case`.
   - Document compatibility, errors, and pagination.
   - Preserve the shared envelope and `request_id`.

3. **Implement the route**
   - Register named middleware before the handler.
   - Use route parameters for resource identity and verified context for
     authenticated ownership.
   - Do not put action verbs in resource paths.

4. **Implement the handler**
   - Parse IDs with handler helpers.
   - Bind request bodies with `handler.BindJSON()`.
   - Call one service operation.
   - Return through `response.Success`, `response.Created`,
     `response.NoContent`, or central error handling.
   - Never expose provider details or branch public behavior on raw error text.

5. **Implement list behavior**
   - Paginate every unbounded list.
   - A finite code-owned catalog may use exactly one reviewed marker:

     ```go
     // luas:bounded-list max=<1..100> reason=<kebab-case>
     ```

   - Keep ordering deterministic and queries bounded.

6. **Preserve authentication semantics**
   - Public credential failures remain generic.
   - Authorization uses verified identity/organization context.
   - `401` means no valid session; `403` means authenticated but denied.
   - Dependency outages remain service-unavailable, not false auth failure.

7. **Test observable behavior**
   - success status and response shape
   - malformed JSON and validation failure
   - one mapped domain error
   - auth/permission failure when protected
   - pagination or finite bound for lists

## Contract Rules

| Behavior | Rule |
|---|---|
| Read/update | `200` with shared success envelope |
| Create | `201` with created data |
| Delete without body | `204` |
| Malformed JSON | `400 COMMON.INVALID_INPUT` |
| Field/schema validation | `422 COMMON.VALIDATION_FAILED` |
| Domain failure | Central mapping to status plus dotted `error_code` |
| Diagnostics | Carry `request_id`; keep messages non-contractual |

Use package-level sentinel errors and wrap them with `%w`. Do not compare error
strings or return ad hoc `gin.H` error bodies.

## Verification

```bash
./.agents/skills/api-development/scripts/validate-api.sh <module>
go test ./internal/modules/<module>/...
bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./internal/modules/<module>/...
```

Also run:

- route discovery check when route assembly changes
- permission/auth guards when those contracts change
- database profiling only when query performance is claimed
- root `make check` once for a cross-boundary or release gate

## Boundaries

- Handler: transport only.
- Service: business decisions.
- Repository: persistence and domain translation.
- Contract: public behavior.
- Mock BFF: browser-compatible development substitute, never API authority.

Do not widen a service/repository interface merely to satisfy a handler test.
Do not use full repository verification as the first feedback loop.

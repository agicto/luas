---
name: contract-evolution
description: Evolve public HTTP behavior across a contract and multiple Luas deployable units. Do not use for a local endpoint implementation that preserves its contract.
---

# Contract Evolution

## Purpose

Keep Luas HTTP behavior aligned across `contracts/`, `api/`, selected browser-shell services, and
the development mock BFF. Use this skill when a change can alter request shape, response shape,
HTTP status, `error_code`, `request_id`, pagination, validation errors, or mock production
guardrails.

## Source Material

Read only the authorities touched by the change:

1. `contracts/README.md` and the owning capability contract.
2. Each affected deployable unit's `AGENTS.md`.
3. Relevant API module or browser feature docs.
4. `web/docs/MOCK_BFF.md` only when mock route handlers are involved.
5. `CONTEXT.md` only when vocabulary or ownership changes.

## Workflow

1. **Classify the change**
   - Additive: new optional field, new endpoint, or new documented `error_code`.
   - Behavioral: status code, validation, pagination, auth, rate-limit, or timeout behavior changes.
   - Breaking: renamed or removed field, changed meaning, changed required field, changed response envelope.
   - Mock-only: development behavior that must still preserve the real API contract shape.
   - Use `grill-before-build` only when a high-impact contract decision remains unresolved after repository discovery.

2. **Update the contract first**
   - Document request and response shape in `contracts/README.md` or the owning contract doc before changing multiple deployable units.
   - Keep JSON fields `snake_case`.
   - Use `code` only for transport/success status and `error_code` for client branching.
   - Include `request_id` when the API can carry one.
   - For validation, keep malformed JSON at `400 COMMON.INVALID_INPUT` and schema or field failures at `422 COMMON.VALIDATION_FAILED` with `errors`.

3. **Update the API behavior**
   - Keep API code inside `api/`; do not import either browser shell.
   - Use response helpers and central error mapping instead of ad hoc response shapes.
   - Add or update tests at the handler or public module seam for success and at least one error path.
   - If adding an `error_code`, update the API domain/response mapping and any API contract tests that assert codes.

4. **Update browser client behavior**
   - Update feature service types, request/response DTOs, hooks, and UI error handling at the feature seam.
   - Use backend `ApiErrorCode` for server contract values and `ClientErrorCode` only for client-owned failures such as network, timeout, or invalid successful-response data.
   - Validate security- or state-sensitive success payloads at the network boundary; TypeScript DTOs do not validate external JSON.
   - Select user-facing copy from stable local mappings. Use backend field-error keys for control association, not backend messages as display copy.
   - Keep both browser shells talking to API behavior over HTTP only; they do not import each other.
   - In `admin/`, keep fixed paths in feature services and validate important responses with Zod.
   - Add or update tests in every changed browser shell for contract-sensitive parsing, error handling, or route behavior.

5. **Update mock BFF behavior**
   - Mock route handlers must call `guardMockBffRoute()` before reading request bodies or touching mock state.
   - Unsafe mock handlers must then call `guardSameOriginMutation(request)` before parsing or mutation.
   - Mock success responses must use `apiSuccessResponse()` so they emit `{ code: 0, message: "success", data }`.
   - Mock responses must emit the shared envelope and the browser-facing contract of the production endpoint or adapter they substitute, including `error_code` for non-2xx responses.
   - Production-disabled mock BFF routes must return `503 COMMON.SERVICE_UNAVAILABLE`.
   - Update `web/docs/MOCK_BFF.md` when mock route, demo credential, or deletion/replacement instructions change.

6. **Search for drift**
   - Search old endpoint paths, field names, `error_code` values, and legacy underscore codes.
   - Confirm docs, API behavior, Web services, adapters, mock BFF behavior, and tests describe the same ownership and mappings.
   - Prefer explicit contract docs over generated shared source until Luas intentionally adopts codegen.

## Verification

Pick the narrowest commands that prove the whole changed contract, then run broader checks when
multiple deployable units move.

- Contract/docs only:
  - `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
  - `python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py`
  - `python3 .agents/skills/luas-framework-review/scripts/check-auth-contract-boundary.py` for authentication changes
  - `git diff --check`
- API behavior:
  - `cd api && go test ./internal/modules/<module>/...`
  - `cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1`
- Web behavior or mock BFF:
  - `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`
  - `cd web && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 0`
- Admin Console behavior:
  - `cd admin && pnpm vitest run src/http/client.test.ts`
  - `cd admin && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 0`
- Cross-boundary change:
  - `make check`
  - targeted `rg` scans for old paths, fields, and error codes

## Anti-patterns

- Changing API or Web behavior before updating the contract.
- Letting mock BFF routes drift from the shared envelope or their owning browser contract.
- Adding a new `error_code` only in Web or only in API.
- Branching client behavior on `message` text or numeric `code` alone.
- Treating generated examples, devtools, or mock flows as production API behavior.
- Sharing source code between deployable units to avoid writing down the contract.

## Related Skills

Select one only when its distinct concern is active:

- `downstream-app-extraction` for converting the scaffold into a product app.
- `tdd-regression` for a known contract regression.
- `api-development` for API handlers and routes.
- `api-error-handling` for Web error parsing and user-facing mappings.

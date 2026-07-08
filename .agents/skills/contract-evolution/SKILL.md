---
name: contract-evolution
description: Guide HTTP contract changes across contracts/, api/, Web services, and mock BFF. Use for endpoints, envelopes, error_code, request_id, pagination, validation, or mock production guardrails.
---

# Contract Evolution

## Purpose

Keep Luas HTTP behavior aligned across `contracts/`, `api/`, Web services, and the development mock BFF. Use this skill when a change can alter request shape, response shape, HTTP status, `error_code`, `request_id`, pagination, validation errors, or mock production guardrails.

## Source Material

Read these before changing behavior:

1. `CONTEXT.md` for `contract`, `mock BFF`, `error_code`, `request_id`, `starter`, and `feature` vocabulary.
2. `contracts/README.md` for the canonical HTTP envelope, errors, pagination, and compatibility checklist.
3. `AGENTS.md`, plus `api/AGENTS.md` and `web/AGENTS.md` when both halves are touched.
4. Relevant API module docs, Web feature docs, and `web/docs/MOCK_BFF.md` when mock route handlers are involved.

## Workflow

1. **Classify the change**
   - Additive: new optional field, new endpoint, or new documented `error_code`.
   - Behavioral: status code, validation, pagination, auth, rate-limit, or timeout behavior changes.
   - Breaking: renamed or removed field, changed meaning, changed required field, changed response envelope.
   - Mock-only: development behavior that must still preserve the real API contract shape.
   - If the change touches persistence, permissions, deployment, or user workflows, use `grill-before-build` first.

2. **Update the contract first**
   - Document request and response shape in `contracts/README.md` or the owning contract doc before changing both halves.
   - Keep JSON fields `snake_case`.
   - Use `code` only for transport/success status and `error_code` for client branching.
   - Include `request_id` when the API can carry one.
   - For validation, keep malformed JSON at `400 COMMON.INVALID_INPUT` and schema or field failures at `422 COMMON.VALIDATION_FAILED` with `errors`.

3. **Update the API behavior**
   - Keep API code inside `api/`; do not import Web code or share source with Web.
   - Use response helpers and central error mapping instead of ad hoc response shapes.
   - Add or update tests at the handler or public module seam for success and at least one error path.
   - If adding an `error_code`, update the API domain/response mapping and any API contract tests that assert codes.

4. **Update Web client behavior**
   - Update feature service types, request/response DTOs, hooks, and UI error handling at the feature seam.
   - Use backend `ApiErrorCode` for server contract values and `ClientErrorCode` only when no backend response exists.
   - Keep Web code talking to API behavior over HTTP only.
   - Add or update Web tests for contract-sensitive parsing, error handling, or route behavior.

5. **Update mock BFF behavior**
   - Mock route handlers must call `guardMockBffRoute()` before reading request bodies or touching mock state.
   - Mock responses must emit the same envelope shape as the real API, including `error_code` for non-2xx responses.
   - Production-disabled mock BFF routes must return `503 COMMON.SERVICE_UNAVAILABLE`.
   - Update `web/docs/MOCK_BFF.md` when mock route, demo credential, or deletion/replacement instructions change.

6. **Search for drift**
   - Search old endpoint paths, field names, `error_code` values, and legacy underscore codes.
   - Confirm docs, API behavior, Web services, mock BFF behavior, and tests describe the same contract.
   - Prefer explicit contract docs over generated shared source until Luas intentionally adopts codegen.

## Verification

Pick the narrowest commands that prove the whole changed contract, then run broader checks when both halves move.

- Contract/docs only:
  - `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
  - `git diff --check`
- API behavior:
  - `cd api && go test ./internal/modules/<module>/...`
  - `cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1`
- Web behavior or mock BFF:
  - `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`
  - `cd web && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 2`
- Cross-boundary change:
  - `make check`
  - targeted `rg` scans for old paths, fields, and error codes

## Anti-patterns

- Changing API or Web behavior before updating the contract.
- Letting mock BFF routes drift from the real API envelope.
- Adding a new `error_code` only in Web or only in API.
- Branching client behavior on `message` text or numeric `code` alone.
- Treating generated examples, devtools, or mock flows as production API behavior.
- Sharing source code between `api/` and `web/` to avoid writing down the contract.

## Pair With

- `luas-framework-review` for global scaffold impact and ranking.
- `api-development` when implementing API handlers or response behavior.
- `api-error-handling` when updating Web error parsing and user-facing surfaces.
- `verification-before-completion` before reporting the contract change complete.

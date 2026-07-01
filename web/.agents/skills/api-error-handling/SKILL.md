---
name: api-error-handling
description: Standardized error format, error code clusters, and API client usage for consistent error handling.
---

# api-error-handling

## Overview

This skill ensures consistent error handling between the frontend and backend. It defines the canonical API error contract and separates backend error codes from frontend-only fallback codes.

## Guidelines

### 1. Error Response Format
All error responses from the backend (BFF or mock) MUST follow this format:

```json
{
  "code": 404,
  "error_code": "COMMON.NOT_FOUND",
  "message": "Human-readable error message",
  "request_id": "req_123"
}
```

`code` is the numeric HTTP status. `error_code` is the canonical machine-readable branch field. Client code must not branch on `message`.

### 2. Error Code Namespaces

Backend `error_code` values use uppercase dot-separated scopes. Use `ApiErrorCode` from `@/http/codes` for mock/BFF responses and server-driven branches.

| Scope | Description | Examples |
|-------|-------------|----------|
| `COMMON.*` | Shared transport, validation, and infrastructure errors | `COMMON.NOT_FOUND`, `COMMON.VALIDATION_FAILED` |
| `AUTH.*` | Authentication and authorization issues | `AUTH.UNAUTHORIZED`, `AUTH.INVALID_CREDENTIALS` |
| `USER.*` | User-domain constraints | `USER.EMAIL_ALREADY_EXISTS` |
| `API_KEY.*` | API key lifecycle errors | `API_KEY.REVOKED` |

Frontend-only fallback values use `ClientErrorCode` and are scoped under `CLIENT.*`. They are only for failures that do not come with a backend response, such as network and timeout failures.

### 3. Usage in Code
- **Constants**: Use `ApiErrorCode` for backend `error_code` values and `ClientErrorCode` for frontend-only fallback values. `src/test/error-code-vocabulary.test.ts` enforces the namespace split.
- **Mock routes**: Return `{ code, error_code, message, request_id? }`; do not return legacy `{ error, code: "VAL_400" }` shapes.
- **Mock BFF guard**: Call `guardMockBffRoute()` from `@/app/api/_shared/mock-bff` before reading request bodies or touching mock state. `src/test/mock-bff-route-contract.test.ts` enforces this for every `src/app/api/**/route.ts` file.
- **Validation**: Return `400 COMMON.INVALID_INPUT` for malformed JSON or transport-level input errors; return `422 COMMON.VALIDATION_FAILED` with `errors` for schema/field validation failures.
- **Auto-handling**: Errors are automatically caught by `HttpClient` and passed to `handleError` (toast notification, etc.).
- **Manual Handling**: Use `skipErrorHandler: true` in the request configuration to handle errors manually within components or hooks.

```typescript
try {
  await request.post('/resource', data, { skipErrorHandler: true });
} catch (error) {
  if (error instanceof ApiError && error.errorCode === ApiErrorCode.COMMON_CONFLICT) {
    // Custom logic
  }
}
```

> [!IMPORTANT]
> **Consistency**: Backend `error_code` is the source of truth. Legacy frontend string codes may be normalized for backward compatibility, but new code must not emit them.

## Related Skills

- [`data-state-management`](../data-state-management/): Error states in queries / mutations.
- [`i18n-handler`](../i18n-handler/): Error message translation.
- [`environment-config`](../environment-config/): What error detail is safe in prod vs dev.
- [`verification-before-completion`](../../../../.agents/skills/verification-before-completion/): Test at least one error path before claiming done.

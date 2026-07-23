---
name: api-error-handling
description: Implement Web API error parsing and user-facing mappings. Use for ApiErrorCode, ClientErrorCode, request_id, field errors, or network failure behavior.
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

Client-owned fallback values use `ClientErrorCode` and are scoped under `CLIENT.*`. They describe
failures that are not backend `error_code` values, such as network, timeout, and malformed
successful-response failures (`CLIENT.INVALID_RESPONSE`).

### 3. Usage in Code
- **Constants**: Use `ApiErrorCode` for backend `error_code` values and `ClientErrorCode` for
  client-owned failures. `src/test/error-code-vocabulary.test.ts` enforces the namespace split.
- **Mock routes**: Return `{ code, error_code, message, request_id? }`; do not return legacy `{ error, code: "VAL_400" }` shapes.
- **Mock BFF guards**: Call `guardMockBffRoute()` before reading request bodies or touching mock state. Unsafe handlers must then call `guardSameOriginMutation(request)`. `src/test/mock-bff-route-contract.test.ts` enforces both boundaries.
- **Validation**: Return `400 COMMON.INVALID_INPUT` for malformed JSON or transport-level input errors; return `422 COMMON.VALIDATION_FAILED` with `errors` for schema/field validation failures.
- **Normalization**: `HttpClient` converts transport failures to `ApiError` and never creates UI
  side effects. Query caches, forms, and hooks own presentation.
- **React Query Ownership**: A locally presented query or mutation must also set
  `meta: LOCAL_ERROR_HANDLING_META` so QueryCache or MutationCache does not add a second surface.
- **User Copy**: Choose reviewed local copy from normalized `error_code` and HTTP status. Never
  display `ApiError.message` directly on authentication, authorization, or other sensitive forms.
- **Field Errors**: Use `ApiError.fieldErrors` to identify which controls failed. Render local
  translations for those fields rather than backend-owned strings.

```typescript
try {
  await request.post('/resource', data);
} catch (error) {
  if (error instanceof ApiError && error.errorCode === ApiErrorCode.COMMON_CONFLICT) {
    // Custom logic
  }
}
```

> [!IMPORTANT]
> **Consistency**: Backend `error_code` is the source of truth. Legacy frontend string codes may be normalized for backward compatibility, but new code must not emit them.

### 4. Authentication Resolution

`/auth/me` bypasses React Query because the auth store and `AuthGuard` own its stable recovery UI.
Classify the normalized `ApiError` by status and `error_code`, never by message:

| Evidence | Auth state | Behavior |
|---|---|---|
| `401`, `AUTH.UNAUTHORIZED`, invalid credentials | `unauthenticated` | Redirect to login. |
| `403`, `AUTH.FORBIDDEN`, `AUTH.ACCOUNT_DISABLED` | `forbidden` | Block content without a login loop. |
| `CLIENT.*`, `429`, `5xx`, malformed, unknown | `unavailable` | Keep the session unresolved and offer retry. |

Do not call `reset()` for availability failures. Retry through `initializeAuth()` so concurrent
attempts share one in-flight request. Validate a successful response with `isAuthResponse()`;
malformed `2xx` JSON resolves as `unavailable`, never `authenticated`.

### 5. Authentication Mutations

- `login`, `register`, `me`, and `logout` must request `unknown` payloads and validate successful
  JSON with endpoint guards before resolving the mutation.
- A malformed `2xx` payload becomes `CLIENT.INVALID_RESPONSE`; login and registration must not
  redirect, and logout must not clear local state.
- Login and registration forms own their manual error surface, including an alert, localized field
  feedback, and clearing stale mutation state after edits.
- Authentication mutations disable retries. Do not retry writes without explicit idempotency
  evidence from the endpoint contract.
- Logout is user-idempotent: `401` / `AUTH.UNAUTHORIZED` means the session is already absent, so
  clear local state and navigate to the logged-out route. In `api-session` mode the server adapter
  always removes the HttpOnly credential after attempting remote revocation; a `503` reports that
  the remote outcome is unknown even though browser credential custody is gone. Client-owned modes
  preserve local state for availability failures until their authority can be resolved again.

## Related Skills

Select another skill only when its distinct concern is active.

- [`data-state-management`](../data-state-management/): Error states in queries / mutations.
- [`i18n-handler`](../i18n-handler/): Error message translation.
- [`environment-config`](../environment-config/): What error detail is safe in prod vs dev.

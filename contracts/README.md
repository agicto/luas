# HTTP Contracts

This directory documents the contracts shared by `api/` and `web/`.

## Success Responses

Non-paginated API success responses use:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Paginated responses add `meta` and `links`:

```json
{
  "code": 0,
  "message": "success",
  "data": [],
  "meta": {
    "current_page": 1,
    "per_page": 15,
    "total": 0,
    "last_page": 1,
    "from": 0,
    "to": 0
  },
  "links": {
    "first": "",
    "last": "",
    "prev": null,
    "next": null
  }
}
```

## Error Responses

API errors expose a numeric HTTP `code` and a stable machine-readable `error_code`:

```json
{
  "code": 404,
  "error_code": "COMMON.NOT_FOUND",
  "message": "User not found",
  "error": "record not found",
  "request_id": "req_123"
}
```

Malformed JSON and transport-level input failures use HTTP 400 with `COMMON.INVALID_INPUT`. Schema and field validation errors use HTTP 422 and include field-level `errors`:

```json
{
  "code": 422,
  "error_code": "COMMON.VALIDATION_FAILED",
  "message": "Validation failed",
  "errors": {
    "email": ["email is required"]
  },
  "request_id": "req_123"
}
```

`error_code` is canonical for client behavior. `code` is transport status and must not be used as the only branching signal. The web HTTP client may normalize legacy mock responses that only expose a string `code`, but new API and mock responses must emit `error_code`.

`message`, `error`, and field-message entries inside `errors` are not a stable localized-copy
contract. User interfaces select reviewed local copy from `error_code` and status; they may use
the keys in `errors` to associate failures with controls without rendering backend detail directly.

Development mock BFF routes must preserve the success and error envelope shapes. When the Web mock BFF is disabled in production runtime, it returns HTTP 503 with `COMMON.SERVICE_UNAVAILABLE`.

Common scaffold-level errors:

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 400 | `COMMON.INVALID_INPUT` | Malformed JSON or transport-level input failure |
| 401 | `AUTH.UNAUTHORIZED` | Authentication is missing or invalid |
| 403 | `AUTH.FORBIDDEN` | Caller is authenticated but not allowed |
| 404 | `COMMON.NOT_FOUND` | Resource or constrained route parameter was not found |
| 409 | `COMMON.CONFLICT` | Resource state conflicts with the request |
| 413 | `COMMON.REQUEST_TOO_LARGE` | Request body exceeded the configured body limit |
| 422 | `COMMON.VALIDATION_FAILED` | Schema or field validation failed |
| 429 | `COMMON.RATE_LIMITED` | Rate limit exceeded |
| 500 | `COMMON.INTERNAL` | Unexpected server error |
| 503 | `COMMON.TIMEOUT` | Request exceeded the configured processing timeout |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | A required service or dependency is unavailable |

Rate-limited responses must include `Retry-After` when the reset time is known. Successful responses that pass through a rate limiter may include `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset`.

## Contract Checklist

- Add or update the documented request and response shape before changing both halves.
- Keep JSON fields in `snake_case`.
- Include stable `error_code` values for non-2xx responses.
- Include `request_id` when the API has one in context.
- Add API and Web tests for contract-sensitive changes.
- From the repo root, run `python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py` after changing scaffold-level HTTP status or `error_code` behavior.

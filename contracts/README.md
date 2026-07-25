# HTTP Contracts

This directory documents the contracts shared by `api/` and its browser-shell alternatives under
`web/` and `web-spa/`.

The response envelope, validation classes, `error_code`, and `request_id` semantics are global.
Feature endpoint paths and DTOs are shared only when their owning contract says so. In particular,
the current browser authentication surface and Go API authentication-session surface require an explicit adapter;
see [`AUTHENTICATION.md`](AUTHENTICATION.md).

Optional backend starter contracts are documented independently. The first ownership foundation is
[`ORGANIZATIONS.md`](ORGANIZATIONS.md); API and Next.js Web activation are disabled independently
by default, and each delivered browser operation is listed explicitly rather than implied by the
backend starter. Static SPA support is listed only after its browser adapter exists.

Organization-scoped access roles, exact permission checks, delegated-management safety, and the
replaceable authorization seam are defined in [`PERMISSIONS.md`](PERMISSIONS.md).

User-scoped in-app records, channel preferences, idempotent internal publication, durable email
delivery, and the fixed browser notification center are defined in
[`NOTIFICATIONS.md`](NOTIFICATIONS.md).

User-owned upload intents, immutable object promotion, private download grants, content inspection,
and deletion semantics are defined in [`ASSETS.md`](ASSETS.md).

Code-owned app, organization, and user setting definitions, optimistic concurrency, public ETag
caching, private responses, and reset history are defined in [`SETTINGS.md`](SETTINGS.md).

Trusted code-owned usage events, finite UTC counters, exact idempotency, atomic quota decisions,
private current-period summaries, and retention are defined in [`USAGE.md`](USAGE.md).

Organization-owned outbound endpoints, finite trusted events, Standard Webhooks signing, encrypted
secret rotation, durable retry/replay, and the privacy-minimized delivery ledger are defined in
[`WEBHOOKS.md`](WEBHOOKS.md).

The default API key lifecycle, one-time plaintext rule, fixed browser adapter paths, scope grammar,
and route guard semantics are defined in [`API_KEYS.md`](API_KEYS.md).

The default user-scoped mutation history, privacy-minimized audit metadata, bounded filters, and
explicit retention command are defined in [`AUDIT.md`](AUDIT.md).

## Contract Discovery

These reviewed Markdown contracts own HTTP semantics. The API's runtime route catalog complements
them by showing which method/path pairs are assembled for one resolved configuration:

```bash
cd api
DB_ENABLED=false AI_ENABLED=false go run ./cmd/luas route:list --format=json
```

The catalog is deterministic, schema-versioned, and generated through the same core and starter
route registration seam used by the server. It is not an OpenAPI Description and does not infer
payloads, responses, authentication, authorization, or middleware. See
[`../api/docs/ROUTE_DISCOVERY.md`](../api/docs/ROUTE_DISCOVERY.md) for its ownership and validation
contract.

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

Development mock BFF routes must preserve the success and error envelope shapes plus the
browser-facing contract of the production endpoint or adapter they substitute. When the Web mock
BFF is disabled in production runtime, it returns HTTP 503 with
`COMMON.SERVICE_UNAVAILABLE`.

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

When the Go API is intentionally started with `DB_ENABLED=false`, default starter routes remain
registered. Requests that reach persistence return `503` with `COMMON.SERVICE_UNAVAILABLE`;
authentication, route constraints, or input validation may still reject a request first. Health
liveness remains available while readiness reports the disabled database as down.

Rate-limited responses must include `Retry-After` when the reset time is known. Successful responses that pass through a rate limiter may include `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset`.

## Contract Checklist

- Add or update the documented request and response shape before changing multiple deployable units.
- Keep JSON fields in `snake_case`.
- Include stable `error_code` values for non-2xx responses.
- Include `request_id` when the API has one in context.
- Add API and every changed browser-shell test for contract-sensitive changes.
- Document adapter-owned path or DTO mappings instead of treating unlike endpoints as interchangeable.
- From the repo root, run `python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py` after changing scaffold-level HTTP status or `error_code` behavior.

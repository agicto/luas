# API Key Contract

This contract owns the default `apikey` starter across the Go API, the Web API key feature, the
fixed production API adapter, and the development mock BFF. It defines user-owned key management
and route-level scope attenuation. It does not define product roles, generalized RBAC, usage
metering, quotas, or billing.

## Ownership And Paths

An authenticated user may manage only their own API keys.

| Operation | Browser route | Go API route | Success |
|---|---|---|---|
| List keys | `GET /api/api-keys` | `GET /v1/api-keys` | Paginated envelope |
| Create key | `POST /api/api-keys` | `POST /v1/api-keys` | `201` envelope |
| Revoke key | `DELETE /api/api-keys/:id` | `DELETE /v1/api-keys/:id` | `204` |

The Web production API adapter owns the HttpOnly API session credential and forwards only these
fixed relative paths. The browser never receives the Go JWT. Every browser response is
`Cache-Control: private, no-store` and varies on `Cookie`.

## Create

Request:

```json
{
  "name": "Deployment automation",
  "scopes": ["models:invoke", "models:read"],
  "expires_at": "2027-01-01T00:00:00Z"
}
```

- `name` is trimmed and contains 1 to 100 Unicode code points.
- `scopes` contains at most 32 input values. Each value is normalized to lowercase, duplicates are
  removed, and the result uses `namespace:action`, where each segment starts with a letter and
  contains lowercase ASCII letters, digits, `_`, or `-`. The explicit `*` wildcard is also accepted.
- `expires_at` is optional and must be in the future.

The response returns metadata plus `plaintext_key`. The plaintext appears only in this successful
create response:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "api_key": {
      "id": 42,
      "user_id": 7,
      "name": "Deployment automation",
      "key_prefix": "luas_ab12cd34",
      "scopes": ["models:invoke", "models:read"],
      "created_at": "2026-07-15T12:00:00Z",
      "updated_at": "2026-07-15T12:00:00Z"
    },
    "plaintext_key": "luas_ab12cd34.<secret>"
  }
}
```

The API persists only the SHA-256 hash. List responses and later reads never contain
`plaintext_key` or `key_hash`. The Web shows the plaintext in one dialog, clears it when the dialog
closes, and immediately resets React Query mutation data instead of placing it in query cache.

## List And Revoke

List metadata may contain `last_used_at`, `expires_at`, and `revoked_at`. New scope storage uses a
JSON string array; the API reads legacy comma-separated rows for compatibility but never writes that
format again.

Revocation is idempotent for the owner: repeated deletes return `204` without rewriting the
revocation timestamp. A missing key or a key owned by another user returns `404 API_KEY.NOT_FOUND`,
so ownership is not disclosed. Revocation updates only `revoked_at` and `updated_at`; a concurrent
last-used update cannot clear revocation.

## Authentication And Scopes

API clients send the plaintext key in `X-API-Key` to routes using the starter-owned `api_key`
middleware. Validation rejects missing, unknown, expired, and revoked keys. Successful validation
adds the key owner and scopes to request context.

Scopes attenuate what a key may do; they never elevate the key beyond its owning user. Scope checks
are exact, except that `*` matches every required scope. A route that needs scopes applies
`apikey.RequireScopes(...)` after API key authentication. Missing API key identity returns
`401 AUTH.UNAUTHORIZED`; insufficient scope returns `403 PERMISSION.DENIED`.

`last_used_at` is operational metadata, not an authorization decision. Its write is best-effort,
atomic, and throttled to at most once per minute per active key.

## Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 401 | `AUTH.UNAUTHORIZED` | Browser management session or `X-API-Key` header is missing |
| 401 | `API_KEY.INVALID` | Client key is unknown or malformed |
| 401 | `API_KEY.EXPIRED` | Client key has expired |
| 401 | `API_KEY.REVOKED` | Client key was revoked |
| 403 | `PERMISSION.DENIED` | Authenticated key lacks a required scope |
| 404 | `API_KEY.NOT_FOUND` | Owned management resource was not found |
| 413 | `COMMON.REQUEST_TOO_LARGE` | Browser create body exceeds its limit |
| 422 | `COMMON.VALIDATION_FAILED` | Request fields fail schema validation |
| 422 | `COMMON.INVALID_INPUT` | Direct API input fails semantic scope or expiry validation |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Adapter, mock backend, or persistence is unavailable |

## Deliberate Deferrals

- Rotation is create-new, migrate callers, then revoke-old; there is no rotate endpoint.
- Scope names belong to downstream product resource vocabulary; Luas defines grammar, not a catalog.
- Usage metering, quotas, billing, key-level rate plans, and organization-owned keys remain separate
  starter decisions.
- Organization roles (`owner`, `admin`, `member`) are not API key scopes or generalized RBAC roles.

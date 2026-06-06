# API Keys Contract

Base path: `/v1`

Authenticated endpoints require `Authorization: Bearer <token>`.

## API Key Shape

```json
{
  "id": 1,
  "user_id": 1,
  "name": "CI",
  "key_prefix": "luas_abcd1234",
  "scopes": ["deploy", "read"],
  "last_used_at": "2026-04-01T00:00:00Z",
  "expires_at": "2026-12-01T00:00:00Z",
  "revoked_at": null,
  "created_at": "2026-04-01T00:00:00Z",
  "updated_at": "2026-04-01T00:00:00Z"
}
```

`key_hash` is never returned. `plaintext_key` is returned only once at creation.

## Create API Key

`POST /v1/api-keys`

Request:

```json
{
  "name": "CI",
  "scopes": ["deploy", "read"],
  "expires_at": "2026-12-01T00:00:00Z"
}
```

Success: `201 Created`

```json
{
  "code": 0,
  "message": "created",
  "data": {
    "api_key": {
      "id": 1,
      "user_id": 1,
      "name": "CI",
      "key_prefix": "luas_abcd1234",
      "scopes": ["deploy", "read"],
      "created_at": "2026-04-01T00:00:00Z",
      "updated_at": "2026-04-01T00:00:00Z"
    },
    "plaintext_key": "luas_abcd1234.secret"
  }
}
```

## List API Keys

`GET /v1/api-keys?page=1&page_size=20`

Success: `200 OK`

The response is paginated with `data`, `meta`, and `links`.

## Revoke API Key

`DELETE /v1/api-keys/{id}`

Success: `204 No Content`

Error codes:

- `API_KEY.NOT_FOUND`
- `AUTH.UNAUTHORIZED`

## API Key Authentication

Routes using the `api_key` middleware accept:

`X-API-Key: luas_abcd1234.secret`

Validation errors:

- `API_KEY.INVALID`
- `API_KEY.EXPIRED`
- `API_KEY.REVOKED`
- `PERMISSION.DENIED`

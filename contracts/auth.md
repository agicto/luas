# Auth Contract

Base path: `/v1`

All JSON responses use the Luas envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Errors use stable `error_code` values and may include `request_id`.

## Register

`POST /v1/register`

Request:

```json
{
  "username": "demo",
  "password": "secret123",
  "email": "demo@example.com",
  "nickname": "Demo",
  "phone": "+3530000000"
}
```

Success: `201 Created`

`data` is a user object. `password` is never returned.

Error codes:

- `COMMON.VALIDATION_FAILED`
- `USER.EMAIL_ALREADY_EXISTS`
- `USER.USERNAME_ALREADY_EXISTS`

## Login

`POST /v1/login`

Request:

```json
{
  "username": "demo",
  "password": "secret123"
}
```

Success: `200 OK`

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "jwt...",
    "user": {
      "id": 1,
      "username": "demo",
      "email": "demo@example.com",
      "status": 1,
      "created_at": "2026-04-01T00:00:00Z",
      "updated_at": "2026-04-01T00:00:00Z"
    }
  }
}
```

Error codes:

- `AUTH.INVALID_CREDENTIALS`
- `AUTH.ACCOUNT_DISABLED`

## Request Password Reset

`POST /v1/password/reset`

Request:

```json
{
  "email": "demo@example.com"
}
```

Success: `200 OK`

The response message is intentionally generic so callers cannot enumerate accounts.

## Confirm Password Reset

`POST /v1/password/reset/confirm`

Request:

```json
{
  "token": "reset-token",
  "new_password": "secret123"
}
```

Success: `200 OK`

Error codes:

- `AUTH.PASSWORD_RESET_TOKEN_INVALID`
- `AUTH.PASSWORD_RESET_TOKEN_EXPIRED`

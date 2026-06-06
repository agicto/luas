# Users Contract

Base path: `/v1`

Authenticated endpoints require `Authorization: Bearer <token>` unless an API key middleware explicitly sets `userID`.

## User Shape

```json
{
  "id": 1,
  "username": "demo",
  "email": "demo@example.com",
  "nickname": "Demo",
  "avatar": "https://example.com/avatar.png",
  "phone": "+3530000000",
  "bio": "Short profile",
  "status": 1,
  "last_login": "2026-04-01T00:00:00Z",
  "created_at": "2026-04-01T00:00:00Z",
  "updated_at": "2026-04-01T00:00:00Z"
}
```

`password` is never returned.

## Get Profile

`GET /v1/users/profile`

Success: `200 OK`

`data` is the current user.

Error codes:

- `AUTH.UNAUTHORIZED`
- `USER.NOT_FOUND`

## Update Profile

`PUT /v1/users/profile`

Request:

```json
{
  "nickname": "Demo",
  "avatar": "https://example.com/avatar.png",
  "phone": "+3530000000",
  "bio": "Short profile"
}
```

Success: `200 OK`

`data` is the updated user.

## Change Password

`PUT /v1/users/password`

Request:

```json
{
  "old_password": "secret123",
  "new_password": "new-secret123"
}
```

Success: `200 OK`

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "Password changed successfully"
  }
}
```

Error codes:

- `AUTH.INVALID_CREDENTIALS`
- `COMMON.VALIDATION_FAILED`

## Delete Account

`DELETE /v1/users/account`

Success: `204 No Content`

Error codes:

- `AUTH.UNAUTHORIZED`
- `USER.NOT_FOUND`

# Team Contract

Base path: `/v1/teams`

All endpoints require the authenticated user session/JWT. Results are scoped to teams where the current user is a member.

All JSON responses use the Luas envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Errors expose stable `error_code` values and may include `request_id`.

## Team Shape

```json
{
  "id": 1,
  "name": "Acme Labs",
  "slug": "acme-labs",
  "owner_user_id": 42,
  "plan": "free",
  "status": "active",
  "created_at": "2026-04-01T00:00:00Z",
  "updated_at": "2026-04-01T00:00:00Z"
}
```

## List Teams

`GET /v1/teams?page=1&page_size=20`

Success: `200 OK`

`data` is paginated and contains only teams visible to the current user.

## Get Team

`GET /v1/teams/{id}`

Success: `200 OK`

`data` is a Team object. Teams outside the current user's membership return `TEAM.NOT_FOUND`.

## Create Team

`POST /v1/teams`

Request:

```json
{
  "name": "Acme Labs",
  "slug": "acme-labs"
}
```

`slug` is optional. When omitted, the API derives it from `name`.

Success: `201 Created`

The creating user becomes the owner member. New teams default to `plan: "free"` and `status: "active"`.

## Update Team

`PUT /v1/teams/{id}`

Request:

```json
{
  "name": "Acme Platform",
  "status": "archived"
}
```

Success: `200 OK`

`status` must be `active` or `archived`.

## Delete Team

`DELETE /v1/teams/{id}`

Success: `204 No Content`

The starter implementation soft-deletes the team record.

## Error Codes

- `AUTH.UNAUTHORIZED`
- `COMMON.VALIDATION_FAILED`
- `COMMON.INVALID_INPUT`
- `TEAM.NOT_FOUND`
- `TEAM.SLUG_ALREADY_EXISTS`

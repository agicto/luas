# Access Contract

Base path: `/v1`

All endpoints require the authenticated user session/JWT.

Access is team-scoped RBAC for Luas starter apps. It provides a stable permission catalog plus custom roles inside a team.

## Permission Shape

```json
{
  "key": "roles:manage",
  "label": "Manage roles",
  "category": "access"
}
```

## Role Shape

```json
{
  "id": 1,
  "team_id": 42,
  "name": "Project Admin",
  "slug": "project-admin",
  "description": "Can manage project settings",
  "permissions": ["roles:manage", "teams:read"],
  "system": false,
  "created_at": "2026-04-01T00:00:00Z",
  "updated_at": "2026-04-01T00:00:00Z"
}
```

## List Permissions

`GET /v1/permissions`

Success: `200 OK`

`data` is a static array of available permission keys.

## List Team Roles

`GET /v1/teams/{team_id}/roles?page=1&page_size=20`

Success: `200 OK`

Returns roles for a team where the current user is a member. Teams outside membership return `TEAM.NOT_FOUND`.

## Create Team Role

`POST /v1/teams/{team_id}/roles`

Request:

```json
{
  "name": "Project Admin",
  "slug": "project-admin",
  "description": "Can manage project settings",
  "permissions": ["roles:manage", "teams:read"]
}
```

`slug` is optional and derived from `name` when omitted.

Success: `201 Created`

## Update Team Role

`PUT /v1/teams/{team_id}/roles/{id}`

Request:

```json
{
  "name": "Project Operator",
  "description": "Can operate project settings",
  "permissions": ["teams:read"]
}
```

Success: `200 OK`

System roles cannot be updated and return `PERMISSION.DENIED`.

## Delete Team Role

`DELETE /v1/teams/{team_id}/roles/{id}`

Success: `204 No Content`

System roles cannot be deleted and return `PERMISSION.DENIED`.

## Error Codes

- `AUTH.UNAUTHORIZED`
- `COMMON.VALIDATION_FAILED`
- `COMMON.INVALID_INPUT`
- `TEAM.NOT_FOUND`
- `ACCESS_ROLE.NOT_FOUND`
- `ACCESS_ROLE.SLUG_ALREADY_EXISTS`
- `PERMISSION.DENIED`

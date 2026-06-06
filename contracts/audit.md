# Audit Contract

Base path: `/v1`

Audit logging is enabled through the global `audit` middleware for mutating API requests.

## Audit Log Shape

```json
{
  "id": 1,
  "user_id": 1,
  "actor_type": "user",
  "actor_id": 1,
  "api_key_id": null,
  "action": "create",
  "resource": "api_keys",
  "target_type": "api_key",
  "target_id": "1",
  "result": "success",
  "method": "POST",
  "path": "/v1/api-keys",
  "route_name": "api_keys.store",
  "status_code": 201,
  "request_id": "req_...",
  "ip_address": "127.0.0.1",
  "user_agent": "curl/8.0",
  "changes": {
    "name": {
      "after": "CI"
    }
  },
  "metadata": {},
  "created_at": "2026-04-01T00:00:00Z",
  "updated_at": "2026-04-01T00:00:00Z"
}
```

## List Audit Logs

`GET /v1/audit-logs?page=1&page_size=20`

Optional query filters:

- `action`
- `resource`
- `method`
- `request_id`
- `status_code`

Success: `200 OK`

The response is paginated with `data`, `meta`, and `links`.

Error codes:

- `AUTH.UNAUTHORIZED`
- `COMMON.INVALID_INPUT`

## Business Change Semantics

Modules may enrich audit entries with:

- `action`
- `resource`
- `target_type`
- `target_id`
- `result`
- `changes`
- `metadata`

When no business change is recorded, audit derives `resource` and `action` from route name, method, and path.

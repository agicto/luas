# Audit Contract

The default `audit` starter owns a durable, user-scoped history of reviewed API mutations. It
records transport context plus optional business change metadata without becoming an analytics
event stream, debug log, authorization source, or immutable regulatory archive.

## Ownership

- The audit middleware observes completed non-`GET`/`HEAD`/`OPTIONS` API requests.
- Starters may attach a finite action, resource, target, result, redacted changes, and
  privacy-reviewed metadata through the audit change seam.
- Audit persistence failure is logged and does not rewrite a completed business response.
- The read API returns only records whose `user_id` is the authenticated caller. It does not expose
  organization-wide, other-user, anonymous, API-key-only, or system-operator history.
- Audit records assist investigation; authorization always uses current application state.

## List Current User Audit History

```http
GET /v1/audit-logs?page=1&per_page=15
Authorization: Bearer <opaque-session-or-api-key>
```

Optional exact-match filters are:

| Query | Maximum | Meaning |
|---|---:|---|
| `action` | 120 characters | Reviewed action such as `update` or `delete` |
| `resource` | 180 characters | Reviewed resource such as `users.profile` |
| `method` | 10 characters | Normalized transport or operator method |
| `request_id` | 80 characters | Correlation identifier |
| `status_code` | 100-599 | Exact HTTP result status |

Pagination follows [`README.md`](README.md). Results are ordered newest first by audit identifier.
Each item may contain:

```json
{
  "id": 42,
  "user_id": 7,
  "actor_type": "user",
  "actor_id": 7,
  "action": "update",
  "resource": "users.profile",
  "target_type": "user",
  "target_id": "7",
  "result": "success",
  "method": "PATCH",
  "path": "/v1/users/profile",
  "route_name": "users.profile.update",
  "status_code": 200,
  "request_id": "req_123",
  "ip_address": "203.0.113.10",
  "user_agent": "ExampleBrowser/1.0",
  "changes": {
    "display_name": {
      "before": "Old name",
      "after": "New name"
    }
  },
  "metadata": {
    "source": "profile"
  },
  "created_at": "2026-07-25T12:00:00Z",
  "updated_at": "2026-07-25T12:00:00Z"
}
```

Optional fields are omitted when unavailable. `changes` and `metadata` are redacted again at the
audit service boundary, but producers must still avoid secrets, credentials, raw provider payloads,
setting values, and unnecessary personal data. Browser interfaces must treat their keys as
reviewed display data, never inject them as markup, and avoid rendering raw user-agent detail by
default.

## Retention

Luas does not choose a universal retention period. Downstream legal, security, privacy, and storage
requirements own that decision. Operators delete only an explicit bounded batch:

```bash
luas audit:prune --before=2026-04-01T00:00:00Z --batch=500
```

- `--before` is required, uses RFC3339, and must be in the past.
- Rows with `created_at < before` are eligible; the cutoff itself is not deleted.
- `--batch` defaults to 500 and is limited to 1-10,000.
- Each deletion batch has a 30-second operation deadline and remains interruptible.
- Selection is deterministic by `(created_at, id)` and uses the retention index.
- Concurrent workers skip rows already locked by another retention batch instead of waiting on the
  same candidates.
- Each successful command records a new system audit entry containing only cutoff, batch, and
  deleted count. A failure to write that follow-up record is warned without retrying the completed
  deletion.
- Large retention jobs repeatedly invoke the bounded command and monitor duration, rows removed,
  database load, and audit-record warnings.

## Errors

| HTTP | `error_code` | Meaning |
|---:|---|---|
| 400 | `COMMON.INVALID_INPUT` | A query value is outside its transport bounds |
| 401 | `AUTH.UNAUTHORIZED` | Authentication is missing or invalid |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Audit persistence is unavailable |

Global envelopes and `request_id` behavior follow [`README.md`](README.md).

## Browser Support

The Go API contract is available by default. The Next.js and static SPA shells do not yet claim an
audit-history feature or mock parity. A browser port must use a fixed same-origin adapter, preserve
pagination and private caching, validate this response before rendering, and keep credentials in
the existing HttpOnly session boundary.

## Deliberate Deferrals

- Organization-wide compliance views and delegated auditor roles.
- Export jobs, legal holds, WORM storage, signing, and tamper-evidence claims.
- Full-text search, arbitrary metadata filters, and raw payload capture.
- Automatic retention schedules without a downstream-reviewed policy.

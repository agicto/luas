# Organization Contract

The `organization` optional starter establishes the tenant/account ownership boundary for
downstream applications. In Luas vocabulary, an organization is not a synonym for a workspace:
a product may add workspaces beneath an organization later, but must not silently alternate the
two terms in routes, tables, DTOs, or authorization checks.

## Activation

The starter is disabled by default. Enable it additively for every API process and migration job:

```dotenv
OPTIONAL_STARTERS=organization
```

The default `audit`, `apikey`, and `user` starters are always active and must not be listed in
`OPTIONAL_STARTERS`. Unknown, duplicate, or default starter names are configuration errors. All
replicas, pre-deploy migration jobs, and one-off seeder jobs must use the same value so route and
schema activation cannot drift.

## Ownership Kernel

All endpoints require the standard Go API bearer token.

| Operation | Endpoint | Request | Successful `data` |
|---|---|---|---|
| Create | `POST /v1/organizations` | `{ name, slug? }` | Organization membership view |
| List mine | `GET /v1/organizations` | Pagination query | Paginated organization membership views |
| Get mine | `GET /v1/organizations/:id` | none | Organization membership view |
| Rename | `PATCH /v1/organizations/:id` | `{ name }` | Organization membership view |

An organization membership view contains:

```json
{
  "id": 42,
  "name": "Acme Europe",
  "slug": "acme-europe",
  "role": "owner",
  "created_at": "2026-07-14T20:00:00Z",
  "updated_at": "2026-07-14T20:00:00Z"
}
```

- `name` is trimmed and must contain 2-100 Unicode characters.
- `slug` is immutable, globally unique, 3-50 characters, and must match
  `^[a-z0-9][a-z0-9-]*[a-z0-9]$`. If omitted, the API generates an opaque `org-...` slug.
- Creating an organization and its `owner` membership is one database transaction.
- Roles are organization-scoped values: `owner`, `admin`, and `member`. They are not global RBAC
  roles. This kernel creates only the owner membership; invitations and membership mutation are a
  later contract.
- Listing and lookup are always scoped by the authenticated user's membership. A non-member gets
  the same `404 ORGANIZATION.NOT_FOUND` response as an absent organization, preventing existence
  disclosure.
- `owner` and `admin` may rename an organization. A known member without that capability gets
  `403 PERMISSION.DENIED`.
- An account that owns one or more organizations cannot self-delete. Until ownership transfer is
  added, deletion returns `409 ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED` instead of orphaning a
  tenant.

## Stable Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 404 | `ORGANIZATION.NOT_FOUND` | The caller has no visible membership for the organization |
| 409 | `ORGANIZATION.SLUG_ALREADY_EXISTS` | The requested immutable slug is already allocated |
| 409 | `ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED` | Account deletion would leave an owned organization without an owner |
| 403 | `PERMISSION.DENIED` | The caller is a member but the role cannot perform the mutation |
| 422 | `COMMON.VALIDATION_FAILED` | A request field fails the documented shape |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Organization persistence is unavailable |

All failures use the global error envelope and `request_id` rules in [`README.md`](README.md).

## Deliberate Deferrals

This is a backend ownership kernel, not yet a complete business-ready organization starter.
Invitations, membership lifecycle, ownership transfer, organization deletion, active organization
context, permission policies, Web UI, and mock BFF parity remain explicit follow-up work. The
starter must not be marked ready in the starter roadmap until those surfaces and extraction rules
exist.

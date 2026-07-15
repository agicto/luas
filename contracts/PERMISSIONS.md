# Permission Contract

The optional `permission` starter provides organization-scoped role-based access control for
downstream applications. It depends on the optional `organization` starter and deliberately keeps
two vocabularies separate:

- organization membership roles (`owner`, `admin`, `member`) govern membership and ownership
  lifecycle;
- access roles group granular permission keys used by product policy checks.

An access role never changes organization ownership, invitation authority, or member lifecycle
rules. Existing organization policies remain authoritative for those operations.

## Activation

Enable both dependent starters for every API process and migration job:

```dotenv
OPTIONAL_STARTERS=organization,permission
```

Enable the matching browser features in the same Web build:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,permission
```

Selecting `permission` without `organization` is a startup/build configuration error. Starter
selection is additive, restart-scoped, and identical across API replicas, migration jobs, and
seed jobs.

## Policy Model

Permission keys are stable lowercase dotted identifiers such as `projects.read`. Each segment
starts with a letter and may contain lowercase letters, digits, or underscores. Keys contain at
least two segments and at most 100 bytes.

The starter uses these rules:

- allow-only and default-deny;
- exact permission matching, with no wildcards or implicit prefix matching;
- permissions are granted through access roles, never directly to users;
- access role assignments belong to an organization membership, not a global user;
- the current organization owner may perform every registered permission check;
- non-owner managers may only create, update, delete, or assign role sets whose permissions are a
  subset of their own current effective permissions;
- replacing assignments is atomic and validates both the existing and requested role sets, so a
  delegated manager cannot grant or revoke privileges above their own authority;
- unknown permission keys fail closed and cannot be persisted.

The built-in catalog contains:

| Permission | Purpose |
|---|---|
| `permission.roles.read` | Read the catalog and access roles |
| `permission.roles.manage` | Create, update, and delete access roles |
| `permission.assignments.read` | Read member access-role assignments |
| `permission.assignments.manage` | Replace member access-role assignments |

Downstream modules extend the code-owned catalog at application assembly time. The catalog is not
runtime administrator input: a deployment cannot persist an arbitrary permission string that no
reviewed policy check owns.

## HTTP API

All endpoints require the standard bearer token and a verified `Organization-Id` header. They run
after the organization-context middleware and consume its typed membership; handlers never trust
the raw header directly.

| Operation | Endpoint | Required permission | Successful `data` |
|---|---|---|---|
| Effective context | `GET /v1/permission-context` | Current membership only | Permission context |
| Permission catalog | `GET /v1/permissions` | `permission.roles.read` | `{ permissions }` |
| List access roles | `GET /v1/access-roles` | `permission.roles.read` | Paginated access roles |
| Get access role | `GET /v1/access-roles/:id` | `permission.roles.read` | Access role |
| Create access role | `POST /v1/access-roles` | `permission.roles.manage` | Access role |
| Update access role | `PATCH /v1/access-roles/:id` | `permission.roles.manage` | Access role |
| Delete access role | `DELETE /v1/access-roles/:id` | `permission.roles.manage` | `204 No Content` |
| Read assignments | `GET /v1/organization-members/:member_id/access-roles` | `permission.assignments.read` | Assignment view |
| Replace assignments | `PUT /v1/organization-members/:member_id/access-roles` | `permission.assignments.manage` | Assignment view |

A permission context is computed from current persistence on every request:

```json
{
  "organization_id": 42,
  "membership_id": 91,
  "is_owner": false,
  "access_role_ids": [7],
  "permissions": ["projects.read"]
}
```

Owner contexts return every registered catalog key and `is_owner: true`. Arrays are sorted and
deduplicated. Permission checks do not trust credential claims or browser state, so revocation takes effect
without token refresh.

An access role contains its immutable organization-scoped slug and complete permission set:

```json
{
  "id": 7,
  "organization_id": 42,
  "name": "Project Viewer",
  "slug": "project-viewer",
  "permissions": ["projects.read"],
  "created_at": "2026-07-15T20:00:00Z",
  "updated_at": "2026-07-15T20:00:00Z"
}
```

Create accepts `{ name, slug, permissions }`. Update accepts `{ name, permissions }`; the slug is
immutable. Names contain 2-100 Unicode characters. Slugs contain 3-50 lowercase letters, digits,
or hyphens, cannot begin or end with a hyphen, and are unique within one organization. A role may
temporarily contain no permissions. Requests contain at most 100 unique permission keys.

An assignment view is deliberately relationship-only:

```json
{
  "member_id": 91,
  "access_role_ids": [7, 8]
}
```

`PUT` accepts the same `access_role_ids` field and replaces the complete set in one transaction.
Every selected role and the target membership must belong to the active organization. An empty
array removes all access roles. Requests contain at most 100 unique role IDs.

## Guard Seam

The module binds a request-aware authorizer for downstream services and exposes a Gin guard factory
for routes. Callers pass the typed `domain.OrganizationContext` and one registered permission key.
The check refreshes membership state and exact grants from persistence; missing context, removed
membership, unknown permission, or unavailable persistence fails closed.

Browser authorization is presentation logic only. The Web feature may hide controls using the
permission context, but the Go service remains the authority for every read and mutation.

## Browser Adapter

The fixed same-origin adapter mirrors the API resources under `/api` and forwards the selected
organization ID explicitly:

| Browser endpoint | Upstream endpoint |
|---|---|
| `GET /api/permission-context` | `GET /v1/permission-context` |
| `GET /api/permissions` | `GET /v1/permissions` |
| `GET/POST /api/access-roles` | `GET/POST /v1/access-roles` |
| `GET/PATCH/DELETE /api/access-roles/:id` | matching `/v1/access-roles/:id` |
| `GET/PUT /api/organization-members/:member_id/access-roles` | matching `/v1/organization-members/:member_id/access-roles` |

Every browser request supplies `Organization-Id`; it is never inferred from a cookie or global
module variable. Unsafe routes enforce same-origin checks and bounded JSON bodies. Successful
payloads are strictly validated before entering the query cache. Development mock state implements
the same owner bypass, subset checks, response envelopes, and error codes only when both optional
Web features and the mock BFF are enabled.

## Audit And Performance

Role and assignment mutations emit audit records containing organization, role, and membership IDs
plus permission-key or role-ID changes. They do not copy user profile fields. Effective checks use
the active membership and indexed assignment/grant joins; the starter deliberately ships without a
cross-request permission cache so revocation semantics do not depend on invalidation correctness.

## Stable Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 403 | `PERMISSION.DENIED` | The caller lacks the required effective permission or dominance |
| 404 | `PERMISSION.ROLE_NOT_FOUND` | The access role is not visible in the active organization |
| 409 | `PERMISSION.ROLE_SLUG_ALREADY_EXISTS` | The organization already has this role slug |
| 422 | `PERMISSION.UNKNOWN` | A requested permission key is not in the code-owned catalog |
| 404 | `ORGANIZATION.MEMBER_NOT_FOUND` | The target member is not in the active organization |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Permission persistence or a configured guard is unavailable |

Organization context errors, validation failures, global envelopes, and `request_id` behavior follow
[`ORGANIZATIONS.md`](ORGANIZATIONS.md) and [`README.md`](README.md).

## Deliberate Deferrals

The starter does not implement explicit deny rules, wildcard grants, role inheritance, direct user
permissions, resource-instance ownership, attribute/condition languages, approval workflows, or an
external policy engine. Business modules keep resource-specific ownership checks in their own
policies and may replace the authorizer behind its interface when those requirements become real.

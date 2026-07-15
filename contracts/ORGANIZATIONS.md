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
  roles. Direct creation grants only the owner membership; the invitation lifecycle may add
  `admin` or `member` memberships.
- Listing and lookup are always scoped by the authenticated user's membership. A non-member gets
  the same `404 ORGANIZATION.NOT_FOUND` response as an absent organization, preventing existence
  disclosure.
- `owner` and `admin` may rename an organization. A known member without that capability gets
  `403 PERMISSION.DENIED`.
- An account that owns one or more organizations cannot self-delete. Deletion returns
  `409 ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED` instead of orphaning a tenant. A non-owner must
  leave every remaining organization before account deletion; otherwise the API returns
  `409 ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED` instead of retaining memberships for a soft-deleted
  user.

## Active Organization Context

Tenant-scoped product routes use an explicit, request-scoped organization selection. They require
exactly one `Organization-Id` request header whose value is the canonical positive decimal
organization ID. Header names are case-insensitive; values containing signs, leading zeroes,
commas, multiple field lines, whitespace-only content, or values outside the platform `uint` range
are invalid.

The optional starter exposes one verification endpoint:

| Operation | Endpoint | Request | Successful `data` |
|---|---|---|---|
| Resolve active context | `GET /v1/organization-context` | `Organization-Id` header | Organization context view |

```json
{
  "organization_id": 42,
  "organization_name": "Acme Europe",
  "organization_slug": "acme-europe",
  "membership_id": 91,
  "user_id": 17,
  "role": "admin"
}
```

- Authentication establishes the user; the header only selects among that user's current
  memberships. A syntactically valid ID for an absent organization or non-member returns the same
  `404 ORGANIZATION.NOT_FOUND` response, preventing existence disclosure.
- Resolution is performed for every protected request through the existing
  `(organization_id, user_id)` membership index. Downstream handlers and services consume the
  typed resolved context, never the raw header.
- The response varies on `Organization-Id`. Context-protected routes add that field to `Vary` so a
  compliant cache cannot reuse one organization's representation for another.
- Existing organization-management routes remain explicitly path-scoped. Do not combine a path
  organization ID and the context header unless the route rejects mismatches.
- The API does not persist a current organization and does not put it into Luas JWTs. Browser
  selection belongs to the active tab or URL, and a production adapter forwards the selected ID on
  each request. This avoids cross-tab selection races and stale membership claims.
- Cross-origin browser deployments must include `Organization-Id` in `CORS_ALLOW_HEADERS` when
  exposing context-protected API routes directly.

## Web Browser Integration

The Web organization feature is independently optional. Enable it only when the API organization
starter is active:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization
```

Browser calls remain same-origin under `/api`. In production, the fixed Web API adapter maps those
paths to the corresponding `/v1` operations and supplies the bearer token from its server-only
HttpOnly cookie. The adapter is an allowlist of route handlers, not an arbitrary proxy: browser
cookies, browser authorization headers, and caller-selected upstream paths are never forwarded.

The browser feature owns these fixed same-origin paths:

| Browser operation | Browser endpoint | Upstream operation |
|---|---|---|
| List organizations | `GET /api/organizations` | `GET /v1/organizations` |
| Create organization | `POST /api/organizations` | `POST /v1/organizations` |
| Get organization | `GET /api/organizations/:id` | `GET /v1/organizations/:id` |
| Rename organization | `PATCH /api/organizations/:id` | `PATCH /v1/organizations/:id` |
| Verify active context | `GET /api/organization-context` plus `Organization-Id` | `GET /v1/organization-context` plus `Organization-Id` |
| List members | `GET /api/organizations/:id/members` | `GET /v1/organizations/:id/members` |
| Change member role | `PATCH /api/organizations/:id/members/:member_id` | `PATCH /v1/organizations/:id/members/:member_id` |
| Remove member or leave | `DELETE /api/organizations/:id/members/:member_id` | `DELETE /v1/organizations/:id/members/:member_id` |
| Transfer ownership | `POST /api/organizations/:id/ownership-transfer` | `POST /v1/organizations/:id/ownership-transfer` |
| List invitations | `GET /api/organizations/:id/invitations` | `GET /v1/organizations/:id/invitations` |
| Create invitation | `POST /api/organizations/:id/invitations` | `POST /v1/organizations/:id/invitations` |
| Revoke invitation | `DELETE /api/organizations/:id/invitations/:invitation_id` | `DELETE /v1/organizations/:id/invitations/:invitation_id` |
| Accept invitation | `POST /api/organization-invitations/accept` | `POST /v1/organization-invitations/accept` |

- The selected organization is represented by `/console/organizations/:id`. It is derived from the
  current URL and is not written to a global cookie, `localStorage`, or a module-level store.
- The browser service validates every successful organization, member, invitation, and ownership
  payload before caching it. Public member and invitation objects are strict: unexpected fields
  such as member email or invitation token produce client-owned `CLIENT.INVALID_RESPONSE` instead
  of entering the UI cache.
- Paginated calls explicitly preserve and validate the global `meta` and `links` envelope fields;
  the default Web response interceptor continues to extract `data` for non-paginated callers.
- Unsafe same-origin routes reject cross-origin requests before reading the body. Both incoming
  JSON and upstream JSON have bounded byte budgets. Adapter errors preserve stable status,
  `error_code`, field ownership, `request_id`, rate-limit headers, and `Vary`, while replacing
  upstream display text with adapter-owned generic text.
- Development mock routes implement the same browser envelope and authorization shape only when
  the mock BFF and organization Web feature are enabled. They are replaceable development state,
  not production persistence.
- Invitation acceptance reads a manually entered bearer token from a password input and sends it
  only in the same-origin POST body. The Web feature never stores the token, adds it to a URL, or
  renders it from invitation-management responses.

## Member Lifecycle

Member endpoints require the standard Go API bearer token. The public `member` resource represents
one internal organization membership; `member_id` is the membership ID, not the user's global ID.

| Operation | Endpoint | Request | Successful `data` |
|---|---|---|---|
| List members | `GET /v1/organizations/:id/members` | Pagination query | Paginated member views |
| Change role | `PATCH /v1/organizations/:id/members/:member_id` | `{ role }` | Member view |
| Remove or leave | `DELETE /v1/organizations/:id/members/:member_id` | none | `204 No Content` |
| Transfer ownership | `POST /v1/organizations/:id/ownership-transfer` | `{ new_owner_member_id }` | `{ previous_owner, new_owner }` |

A member view contains the relationship ID and a deliberately small public identity projection:

```json
{
  "id": 91,
  "user_id": 17,
  "username": "alex",
  "nickname": "Alex",
  "avatar": "https://cdn.example.com/alex.png",
  "role": "member",
  "joined_at": "2026-07-15T20:00:00Z",
  "updated_at": "2026-07-15T20:00:00Z"
}
```

- Every current member may list the member directory. Email, phone, account status, and other user
  profile fields are not part of the member view. A non-member receives the same
  `404 ORGANIZATION.NOT_FOUND` response as an absent organization.
- Only the current `owner` may change another member between `admin` and `member`. Ownership is
  never granted or removed through the role endpoint; attempting to demote the owner returns
  `409 ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED`.
- The `owner` may remove an `admin` or `member`. An `admin` may remove a `member`, but not another
  admin or the owner. Any non-owner may delete their own member resource to leave. The owner must
  transfer ownership before leaving.
- Ownership transfer accepts an existing `admin` or `member` in the same organization. It is one
  database transaction: the target becomes `owner` and the previous owner becomes `admin`. The
  transaction locks the persisted memberships so concurrent requests cannot create two owners or
  leave the organization ownerless.
- Account deletion locks the undeleted user row, evaluates every active starter deletion guard, and
  performs the soft delete in one transaction. Organization creation and invitation acceptance
  acquire the same undeleted-user row lock before writing a membership. Therefore concurrent
  deletion and membership creation may let either business action win, but may not leave a
  membership pointing at a soft-deleted user.
- Member role changes, removals, leaves, and ownership transfers emit audit changes using member,
  organization, and user IDs plus roles. Member profile fields are not copied into audit metadata.

## Invitation Lifecycle

Invitation endpoints also require the standard Go API bearer token. Invitation management is
organization-scoped; acceptance is top-level because the opaque token identifies the organization.

| Operation | Endpoint | Request | Successful `data` |
|---|---|---|---|
| Invite | `POST /v1/organizations/:id/invitations` | `{ email, role }` | `{ invitation, email_send_status }` |
| List invitations | `GET /v1/organizations/:id/invitations` | Pagination query | Paginated invitation views |
| Revoke | `DELETE /v1/organizations/:id/invitations/:invitation_id` | none | `204 No Content` |
| Accept | `POST /v1/organization-invitations/accept` | `{ token }` | Organization membership view |

An invitation view contains no plaintext token:

```json
{
  "id": 73,
  "organization_id": 42,
  "email": "member@example.com",
  "role": "member",
  "status": "pending",
  "expires_at": "2026-07-22T20:00:00Z",
  "created_at": "2026-07-15T20:00:00Z",
  "updated_at": "2026-07-15T20:00:00Z"
}
```

- `owner` and `admin` may invite, list, and revoke. A non-member gets
  `404 ORGANIZATION.NOT_FOUND`; a known `member` gets `403 PERMISSION.DENIED`.
- Invitation roles are only `admin` and `member`. Ownership is never granted through an
  invitation.
- Invitation email is trimmed and lower-cased. An existing member or a second unexpired pending
  invitation for the same organization and email is rejected. Expired invitations remain immutable
  history while releasing the active-invitation uniqueness slot for a replacement.
- Invitation tokens are high-entropy bearer secrets. Luas returns them only to the organization
  mail adapter in process, stores only their SHA-256 hash, and never places them in URLs, API
  responses, audit metadata, or logs.
- `ORGANIZATION_INVITATION_TTL` defaults to `168h` (7 days) and must be positive. A token is
  one-time use and must belong to the authenticated user's email, compared case-insensitively.
- Accepting an invitation and creating the membership is one database transaction. Revoked,
  accepted, unknown, or malformed tokens share the invalid-token branch; expiry and wrong-account
  failures remain separately actionable.
- Invitation persistence commits before email is attempted. The invite call still returns `201`
  when email is unavailable or rejected so retrying the HTTP request cannot accidentally create a
  second business record. `email_send_status` is one of `accepted_by_provider`, `failed`, or
  `not_configured`; it describes only this synchronous send attempt and is not a delivery receipt.
- Invitation create, revoke, and accept transitions emit audit changes using internal invitation,
  organization, and user IDs. Email addresses and tokens are not audit metadata.

## Stable Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 400 | `ORGANIZATION.CONTEXT_REQUIRED` | A context-protected route received no organization selection |
| 400 | `ORGANIZATION.CONTEXT_INVALID` | The organization selection is malformed or ambiguous |
| 404 | `ORGANIZATION.NOT_FOUND` | The caller has no visible membership for the organization |
| 404 | `ORGANIZATION.INVITATION.NOT_FOUND` | The invitation is not visible in the managed organization |
| 404 | `ORGANIZATION.INVITATION.INVALID` | The acceptance token is malformed, unknown, revoked, or already consumed |
| 410 | `ORGANIZATION.INVITATION.EXPIRED` | The acceptance token is valid but past its configured lifetime |
| 409 | `ORGANIZATION.SLUG_ALREADY_EXISTS` | The requested immutable slug is already allocated |
| 409 | `ORGANIZATION.INVITATION.ALREADY_PENDING` | An unexpired invitation already exists for this email |
| 409 | `ORGANIZATION.MEMBER_ALREADY_EXISTS` | The invited email already belongs to an organization member |
| 409 | `ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED` | Account deletion would leave an owned organization without an owner |
| 409 | `ORGANIZATION.OWNERSHIP_TRANSFER_TARGET_INVALID` | Ownership transfer targets the current owner or another invalid member state |
| 409 | `ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED` | Account deletion would retain one or more non-owner memberships |
| 404 | `ORGANIZATION.MEMBER_NOT_FOUND` | The member resource does not exist in the visible organization |
| 403 | `ORGANIZATION.INVITATION.EMAIL_MISMATCH` | The token belongs to a different account email |
| 403 | `PERMISSION.DENIED` | The caller is a member but the role cannot perform the mutation |
| 422 | `COMMON.VALIDATION_FAILED` | A request field fails the documented shape |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Organization persistence is unavailable |

All failures use the global error envelope and `request_id` rules in [`README.md`](README.md).

## Deliberate Deferrals

The optional starter now owns the reusable organization lifecycle across API, fixed browser
adapter, development mock, UI, contracts, tests, and downstream extraction guidance. It remains
optional and must be enabled in both deployable halves.

Organization deletion, durable invitation delivery retries, generalized permission policies, and
arbitrary product resources remain deliberate follow-up work. Those concerns require separate
domain decisions and must not be inferred from the organization-scoped `owner`, `admin`, and
`member` lifecycle delivered here.

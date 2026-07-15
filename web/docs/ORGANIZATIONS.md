# Organization Web Feature

The optional Web `organization` feature is the browser workflow for the API `organization` starter.
It consumes the contract in [`../../contracts/ORGANIZATIONS.md`](../../contracts/ORGANIZATIONS.md);
it does not redefine tenant ownership or authorization in React.

## Activation

Activate both deployable halves explicitly:

```dotenv
# API
OPTIONAL_STARTERS=organization

# Web
NEXT_PUBLIC_OPTIONAL_FEATURES=organization
```

For the production same-origin adapter, also configure:

```dotenv
NEXT_PUBLIC_API_URL=/api
API_ADAPTER_ENABLED=true
API_UPSTREAM_URL=http://api:8025/v1
API_UPSTREAM_TIMEOUT_MS=5000
API_UPSTREAM_MAX_RESPONSE_BYTES=1048576
API_CLIENT_IP_HEADER=x-real-ip
```

The Web feature and API starter are independently deployable settings. Release configuration must
enable or disable them together. A disabled Web feature returns `404` for its console pages and
`503 COMMON.SERVICE_UNAVAILABLE` from its dormant browser Route Handlers; it performs no list query
from the console shell.

## Delivered Workflow

- `/console/organizations` lists memberships and creates organizations.
- `/console/organizations/:id` is the selected organization URL and resolves the canonical
  `Organization-Id` context before rendering settings.
- The console switcher derives selection from that URL. It does not use `localStorage`, a global
  Zustand store, a cookie, or a persisted current-organization field.
- Owners and administrators can rename. Members receive a read-only profile; the Go API remains the
  authorization boundary for every write.
- Every member can view the PII-minimized member directory. Owners can change `admin`/`member`
  roles and transfer ownership; owners and administrators can remove roles permitted by the API,
  while non-owners can leave through their own member resource.
- Owners and administrators can create, list, and revoke invitations. Authenticated users accept a
  token through a password input and same-origin POST body; the token is never put in a URL,
  browser storage, invitation response, or rendered history.
- Successful payloads are validated before entering TanStack Query. Paginated organization,
  member, and invitation lists preserve `meta` and `links` through the explicit `getEnvelope()`
  HTTP mode. Strict member and invitation objects reject accidental PII or token fields.

## Adapter Boundary

`src/server/api-adapter/` owns the shared server-only transport. It accepts only checked-in fixed
relative paths, supplies the bearer token from the HttpOnly API cookie, validates one trusted client
IP, applies a timeout, bounds upstream response bytes, and forwards only reviewed response headers.
Browser cookies, browser authorization, arbitrary paths, `Set-Cookie`, and arbitrary request headers
never cross the boundary.

`src/app/api/organizations/**`, `src/app/api/organization-context/`, and
`src/app/api/organization-invitations/accept/` are explicit allowlist Route Handlers. Unsafe
operations reject cross-origin requests before authentication or bounded JSON reads. The adapter
preserves stable status, `error_code`, field ownership, `request_id`, rate-limit headers, and
`Vary`, but does not render upstream copy. Every organization Route Handler also applies
`Cache-Control: private, no-store` and merges `Vary: Cookie`; context responses retain
`Organization-Id` as an additional cache dimension.

## Development Mock

Outside production, or with an explicit demo deployment, the same routes use an in-memory mock
store after verifying the mock session. It starts with one `Luas Demo` organization and seeded
owner, administrator, and member roles. It implements organization, context, member, ownership,
invitation, and acceptance transitions with the production role matrix. It is development state
only and resets with the Web process.

The mock and production route shapes are tested together. Invitation token generation is injectable
only at the store test seam; the singleton route store never returns or logs bearer tokens. Local
acceptance therefore still requires a token delivered out of band, matching the production trust
boundary instead of adding a mock-only secret response.

## Downstream Replacement

- Keep the feature when the downstream product uses Luas organizations as its tenant/account owner.
- Replace only `src/server/api-adapter/` and the fixed Route Handlers when another backend owns the
  same browser contract.
- Delete the route group, feature folder, i18n module, and `organization` feature selection together
  when the product is single-user or uses another tenant concept.
- Do not rename organization to workspace unless the downstream domain deliberately introduces a
  distinct workspace concept and changes its contracts, routes, persistence, and authorization.

## Deliberate Deferrals

Organization deletion, durable invitation delivery retries, generalized permission policies, and
arbitrary product resource routes remain deferred by the API contract. They belong to separate
business decisions or starters; the organization feature does not pretend that its three scoped
roles are a general RBAC system.

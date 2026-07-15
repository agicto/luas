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
- Successful payloads are validated before entering TanStack Query. Paginated organization lists
  preserve `meta` and `links` through the explicit `getEnvelope()` HTTP mode.

## Adapter Boundary

`src/server/api-adapter/` owns the shared server-only transport. It accepts only checked-in fixed
relative paths, supplies the bearer token from the HttpOnly API cookie, validates one trusted client
IP, applies a timeout, bounds upstream response bytes, and forwards only reviewed response headers.
Browser cookies, browser authorization, arbitrary paths, `Set-Cookie`, and arbitrary request headers
never cross the boundary.

`src/app/api/organizations/**` and `src/app/api/organization-context/` are explicit allowlist Route
Handlers. Unsafe operations reject cross-origin requests before reading bounded JSON. The adapter
preserves stable status, `error_code`, field ownership, `request_id`, rate-limit headers, and `Vary`,
but does not render upstream copy. Every organization Route Handler also applies
`Cache-Control: private, no-store` and merges `Vary: Cookie`; context responses retain
`Organization-Id` as an additional cache dimension.

## Development Mock

Outside production, or with an explicit demo deployment, the same routes use an in-memory mock
store after verifying the mock session. It starts with one `Luas Demo` owner organization and
supports list, create, get, rename, and context resolution. It is development state only and resets
with the Web process.

The mock and production route shapes are tested together. Do not add mock-only response fields or
return invitation bearer tokens through future mock routes.

## Downstream Replacement

- Keep the feature when the downstream product uses Luas organizations as its tenant/account owner.
- Replace only `src/server/api-adapter/` and the fixed Route Handlers when another backend owns the
  same browser contract.
- Delete the route group, feature folder, i18n module, and `organization` feature selection together
  when the product is single-user or uses another tenant concept.
- Do not rename organization to workspace unless the downstream domain deliberately introduces a
  distinct workspace concept and changes its contracts, routes, persistence, and authorization.

## Deliberate Deferrals

Member administration, invitations, ownership transfer, and invitation acceptance remain the next
browser slice. Organization deletion and arbitrary product resource routes remain deferred by the
API contract. The starter stays `Foundation only` until those browser workflows and production
integration checks are complete.

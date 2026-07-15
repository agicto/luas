# Permission Web Feature

The optional Web `permission` feature manages the API permission starter without becoming an
authorization authority. It consumes
[`../../contracts/PERMISSIONS.md`](../../contracts/PERMISSIONS.md) and depends on the organization
browser feature.

## Activate

```dotenv
# API replicas and migration jobs
OPTIONAL_STARTERS=organization,permission

# One Web build
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,permission
```

Selecting the Web permission feature without `organization` fails environment validation. Configure
the production API adapter as described in [`ORGANIZATIONS.md`](ORGANIZATIONS.md); development may
use the guarded mock BFF.

## Delivered Workflow

The organization detail page lazy-loads a Permissions tab when the feature is selected. It shows:

- current effective permission state;
- access-role name, immutable slug, and exact permission grants;
- create, edit, and delete controls when the caller can manage roles;
- organization members and on-demand role assignments when the caller can read assignments;
- atomic complete-set replacement when the caller can manage assignments.

The UI uses effective permissions to avoid presenting unavailable controls. This is presentation
logic only: every read and mutation is independently authorized by the Go service against current
persistence.

## Fixed Adapter Boundary

`src/app/api/permission-context`, `permissions`, `access-roles/**`, and
`organization-members/[memberId]/access-roles` are fixed same-origin routes. Every call explicitly
supplies `Organization-Id` from the organization URL; no cookie, local storage, Zustand store, or
module variable holds a global current organization.

Unsafe operations reject cross-origin requests before reading authentication state or bounded JSON.
The production adapter forwards only the fixed upstream method and path. Responses are private and
no-store, vary by cookie and organization where relevant, and preserve safe upstream status,
`error_code`, request ID, and rate-limit headers.

The browser service strictly validates permission keys, unique IDs, immutable slugs, timestamps,
pagination, and unexpected fields before writing TanStack Query state. Invalid success payloads
become `CLIENT.INVALID_RESPONSE`.

## Development Mock

The mock store implements owner bypass, exact matching, catalog rejection, organization-scoped
roles, delegated subset checks, assignment dominance, cascade removal, and the same stable errors.
It is process-local demonstration state and never substitutes for production persistence.

## Remove Or Replace

- Keep both optional Web features when Luas organizations and access roles match the product domain.
- Replace the fixed Route Handlers only when another backend preserves the same browser contract.
- Delete `src/features/permission`, its `/api` routes, i18n module, feature selection, tests, and
  organization tab together when the product uses another authorization system.
- Never keep the UI while disabling API enforcement, and never treat hidden buttons as policy.

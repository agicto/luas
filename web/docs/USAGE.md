# Usage Feature

The optional Web `usage` feature exposes read-only current-period summaries for the authenticated
user and, when selected, one managed organization. Enable it with its organization dependency:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,usage
```

The API must use the matching selection:

```dotenv
OPTIONAL_STARTERS=organization,usage
```

The canonical behavior is defined in [`../../contracts/USAGE.md`](../../contracts/USAGE.md).

## Browser Routes

| Browser route | Go route | Scope |
|---|---|---|
| `GET /api/usage/user` | `GET /v1/usage/user` | Authenticated user |
| `GET /api/organization-usage` | `GET /v1/organization-usage` | Selected owner/admin organization |

Both handlers are fixed allowlist adapters, never catch-all proxies. They return private no-store
responses and preserve the verified `Organization-Id` boundary. The feature has no browser record,
consume, quota mutation, event history, billing, or provider routes.

## Strict Contract

`src/features/usage/schemas.ts` accepts only the five shipped metric identities, fixed scopes and
units, safe integers, UTC period timestamps, and semantically consistent limit/remaining/overage
fields. It also requires one complete finite catalog per scope. Unknown, missing, duplicate, or
mixed-scope metrics fail as `ClientErrorCode.INVALID_RESPONSE`.

When a downstream API catalog changes, update the contract, Go catalog, Web schema, localized metric
labels, mock state, and tests together. Do not replace the schema with `record`, `unknown`, or a
generic metric renderer that silently accepts server drift.

## Backend Resolution

Development can use the bounded server-only mock store. Production uses the existing authenticated
Go adapter and forwards only the two fixed paths above. The mock models finite current summaries and
manager authorization; it is not a browser-writeable metering database.

The console route is lazy and appears only when the feature is enabled. Organization summaries are
part of the organization overview only for owners and admins. Event identity, source, dimensions,
fingerprints, and receipts never enter client state or rendered output.

## Downstream Removal

To remove usage, delete the feature, two Route Handlers, console route/navigation entry, organization
overview tab, i18n namespace, and tests; then remove `usage` from
`NEXT_PUBLIC_OPTIONAL_FEATURES`. Remove the API starter, migration, contract, and deployment
selection in the same downstream change if the backend is also being removed.

## Verification

```bash
pnpm type-check
pnpm lint
pnpm test -- --run src/test/usage-service.test.ts src/test/usage-route.test.ts
pnpm build
cd .. && python3 .agents/skills/luas-framework-review/scripts/check-usage-boundary.py
```

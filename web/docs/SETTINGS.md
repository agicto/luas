# Setting Feature

The optional Web `setting` feature is the browser contract for the API's typed setting starter. It
provides strict public, current-user, and active-organization adapters plus real console controls for
the shipped locale/timezone definitions. It does not provide a generic settings editor.

## Activation

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,setting
```

The Web selector rejects `setting` without `organization`. The API selection must also include both:

```dotenv
OPTIONAL_STARTERS=organization,setting
```

With `API_ADAPTER_ENABLED=true`, Route Handlers use the fixed server-only Go adapter. With the
development mock BFF enabled, the bounded mock preserves scoped state and version semantics. If
neither backend is available, routes return `503 COMMON.SERVICE_UNAVAILABLE`; they never invent
production values.

## Browser Routes

| Browser route | Fixed API operation |
|---|---|
| `GET /api/settings/public` | `GET /v1/settings/public` |
| `GET /api/settings/user` | `GET /v1/settings/user` |
| `PATCH/DELETE /api/settings/user/:key` | same fixed user setting operation |
| `GET /api/organization-settings` | `GET /v1/organization-settings` |
| `PATCH/DELETE /api/organization-settings/:key` | same fixed organization setting operation |

The adapter forwards only explicitly supplied `If-Match`, `If-None-Match`, and active organization
identity. It never forwards browser cookies or authorization upstream. Public cache headers and ETag
are allowlisted; authenticated responses are rewritten to `private, no-store` and vary on `Cookie`,
plus `Organization-Id` for organization routes.

Unsafe mutations run the same-origin guard before authentication and body parsing. Route keys are
fixed unions, mutation bodies are strict scalar schemas, and successful JSON is parsed as `unknown`
against the five shipped definitions before entering React Query. Unknown definitions fail closed;
downstream apps that add browser-visible settings must extend the schema, types, and UI deliberately.

## UI Ownership

`/console/settings` always contains the real API key workflow. When `setting` is selected it also
contains current-user locale and IANA timezone controls. Organization pages add one locale tab;
members can read it and owner/admin roles can mutate it.

The old placeholder controls for company URL, support email, dark mode, SMS, browser push, 2FA, and
session timeout were removed because no durable owner implemented them. Notification channel
preferences remain in the notification starter; security controls require dedicated auth contracts.

Mutations send the current item version. A 412 conflict invalidates and refetches the query; UI copy
never displays backend messages. Reset uses the same expected-version contract and refetches the
resulting default/tombstone version.

## Mock And Removal

The mock store is isolated by authenticated user or numeric organization, has a 1000-subject bound,
keeps monotonic reset history, and emits the same stable setting errors. It is development behavior,
not a production datastore.

To remove the feature, remove `setting` from `NEXT_PUBLIC_OPTIONAL_FEATURES`, then delete its feature
folder, five Route Handler surfaces, `setting` translation namespace, settings/organization panel
mounts, tests, and this guide. Keep API and Web selection aligned. Full downstream extraction rules
remain in [`MOCK_BFF.md`](MOCK_BFF.md) and the root downstream extraction skill.

## Verification

```bash
pnpm vitest run src/test/setting-service.test.ts src/test/setting-route.test.ts
pnpm type-check
pnpm lint
pnpm build
```

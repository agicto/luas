# Webhook Feature

The optional Web `webhook` feature is the organization-manager surface for the API outbound
webhook starter. Enable it with its ownership dependency:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,webhook
```

The API must select the matching starters and provide encryption material:

```dotenv
OPTIONAL_STARTERS=organization,webhook
WEBHOOK_ENCRYPTION_KEY=<at-least-32-random-characters>
```

The canonical protocol, authorization, retry, signing, privacy, and error behavior lives in
[`../../contracts/WEBHOOKS.md`](../../contracts/WEBHOOKS.md).

## Browser Surface

The organization overview lazy-loads the management tab only for an owner or administrator when
`webhook` is selected. The feature supports endpoint create/update/delete, explicit status changes,
one-time secret rotation, fixed test publication, paginated delivery history, and minimized attempt
history. It does not expose arbitrary event publication, replay, payloads, provider diagnostics, or
target invocation.

Every browser handler is a fixed same-origin adapter:

| Browser route | Go route |
|---|---|
| `GET /api/webhook-event-types` | `GET /v1/webhook-event-types` |
| `GET, POST /api/webhook-endpoints` | `GET, POST /v1/webhook-endpoints` |
| `PATCH, DELETE /api/webhook-endpoints/:id` | Matching fixed endpoint route |
| `PUT /api/webhook-endpoints/:id/status` | Matching fixed status route |
| `POST /api/webhook-endpoints/:id/secret-rotations` | Matching fixed rotation route |
| `POST /api/webhook-endpoints/:id/tests` | Matching fixed test route |
| `GET /api/webhook-deliveries` | `GET /v1/webhook-deliveries` |
| `GET /api/webhook-deliveries/:id/attempts` | Matching fixed attempt route |

The server-only adapter forwards only a validated `Organization-Id`, canonical `If-Match`, bounded
`Idempotency-Key`, selected pagination/filter values, and the HttpOnly API bearer credential. It
never accepts an upstream URL or forwards browser cookies/authorization.

## Secrets And State

Successful create and rotation responses are the only places where `signing_secret` exists in
browser data. The component copies it into dialog-local state, resets React Query mutation data
immediately, and destroys the local value when the dialog closes. Endpoint lists validate that only
the non-secret hint and version are present.

Endpoint mutations use `If-Match: "webhook-endpoint-v<version>"`. A conflict invalidates endpoint
and delivery queries before local copy explains that newer state was loaded. Test publication uses a
new canonical idempotency key per explicit click; automatic mutation retry remains disabled.

## Development Mock

The server-only mock store preserves organization manager checks, endpoint ownership, monotonic
versions, one-time secret shape, finite `webhook.test` selection, status transitions, pagination,
and idempotent test commands. It never sends a network request. A mock test is recorded as terminal
`canceled` with local failure identifier `WEBHOOK.MOCK_NOT_DELIVERED` and zero attempts, so the UI
does not imply that production delivery was verified.

Production never silently falls back to this store. Use `API_ADAPTER_ENABLED=true` with the fixed
Go adapter, or replace the browser contract intentionally.

## Downstream Removal

Remove the feature directory, webhook Route Handlers, organization tab and route message scope,
translation module, tests, and `webhook` selection together. If the API starter is also removed,
stop publishers/workers first, decide retained-event policy, remove every API/migration/worker
selection, deploy without the owning runtime, and only then drop webhook tables.

## Verification

```bash
pnpm type-check
pnpm lint
pnpm exec vitest run src/test/webhook-route.test.ts src/test/webhook-ui.test.tsx
pnpm build
cd .. && python3 .agents/skills/luas-framework-review/scripts/check-webhook-boundary.py
```

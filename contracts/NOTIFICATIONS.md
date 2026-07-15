# Notification Contract

The optional `notification` starter owns a user-facing notification center and the reliable
delivery boundary behind it. It depends only on the default `user` and `audit` starters, so
single-user applications can enable it without adopting organizations or generalized permissions.

The starter keeps two concepts separate:

- a **notification** is one immutable application event addressed to one Luas user;
- a **delivery** is that notification's channel-specific execution record.

One notification may therefore have an immediately delivered `in_app` delivery and an independently
retried `email` delivery. Provider-specific messages are implementation details of a delivery, not
new notification records.

## Activation

Enable the API starter for every API process, migration job, and notification worker:

```dotenv
OPTIONAL_STARTERS=notification
```

Enable the matching browser feature in the same Web build:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=notification
```

Selection is additive and restart-scoped. Enabling the Web feature without either the production
adapter or development mock BFF returns `503 COMMON.SERVICE_UNAVAILABLE`; it never falls back to
fabricated production data.

## Publication Model

Downstream API modules publish through `domain.NotificationPublisher`; there is deliberately no
browser or public HTTP endpoint that can create notifications. A publication contains:

| Field | Rule |
|---|---|
| `user_id` | Existing Luas user; delivery routing always resolves the current user record |
| `idempotency_key` | Required, caller-owned, 1-128 ASCII letters, digits, `.`, `_`, `:`, or `-` |
| `kind` | Required lowercase dotted identifier, 3-100 bytes, such as `billing.invoice_paid` |
| `title` | Plain text, 1-160 Unicode characters |
| `body` | Plain text, 1-4,000 Unicode characters |
| `action_url` | Optional same-origin relative path beginning with one `/`, at most 2,048 bytes |
| `channels` | Unique subset of `in_app` and `email` |
| `required_channels` | Unique subset of `channels` that user preferences cannot disable |

`(user_id, idempotency_key)` is unique. Repeating the same publication returns the original
notification and does not create duplicate deliveries. Reusing a key with different immutable
content fails closed as an idempotency conflict.

Unknown channels, malformed keys, absolute or protocol-relative action URLs, and conflicting
replays are rejected before persistence. Titles and bodies are stored as plain text. The email
adapter HTML-escapes them; the Web renders them as text and never as HTML.

## Preference Priority

Every user has effective global channel preferences. Missing rows resolve to:

```json
{
  "in_app_enabled": true,
  "email_enabled": true
}
```

For each publication, channel selection is evaluated in this order:

1. the publication's declared channels are the upper bound;
2. a required channel is always selected;
3. a non-required channel is selected only when the user's current global preference enables it.

Preferences affect future publications only. Enabling a channel does not retroactively create
deliveries; disabling a channel does not delete existing records or cancel an already claimed
delivery. Security-sensitive downstream workflows may declare reviewed required channels.

## Delivery Lifecycle

`in_app` delivery is committed atomically with the notification and begins as `delivered`. An email
delivery begins as `pending` and is processed by `luas notification:work`.

Email delivery states are:

| State | Meaning |
|---|---|
| `pending` | Eligible at `available_at`, including a transient retry |
| `processing` | Owned by one worker lease until `lease_expires_at` |
| `delivered` | Provider accepted the message |
| `failed` | Permanently invalid, unconfigured, or retry budget exhausted |

Workers claim due rows in short database transactions using row locking, a random lease token, and
`SKIP LOCKED` where supported. Provider I/O happens outside the claim transaction. Completion
updates require the same lease token, so an expired worker cannot overwrite a newer worker's result.
Expired `processing` rows are reclaimable.

Transient failures use bounded exponential retry and a maximum attempt count. Persistent failure
data stores only a stable local code such as `EMAIL.NOT_CONFIGURED`, `EMAIL.PROVIDER_REJECTED`, or
`EMAIL.PROVIDER_UNAVAILABLE`; recipient addresses, title/body text, credentials, provider response
bodies, and free-form error strings are never copied into the delivery ledger or logs.

Every email retry uses the stable provider key `notification-email-<delivery_id>`. The first attempt
binds a hash of the current recipient route; if that route changes before a retry, the delivery fails
closed instead of sending the same provider operation to a different address.

The shipped worker uses the database ledger as its durable source of truth. It does not depend on
the process-local memory queue and can be run independently in multiple replicas.

## HTTP API

All endpoints require the standard bearer token and operate only on the authenticated user's data.
List operations are paginated and ordered newest first.

| Operation | Endpoint | Successful `data` |
|---|---|---|
| List center records | `GET /v1/notifications?status=all\|unread` | Paginated notifications |
| Read center status | `GET /v1/notification-status` | `{ unread_count }` |
| Replace one read state | `PATCH /v1/notifications/:id` | Notification |
| Mark existing records read | `PUT /v1/notification-read-state` | `{ updated_count, unread_count }` |
| Read preferences | `GET /v1/notification-preferences` | Effective preferences |
| Replace preferences | `PUT /v1/notification-preferences` | Effective preferences |

A notification response is plain-text and user scoped:

```json
{
  "id": 81,
  "kind": "billing.invoice_paid",
  "title": "Invoice paid",
  "body": "Invoice 1042 was paid successfully.",
  "action_url": "/console/invoices/1042",
  "is_read": false,
  "read_at": null,
  "created_at": "2026-07-15T20:00:00Z"
}
```

`status` accepts only `all` or `unread` and defaults to `all`. `PATCH` accepts exactly
`{ "is_read": boolean }`. Setting the same state twice is idempotent.

Bulk read state uses a high-water mark rather than an unbounded list:

```json
{
  "through_id": 81
}
```

Only delivered `in_app` records owned by the current user with `id <= through_id` are marked read.
Notifications created concurrently above the high-water mark remain unread.

Preference replacement accepts both booleans and is idempotent:

```json
{
  "in_app_enabled": true,
  "email_enabled": false
}
```

## Browser Adapter

The fixed same-origin adapter mirrors the resources under `/api`:

| Browser endpoint | Upstream endpoint |
|---|---|
| `GET /api/notifications` | `GET /v1/notifications` |
| `PATCH /api/notifications/:id` | `PATCH /v1/notifications/:id` |
| `GET /api/notification-status` | `GET /v1/notification-status` |
| `PUT /api/notification-read-state` | `PUT /v1/notification-read-state` |
| `GET/PUT /api/notification-preferences` | matching `/v1/notification-preferences` |

Every response is `Cache-Control: private, no-store` and varies on `Cookie`. Unsafe methods enforce
same-origin checks before authentication or body parsing. Successful upstream and mock payloads are
strictly validated before entering React Query state. The mock BFF is development-only and must
preserve ownership, pagination, read high-water marks, preferences, envelopes, and stable errors.

## Audit And Privacy

Preference changes and user-initiated read-state changes emit audit metadata containing only user,
notification, channel-setting, and count identifiers. Publication content, recipient addresses,
and delivery provider details are excluded from audit records.

Notification rows contain user-visible content and require an application retention policy.
Downstream apps should avoid secrets, reset tokens, credentials, and unnecessary personal data in
titles, bodies, idempotency keys, kinds, and action URLs.

## Stable Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 404 | `NOTIFICATION.NOT_FOUND` | The notification is absent, not delivered in-app, or owned by another user |
| 409 | `NOTIFICATION.IDEMPOTENCY_CONFLICT` | Internal publication reused a key with different immutable content |
| 422 | `NOTIFICATION.INVALID_CHANNEL` | Internal publication declared an unsupported or inconsistent channel |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Notification persistence or the selected backend is unavailable |

Malformed JSON, field validation, global envelopes, and `request_id` behavior follow
[`README.md`](README.md). The internal publication errors are documented even though browser clients
cannot publish, because downstream modules and operational tests must branch on stable semantics.

## Deliberate Deferrals

The starter does not ship SMS, push, Slack/chat, broadcast/WebSocket fan-out, digests, schedules,
topics, segments, organization-level preference inheritance, a template editor, provider callbacks,
or an administrator activity UI. Those concerns can extend the publisher and delivery adapter seams
without changing the notification center contract.

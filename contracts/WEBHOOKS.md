# Webhook Contract

The optional `webhook` starter owns organization-scoped outbound endpoint subscriptions, durable
event and delivery records, Standard Webhooks signatures, retries, delivery logs, secret rotation,
and operator replay. It depends on the default `user` and `audit` starters and the optional
`organization` starter.

Webhook is an outbound integration boundary. It is not the in-process domain event bus, an inbound
callback router, a generic HTTP proxy, or an analytics stream.

## Activation

Enable the API starter in every API, migration, worker, and operator process:

```dotenv
OPTIONAL_STARTERS=organization,webhook
WEBHOOK_ENCRYPTION_KEY=replace_with_at_least_32_random_characters
```

Enable the matching browser feature in the Web build:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,webhook
```

Selection is additive and restart-scoped. The API refuses to start with `webhook` selected and no
strong encryption key. The Web feature never silently falls back to mock data in production.

## Ownership And Authorization

Every endpoint, event, delivery, and attempt belongs to exactly one organization. HTTP management
routes require authentication, a verified `Organization-Id` context, and an `owner` or `admin`
organization role. Members receive `403 PERMISSION.DENIED`; cross-organization identifiers resolve
as `404` without revealing existence.

The browser cannot publish arbitrary events, override delivery state, read event payloads, or invoke
an arbitrary URL. Downstream server modules publish only through `domain.WebhookPublisher`.

## Finite Event Catalog

Event types are a finite code-owned catalog. Keys use lowercase dotted segments and each definition
owns a payload validator. The shipped catalog contains only `webhook.test`, whose exact object
schema is owned by the starter. Downstream apps add reviewed product event definitions and typed
publisher adapters before subscriptions can select them.

An internal publication contains:

| Field | Rule |
|---|---|
| `organization_id` | Existing organization owner scope |
| `source` | Required lowercase dotted producer identifier, at most 64 bytes |
| `event_id` | Required producer idempotency key, at most 128 safe ASCII bytes |
| `type` | Exact catalog entry |
| `occurred_at` | UTC timestamp, no more than 24 hours in the future or 30 days in the past |
| `data` | Exact JSON object accepted by the event definition, at most 64 KiB |

`(organization_id, source, event_id)` is unique. Repeating identical content returns the original
message and creates no duplicate deliveries. Reusing the tuple with different type, occurrence, or
payload fails with `WEBHOOK.IDEMPOTENCY_CONFLICT`.

The durable publisher honors Luas's transaction context. A business repository can therefore store
its state and enqueue the webhook event in one database transaction. Publishing through the
process-local event bus alone is best-effort and is not a durable outbox guarantee.

## Endpoint Model

An endpoint has a bounded display name, one target URL, one or more exact event types, an active or
disabled status, a monotonic version, and one endpoint-unique signing secret.

Target rules:

- absolute `https` URL by default;
- no user information, fragment, query string, or ambiguous hostname;
- at most 2,048 bytes;
- redirects are never followed;
- DNS is resolved again at delivery time and the exact resolved IP is dialed;
- loopback, private, link-local, multicast, unspecified, and metadata-network destinations are
  rejected unless explicitly allowed for non-production local verification;
- TLS verification remains enabled.

Production rejects insecure HTTP and private-target overrides. Deployment egress controls remain a
recommended defense in depth boundary.

## Signing Secret

Creation and rotation generate 32 random bytes serialized as `whsec_<base64>`. Plaintext appears
only in the successful creation or rotation response. Persistence contains AES-256-GCM ciphertext,
a non-secret hint, and a key version; list and delivery responses never contain ciphertext or
plaintext.

Rotation retains the previous encrypted signing secret for a bounded overlap window. During that
window the sender emits signatures for both current and previous secrets, allowing zero-downtime
consumer rotation. After the window only the current signature is emitted. Delete clears encrypted
secret material and cancels undelivered work.

## Standard Webhooks Delivery

One event receives a server-generated `msg_` identifier that remains stable across automatic retry
and operator replay. The compact JSON request body is:

```json
{
  "id": "msg_01J...",
  "type": "webhook.test",
  "timestamp": "2026-07-15T20:00:00Z",
  "data": {
    "organization_id": 42,
    "endpoint_id": 9
  }
}
```

Each `POST` uses `Content-Type: application/json` and these exact headers:

```text
webhook-id: msg_01J...
webhook-timestamp: 1784145600
webhook-signature: v1,<base64-hmac> [v1,<previous-base64-hmac>]
user-agent: Luas-Webhooks/1.0
```

For each secret, HMAC-SHA256 signs the exact bytes
`<webhook-id>.<webhook-timestamp>.<request-body>`. The sent body is the signed body. A `2xx` status is
successful; every other status is a failure. `408`, `425`, `429`, `5xx`, timeout, and transient
network errors are retryable. Redirects, invalid targets, and other `4xx` responses are terminal.

Responses are read only through a configured byte limit and are never persisted. Delivery and
attempt records store local failure codes, status code, duration, attempt number, and whether the
response exceeded the drain limit. They never store response bodies, free-form network errors,
target URLs, event payloads, secrets, or request signatures.

## Retry, Lease, And Disable Policy

Workers claim due deliveries with database row locks, `SKIP LOCKED` where supported, random lease
tokens, and expiring leases. Network I/O happens outside the claim transaction. Completion requires
the same lease token, so a stale worker cannot overwrite a newer claim.

Retries use bounded exponential delays with deterministic jitter over roughly three days and at
most ten attempts. Terminal failures increment an endpoint's consecutive-failure counter; a success
resets it. Five consecutive terminal failures automatically disable the endpoint and cancel pending
work. Re-enabling is an explicit owner/admin action and does not silently replay canceled events.

Operator replay preserves the original message ID, appends new attempt records, and is allowed only
for terminal deliveries whose retained event and active endpoint still exist. Retention pruning
never removes events needed by pending or processing deliveries.

## HTTP API

All routes below require authentication, active organization context, and owner/admin authorization.
Responses are `private, no-store` and vary on `Authorization` and `Organization-Id`.

| Operation | Endpoint | Successful `data` |
|---|---|---|
| List catalog | `GET /v1/webhook-event-types` | Bounded event type list |
| List endpoints | `GET /v1/webhook-endpoints` | Paginated endpoint summaries |
| Create endpoint | `POST /v1/webhook-endpoints` | Endpoint plus one-time `signing_secret` |
| Replace endpoint configuration | `PATCH /v1/webhook-endpoints/:id` | Endpoint summary |
| Delete endpoint | `DELETE /v1/webhook-endpoints/:id` | No content |
| Replace endpoint status | `PUT /v1/webhook-endpoints/:id/status` | Endpoint summary |
| Rotate signing secret | `POST /v1/webhook-endpoints/:id/secret-rotations` | Endpoint plus one-time secret |
| Queue endpoint test | `POST /v1/webhook-endpoints/:id/tests` | Delivery summary |
| List deliveries | `GET /v1/webhook-deliveries` | Paginated delivery summaries |
| List attempts | `GET /v1/webhook-deliveries/:id/attempts` | Paginated attempt summaries |

Configuration replacement accepts exactly:

```json
{
  "name": "Order processor",
  "url": "https://hooks.example.com/luas",
  "event_types": ["webhook.test"]
}
```

Create uses the same object. Status replacement accepts exactly `{ "enabled": true }`. Endpoint
update, status, delete, and rotation require `If-Match: "webhook-endpoint-v<version>"`; absence is
`428 WEBHOOK.PRECONDITION_REQUIRED`, while stale state is
`409 WEBHOOK.ENDPOINT_VERSION_CONFLICT`.

Test delivery requires one canonical `Idempotency-Key` header. It can publish only the fixed
`webhook.test` schema and only to the selected endpoint; it is not a generic browser publication
surface.

## Browser Adapter

The same-origin Web handlers mirror the fixed resources under `/api`. They forward only reviewed
paths and headers through the authenticated Go adapter, enforce same-origin checks before unsafe
work, bound request and response bodies, strictly validate successful JSON, and never accept a
caller-provided upstream URL.

The development mock BFF preserves organization ownership, role checks, endpoint versions,
one-time secrets, finite event selection, status transitions, and delivery response shapes. It does
not make outbound network calls or pretend to prove production delivery.

## Stable Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 404 | `WEBHOOK.ENDPOINT_NOT_FOUND` | Endpoint is absent or outside the active organization |
| 404 | `WEBHOOK.DELIVERY_NOT_FOUND` | Delivery is absent or outside the active organization |
| 409 | `WEBHOOK.IDEMPOTENCY_CONFLICT` | Trusted publisher reused an event identity with different content |
| 409 | `WEBHOOK.ENDPOINT_VERSION_CONFLICT` | Endpoint state changed since the caller read it |
| 409 | `WEBHOOK.REPLAY_NOT_ALLOWED` | Delivery is not terminal or required retained state is unavailable |
| 422 | `WEBHOOK.INVALID_EVENT_TYPE` | Event type is missing from the finite catalog |
| 422 | `WEBHOOK.INVALID_TARGET` | Endpoint target violates URL or network policy |
| 428 | `WEBHOOK.PRECONDITION_REQUIRED` | A versioned endpoint mutation omitted `If-Match` |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Starter persistence, encryption, or dispatch is unavailable |

Malformed JSON, schema errors, pagination, global envelopes, and `request_id` follow
[`README.md`](README.md).

## Privacy And Audit

Endpoint create/update/status/rotation/delete/test and operator replay/prune actions emit minimized
audit metadata. Audit and logs may include organization, endpoint, event type, delivery, version,
status, and local failure identifiers. They must not include endpoint URLs, DNS answers, payloads,
signing secrets, ciphertext, signatures, response bodies, or free-form network errors.

Event payloads are business data retained only while needed for delivery and the configured replay
horizon. Downstream definitions must exclude credentials, reset tokens, and unnecessary personal
data. The browser never receives event payloads.

## Deliberate Deferrals

The starter does not provide inbound webhooks, provider callbacks, CloudEvents translation,
arbitrary headers, custom methods, query-string credentials, payload transforms, endpoint-specific
event schemas created at runtime, wildcard subscriptions, broadcast topics, billing events,
cross-organization endpoints, or exactly-once consumer processing. Consumers must verify timestamps
and signatures and deduplicate `webhook-id`.

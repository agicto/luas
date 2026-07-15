# Webhook Starter

The optional `webhook` starter provides organization-owned outbound subscriptions and a durable
delivery ledger. The canonical cross-half behavior is
[`../../contracts/WEBHOOKS.md`](../../contracts/WEBHOOKS.md); the durable architecture decision is
[`adr/0009-outbound-webhook-delivery-boundary.md`](adr/0009-outbound-webhook-delivery-boundary.md).

## Runtime Shape

- `domain.WebhookPublisher` is the trusted server-side publication seam.
- `domain.WebhookDispatcher` claims and sends due deliveries.
- `domain.WebhookMaintainer` replays terminal deliveries and prunes retained terminal history.
- `internal/modules/webhook` owns catalog, encryption adapter, endpoint policy, persistence, HTTP
  sender, service, routes, and tests.
- `luas webhook:work` runs one or more independent database-ledger workers.
- `luas webhook:publish-test`, `webhook:replay`, and `webhook:prune` are bounded operator tools.

The worker uses the same image, database, `WEBHOOK_ENCRYPTION_KEY`, and `OPTIONAL_STARTERS` snapshot
as the API process. It does not depend on the memory workflow queue.

## Production Configuration

```dotenv
OPTIONAL_STARTERS=organization,webhook
WEBHOOK_ENCRYPTION_KEY=<at-least-32-random-characters>
WEBHOOK_REQUEST_TIMEOUT=15s
WEBHOOK_MAX_RESPONSE_BYTES=65536
WEBHOOK_SECRET_OVERLAP=24h
WEBHOOK_EVENT_RETENTION=720h
```

`WEBHOOK_ALLOW_INSECURE_HTTP` and `WEBHOOK_ALLOW_PRIVATE_TARGETS` default to false and are rejected
in production. They exist only for explicit local/Compose verification. The application SSRF
boundary validates every resolved address and pins each connection to an approved IP. Production
should also restrict worker egress at the network layer and keep cloud metadata defenses enabled.

## Adding A Product Event

Add one code-owned definition to the webhook catalog with an exact payload validator, then expose a
small typed publisher adapter from the owning product module. Call `WebhookPublisher.PublishWebhook` inside
the product repository's transaction context when durable atomicity matters. Do not publish an
untyped map from handlers, expose generic browser ingestion, or make runtime event definitions.

Event changes are contract changes. Version an incompatible schema with a new event type rather than
silently changing the old payload.

## Operations

Run continuously:

```bash
luas webhook:work --batch=25 --poll=2s
```

One-shot verification and maintenance:

```bash
luas webhook:work --once
luas webhook:publish-test --organization=42 --endpoint=9 --actor=7 --idempotency-key=deploy-check-001
luas webhook:replay --organization=42 --delivery=81 --actor=7
luas webhook:prune --before=2026-06-15T00:00:00Z
```

Replay is at-least-once and preserves `webhook-id`. Consumers must deduplicate. Pruning never removes
pending/processing state or events still referenced by retained deliveries.

## Replacement And Removal

To replace dispatch with a broker or webhook provider, preserve `domain.WebhookPublisher`, stable
message identity, subscription filtering, Standard Webhooks signing, and the documented HTTP
management contract. Keep provider SDKs outside domain and Web code.

To remove the starter, stop publishers and workers, disable or drain endpoints, remove `webhook` from
every API/Web selection, deploy without its routes and runtime seams, then apply the downstream data
retention policy before dropping its tables.

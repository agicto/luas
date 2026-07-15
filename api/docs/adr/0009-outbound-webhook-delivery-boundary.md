# ADR 0009: Outbound Webhook Delivery Boundary

## Status

Accepted

## Context

Luas has an in-process event bus, a process-local workflow queue, durable notification delivery,
organizations, audit logs, and usage accounting. Integration-heavy downstream apps still repeat a
more security-sensitive workflow: tenant-owned endpoint configuration, signing-key custody, durable
fan-out, retries, redelivery, delivery visibility, and SSRF controls.

The memory workflow queue cannot be the durability authority because work is lost with the process.
The notification starter cannot own third-party integration payloads because its records and
preferences are addressed to Luas users. Treating an arbitrary callback URL as a generic capability
would hide organization ownership, authorization, retention, and disable policy.

## Decision

Add an organization-dependent optional `webhook` starter. It owns a finite code-defined event
catalog, organization endpoints and subscriptions, idempotent durable events, lease-safe deliveries,
minimized attempt logs, endpoint-unique encrypted signing secrets, bounded dual-signature rotation,
automatic retry, failure-threshold disablement, and operator replay/prune seams.

Trusted modules publish through `domain.WebhookPublisher`. The repository resolves Luas's bound
transaction context so a downstream business write and webhook outbox insert can commit atomically.
The in-process event bus remains useful for best-effort reactions but is not advertised as a durable
webhook source.

Outbound requests follow Standard Webhooks HMAC-SHA256 headers and sign the exact sent body. The
HTTP adapter resolves and validates targets at delivery time, dials the validated IP, disallows
redirects, bounds time and response bytes, verifies TLS, and rejects non-public destinations by
default. Production cannot enable insecure HTTP or private-target escape hatches.

The browser manages only active-organization endpoints and reads minimized delivery state. It never
publishes arbitrary events or reads payloads, secrets after the one-time response, ciphertext,
signatures, target-resolution details, or provider responses. The built-in `webhook.test` definition
proves the pipeline without pretending Luas owns downstream product event semantics.

## Consequences

- New projects gain a production-oriented outbound integration path without selecting a broker or
  webhook SaaS provider.
- Reliable publication requires callers to use the publisher inside their owning transaction; a
  post-commit call still has an unavoidable application-level gap.
- PostgreSQL is the durable starter-scale ledger. High-volume apps may replace the publisher and
  dispatcher seams with a stream/outbox platform while preserving HTTP and signing behavior.
- Endpoint URLs and event payloads remain sensitive application data with explicit retention and
  egress policy obligations.
- Symmetric HMAC is broadly interoperable and simple, while asymmetric signatures remain a future
  adapter decision.
- Runtime-created event schemas, wildcard subscriptions, inbound provider callbacks, and arbitrary
  proxy behavior remain outside the starter.

## Verification Impact

- catalog and service tests cover exact event schemas, publication idempotency, fan-out, secret
  one-time handling, CAS mutations, rotation overlap, retry classification, disablement, replay, and
  retention;
- network tests cover URL parsing, public-IP enforcement, DNS pinning resistance, redirects,
  response limits, timeout, TLS, and Standard Webhooks signatures;
- PostgreSQL Compose runs a real receiver and verifies signature headers, one delivery under replay,
  retry/disable state, migration down/up, and organization isolation;
- Web tests cover fixed adapters, same-origin mutation guards, strict schemas, manager-only UI, and
  mock/production parity;
- root governance keeps starter selection, contracts, secret configuration, no-public-publish,
  security controls, privacy, worker commands, and extraction guidance aligned.

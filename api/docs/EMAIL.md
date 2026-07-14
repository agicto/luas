# Email Capability

`internal/infra/email` is an optional outbound email capability backed by Resend. It is not a
provider-neutral transport and it is not a notification starter: it owns provider delivery mechanics,
while a future notification starter owns
preferences, records, retries, delivery status, and user-facing workflows.

## Configuration

Configure the sender and provider key together:

```dotenv
MAIL_FROM=Luas <noreply@example.com>
RESEND_API_KEY=re_...
MAIL_REQUEST_TIMEOUT=10s
```

`MAIL_FROM` and `RESEND_API_KEY` are an all-or-none pair. A partial configuration, an invalid sender,
or a non-positive timeout fails typed startup validation. With neither value set, the capability is
disabled and `Service.IsConfigured()` returns false. `MAIL_REQUEST_TIMEOUT` defaults to 10 seconds.

Configuration follows the startup snapshot and restart lifecycle in
[`CONFIGURATION.md`](CONFIGURATION.md). Provider credentials must come from the deployment secret
store and must not be committed to an environment file.

## Delivery Boundary

Every send:

- accepts the caller's `context.Context` and applies the smaller of caller cancellation and the
  capability-owned request timeout;
- reuses one process-owned `http.Client` instead of constructing a client per message;
- accepts one to 50 recipient addresses and requires non-empty subject and HTML content before
  network I/O;
- reads at most 64 KiB from the provider response;
- accepts only a 2xx response containing a non-empty provider message ID;
- returns `ErrNotConfigured`, `ErrInvalidMessage`, `ErrProviderResponseTooLarge`,
  `ErrInvalidProviderResponse`, or a status-only `ProviderError` as appropriate;
- never returns the provider response body, which may contain recipient PII or provider diagnostics.

Password-reset and welcome templates HTML-escape dynamic values. The capability does not log
recipients, subject lines, message bodies, API keys, or provider response bodies. Feature-owned outer
layers may log stable operation names, internal IDs, outcomes, and the sanitized error.

## Current Semantics

Delivery is direct and best-effort. Luas does not currently persist an outbox, retry provider
failures, expose delivery receipts, or provide exactly-once semantics.

- Password-reset requests intentionally preserve their generic public success response on lookup,
  storage, configuration, or delivery failure to avoid account enumeration. Internal logs use the
  user ID when known, never the requested email address; lookup failures record only the error
  type rather than a potentially sensitive repository error message.
- Welcome delivery is fire-and-forget after account creation. It keeps request correlation values but
  is detached from request cancellation; the email timeout still bounds the provider operation.
- Organization invitations must persist their business state before attempting email delivery. A
  provider failure must not erase or silently accept an invitation state transition.

Use workflow/queue infrastructure or a future notification starter when a downstream app needs
durable retries, delivery history, preferences, or multiple channels.

## Replacement

Downstream starters should depend on a narrow feature-owned mailer interface and adapt `*email.Service`
in their provider wiring, as the `user` starter does. Replacing Resend should remain local to this
capability or a new adapter; provider request/response types must not leak into domain entities,
contracts, handlers, or Web code.

## Verification

```bash
cd api
go test ./internal/infra/email ./internal/infra/config ./internal/modules/user
go test -race ./internal/infra/email ./internal/modules/user
```

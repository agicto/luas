# Notification Starter

The optional `notification` starter owns user-scoped notification records, global channel
preferences, in-app read state, and a durable email delivery ledger. The shared email and event
packages remain technical capabilities; they do not own notification workflow state.

The HTTP and browser contract is canonical in
[`../../contracts/NOTIFICATIONS.md`](../../contracts/NOTIFICATIONS.md).

## Activation

Use one additive starter selection for API replicas, migration jobs, seeders, and workers:

```dotenv
OPTIONAL_STARTERS=notification
```

The starter depends only on the default `user` and `audit` starters. It does not require
`organization` or `permission`. Selection is restart-scoped; disabled services fail with
`COMMON.SERVICE_UNAVAILABLE`, and disabled manifests contribute no routes or migrations.

Run the migration before serving traffic. The starter owns:

- `notifications`: immutable publication content, recipient ID, idempotency key, fingerprint, and
  mutable in-app read timestamp;
- `notification_deliveries`: one channel state machine, attempt count, availability, lease,
  destination hash, stable failure code, and completion timestamp;
- `notification_preferences`: one global `in_app`/`email` preference row per user.

## Publishing From A Starter

Inject `domain.NotificationPublisher` into the downstream service that owns the business event:

```go
notification, err := publisher.Publish(ctx, domain.NotificationPublication{
	UserID:         userID,
	IdempotencyKey: "invoice:1042:paid",
	Kind:           "billing.invoice_paid",
	Title:          "Invoice paid",
	Body:           "Invoice 1042 was paid successfully.",
	ActionURL:      "/console/invoices/1042",
	Channels: []domain.NotificationChannel{
		domain.NotificationChannelInApp,
		domain.NotificationChannelEmail,
	},
})
```

The caller owns a stable idempotency key derived from its business operation. Replaying an identical
publication returns the original record; reusing the key for different immutable content returns
`NOTIFICATION.IDEMPOTENCY_CONFLICT`. Do not generate a random key inside a retry loop.

Only reviewed application services publish. There is no public HTTP creation endpoint. Keep titles
and bodies plain text, use a local relative action URL, and never put secrets, reset tokens,
credentials, recipient addresses, or unnecessary personal data in publication fields.

Preferences are evaluated transactionally when the publication is first created. A channel listed
in `RequiredChannels` bypasses the user's preference; use this only for a reviewed transactional or
security requirement. Preference changes do not cancel or create existing delivery rows.

## Delivery Worker

Email provider I/O is outside request paths. Run at least one worker when applications publish the
`email` channel:

```bash
/app/luas notification:work --batch=25 --poll=2s
```

Operational flags:

| Flag | Contract |
|---|---|
| `--batch` | Claim 1-100 due rows per dispatch call; default 25 |
| `--poll` | Idle/error base interval from 100 ms to 1 minute; default 2 seconds |
| `--max-attempts` | Stop after this many completed delivery attempts; zero means unlimited |
| `--once` | Run one bounded dispatch call, useful for a scheduled job or verification |

Multiple workers may share one database. Claims use a short transaction, due-state compare/update,
random lease token, and `SKIP LOCKED` on PostgreSQL/MySQL. Provider calls run after commit. A worker
may complete only with its current lease token; expired processing rows can be reclaimed without an
old worker overwriting the newer result.

Transient provider/network failures retry with bounded exponential delay and at most five attempts.
Every provider attempt uses `notification-email-<delivery_id>` as the Resend idempotency key. The
first attempt binds a hash of the current recipient route; a changed email address fails closed
rather than reusing the same provider key for a different recipient.

If email is not configured, an email delivery terminates with `EMAIL.NOT_CONFIGURED`. Configure
`MAIL_FROM`, `RESEND_API_KEY`, and `MAIL_REQUEST_TIMEOUT` as described in
[`EMAIL.md`](EMAIL.md). A worker deployment must use the same image, database, secrets, and
`OPTIONAL_STARTERS` selection as the serving API.

## Events, Audit, And Privacy

After a newly created publication commits, the starter best-effort publishes
`notification.created`. Its payload contains notification ID, user ID, and kind only. A failed
in-process event publication does not roll back the durable notification.

User-initiated read and preference changes emit minimized audit metadata. The delivery ledger and
logs store stable local failure codes, internal IDs, and hashes; they do not store recipient
addresses, provider response bodies, free-form provider errors, title/body copies, or credentials.

Notification content still belongs to application data retention. A downstream app must choose
retention and deletion policy for old notifications and terminal deliveries. Do not prune pending or
processing rows without an explicit operational policy.

## Replacement Or Removal

To replace email delivery, adapt the private `emailSender` seam without leaking provider types into
domain, HTTP, or Web code. New channels require a deliberate domain/contract extension, a unique
delivery row per notification/channel, worker semantics, preference behavior, privacy review, and
tests; do not model provider messages as additional notifications.

To remove the starter, first stop publications and workers, then remove `notification` from every
`OPTIONAL_STARTERS` value. Deploy code without its routes, publisher consumers, worker process,
manifest/provider contribution, migration ownership, contract/docs, and matching Web feature before
dropping retained tables according to the downstream data policy.

## Verification

```bash
cd api
go test ./internal/modules/notification ./internal/infra/email ./internal/infra/console/commands
OPTIONAL_STARTERS=notification go run ./cmd/luas route:list
go run ./cmd/luas route:list
```

The default route list must contain no notification routes. The selected list must add exactly the
six authenticated resources from the contract. Migration verification must exercise both `Up` and
`Down` against the production database driver.

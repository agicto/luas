# Notification Web Feature

The optional Web `notification` feature presents user-scoped in-app notifications and global
channel preferences. It consumes the fixed browser resources in
[`../../contracts/NOTIFICATIONS.md`](../../contracts/NOTIFICATIONS.md); publication and provider
delivery remain API-only concerns.

## Activation

Enable the API starter and matching browser feature independently but deploy them as one reviewed
release:

```dotenv
# API processes, migration jobs, and notification workers
OPTIONAL_STARTERS=notification

# Web build
NEXT_PUBLIC_OPTIONAL_FEATURES=notification
```

For production against the Luas API, also configure `API_ADAPTER_ENABLED=true` and the server-only
adapter settings from [`AUTHENTICATION.md`](AUTHENTICATION.md). Local development can use the mock
BFF. An enabled feature with neither backend returns `503 COMMON.SERVICE_UNAVAILABLE`; it never
invents production state.

## Browser Boundary

The feature owns explicit same-origin Route Handlers for:

- `GET /api/notifications`;
- `PATCH /api/notifications/:id`;
- `GET /api/notification-status`;
- `PUT /api/notification-read-state`;
- `GET` and `PUT /api/notification-preferences`.

Production handlers forward only their fixed `/v1` counterparts with the server-owned API session.
Unsafe requests enforce same-origin checks before authentication or body parsing. Responses are
private and `no-store`, and browser cookies or authorization input are never forwarded upstream.

Successful responses are parsed with strict Zod schemas before React Query state is updated. IDs,
timestamps, pagination, kind grammar, text bounds, and known fields are validated. Action URLs must
remain same-origin relative paths after percent-decoding; protocol-relative paths, backslashes,
control characters, malformed encoding, and absolute URLs are rejected. Titles and bodies are
rendered as React text, never injected HTML.

## UI And Query Behavior

The console loads the notification component only when the feature is selected. The unread status
query polls every 60 seconds with a 30-second stale window. The paginated list is requested only when
the center opens, and preferences only when their dialog opens. This keeps the disabled default
scaffold free of notification polling and keeps optional list/prefetch work out of initial use.

"Mark all read" sends the newest loaded notification ID as a high-water mark. Notifications created
later remain unread. Opening a notification first attempts its user-scoped read mutation, then
navigates only through the already validated local action URL.

The mock backend keeps state per authenticated mock user and preserves ownership non-disclosure,
newest-first pagination, unread filtering, read high-water behavior, preferences, envelopes, and
stable errors. Its seed notifications are development scaffold examples, not API seed data or
product content.

## Downstream Replacement Or Removal

To keep the feature with another backend, preserve the documented browser contract or replace the
feature service and explicit Route Handlers together. Do not turn `src/server/api-adapter` into a
catch-all proxy or expose a provider API directly to browser JavaScript.

To remove it:

1. Remove `notification` from `NEXT_PUBLIC_OPTIONAL_FEATURES` and rebuild Web.
2. Delete `src/features/notification`, its five `/api` resource directories, notification i18n
   module/namespace, route tests, and console layout contribution.
3. Remove the matching API starter and data only through the staged process in
   [`../../api/docs/NOTIFICATIONS.md`](../../api/docs/NOTIFICATIONS.md).
4. Update contract, catalog, surface, and governance references in the same change.

## Verification

```bash
cd web
pnpm vitest run \
  src/test/notification-contract.test.ts \
  src/test/notification-route.test.ts \
  src/test/notification-ui.test.tsx
pnpm type-check
pnpm lint
NEXT_PUBLIC_OPTIONAL_FEATURES=notification pnpm build
```

Also exercise mock and production-adapter modes in a browser at desktop and mobile widths. Check the
unread badge, plain-text rendering, high-water read update, preference replacement, disabled feature,
cross-user non-disclosure, cross-origin rejection, and browser console.

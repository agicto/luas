# Asset Web Feature

The optional Web `asset` feature provides a private upload, list, download, and delete workflow for
the API contract in [`../../contracts/ASSETS.md`](../../contracts/ASSETS.md). An asset is browser
metadata and lifecycle, while the stored object is opaque provider bytes. The feature never exposes
provider SDKs, bucket names, object keys, checksums, or durable public URLs to browser state.

## Activation

Build the feature only when the API starter is deployed in the same release:

```dotenv
# API replicas, migration jobs, and cleanup jobs
OPTIONAL_STARTERS=asset

# Web build
NEXT_PUBLIC_OPTIONAL_FEATURES=asset
```

Production uses `API_ADAPTER_ENABLED=true` and the server-only adapter settings from
[`AUTHENTICATION.md`](AUTHENTICATION.md). Local development can use the mock BFF. An enabled feature
with neither backend returns `503 COMMON.SERVICE_UNAVAILABLE`; it does not fabricate production
state.

## Browser Boundary

The feature owns fixed same-origin management routes under `/api/assets` plus the development-only
`/api/asset-transfers/:token` transport. Production management handlers forward only code-owned
`/v1/assets` paths. Unsafe requests enforce same-origin checks before authentication or body
parsing, and private responses use `Cache-Control: private, no-store` with `Vary: Cookie`.

All successful JSON is parsed with strict Zod schemas before entering React Query. Unknown fields,
invalid UUIDs, unsupported methods, malformed timestamps, unknown lifecycle states, and provider
metadata fail closed. Upload and download grants stay in the mutation call stack; they are not
cached in React Query, Zustand, browser storage, analytics, or UI error text.

Transfer URLs are accepted only when they are HTTPS or the exact same-origin local/mock transfer
path. URL credentials, fragments, protocol-relative values, redirects, and forbidden request
headers are rejected before `fetch`. Transfer requests use `credentials: omit`, `cache: no-store`,
`redirect: error`, and no referrer. Downloads are fetched as bytes and exposed only through a
short-lived browser object URL. Download responses are streamed through the declared asset size and
media-type boundary before a Blob is created; a missing, short, oversized, or mismatched response
fails closed.

## UI And Query Behavior

`/console/assets` is available only in an asset-enabled build. The console shows a compact status
filter, bounded file picker, newest-first table, status labels, and explicit download/delete
actions. The browser checks the declared allowlist and configured scaffold limit before requesting
an intent; the API remains authoritative.

Next.js still compiles the guarded file-system route in a disabled build. It returns not-found and
does not enter navigation or the initial `/console` client chunk set; downstream apps that require
zero asset artifact inventory should remove the feature using the extraction steps below.

The upload workflow is deliberately sequential:

```text
create intent -> upload bytes to short-lived grant -> complete -> invalidate list
```

Completion or transfer failure never inserts a fake ready row. Delete requires confirmation and
invalidates the list after success. File contents, signed query values, and provider response text
must never appear in toast messages.

The development mock keeps state and object bytes per authenticated mock user, caps active objects
and aggregate bytes, enforces lifecycle and inspection parity, and returns attachment downloads. It
is a browser-contract substitute, not durable storage or a production endpoint.

## Replacement Or Removal

To keep the UI with another backend, preserve the documented HTTP and grant contract or replace the
service and fixed Route Handlers together. Keep transfer credentials ephemeral and do not turn the
server adapter into a catch-all proxy.

To remove it:

1. remove `asset` from `NEXT_PUBLIC_OPTIONAL_FEATURES` and rebuild Web;
2. delete `src/features/asset`, the asset management and transfer Route Handlers, asset i18n module,
   console page/navigation contribution, and asset tests;
3. remove the API starter and provider data through the staged process in
   [`../../api/docs/ASSETS.md`](../../api/docs/ASSETS.md);
4. update contract, catalog, surface, and governance references in the same change.

## Verification

```bash
cd web
pnpm vitest run src/test/asset-contract.test.ts src/test/asset-route.test.ts src/test/asset-ui.test.tsx
pnpm type-check
pnpm lint
NEXT_PUBLIC_OPTIONAL_FEATURES=asset pnpm build
```

Also exercise mock and production-adapter modes in a browser at desktop and mobile widths. Verify a
valid text/PDF upload, completion, download bytes and filename, delete confirmation, disabled-route
404, rejected content, expired/foreign grants, cross-user non-disclosure, cross-origin rejection,
responsive table overflow, and an empty browser console.

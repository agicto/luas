# Asset Starter

The optional `asset` starter provides a private, user-owned upload lifecycle without coupling
feature code to a filesystem, bucket, or storage SDK. The shared HTTP contract is
[`../../contracts/ASSETS.md`](../../contracts/ASSETS.md).

An **asset** is the persisted owner, metadata, policy, and lifecycle record. A **stored object** is
the opaque byte payload behind `internal/capabilities/storage`. Keep those concepts separate:
provider keys and local paths are infrastructure details, not public asset identity.

## Activation

Use one identical selection for serving replicas, migration jobs, cleanup jobs, and CLI processes:

```dotenv
OPTIONAL_STARTERS=asset
```

Outside production, selecting `asset` defaults `OBJECT_STORAGE_DRIVER` to `local`. Production
selection fails startup unless the driver is explicitly `r2`; Luas never silently places
production uploads on an application container filesystem.

Relevant configuration:

| Variable | Default | Contract |
|---|---:|---|
| `OBJECT_STORAGE_DRIVER` | `disabled`, or `local` when selected outside production | `disabled`, `local`, or `r2`; production asset deployments require `r2` |
| `OBJECT_STORAGE_LOCAL_ROOT` | `storage/objects` | Private development root; directories are `0700`, files are `0600` |
| `OBJECT_STORAGE_REQUEST_TIMEOUT` | `30s` | Bounds one provider metadata, object, or signing operation |
| `ASSET_MAX_BYTES` | `10485760` | Per-object limit; valid range is 1 byte through 100 MiB |
| `ASSET_UPLOAD_GRANT_TTL` | `10m` | Upload credential lifetime; no more than one hour |
| `ASSET_DOWNLOAD_GRANT_TTL` | `5m` | Download credential lifetime; no more than 15 minutes |
| `ASSET_PENDING_TTL` | `1h` | Pending cleanup lifetime; at least the upload TTL and no more than 24 hours |

R2 additionally requires `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`,
`R2_REGION`, and an HTTPS `R2_ENDPOINT`. The access key, secret, bucket, and endpoint are an
all-or-none group; region defaults to `auto`. The account-specific host belongs in the endpoint. Bucket
names, credentials, object keys, signed URLs, checksums, and provider errors must not enter logs or
browser metadata.

## Architecture

The starter owns metadata and lifecycle in `internal/modules/asset`. The provider-neutral
`ObjectStore` and `DirectTransferStore` interfaces live in `internal/capabilities/storage`; local
and R2 adapters live in `internal/infra/storage`.

```text
authenticated route -> asset service -> asset repository -> PostgreSQL
                              |-> content inspector
                              |-> object-store capability -> local or R2 adapter
```

Uploads first target `asset-uploads/<uuid>/object`. Completion checks staging metadata, freezes bytes
under `assets/<uuid>/object`, rechecks and streams that immutable final snapshot through bounded
inspection, computes SHA-256, removes current staging bytes, and only then marks metadata `ready`.
The private staging key remains until grant expiry so cleanup can remove a late provider PUT. The
browser never receives either key.

The local adapter uses Go's rooted filesystem API so traversal and symlink escapes fail beneath the
configured root. Writes are bounded and atomically renamed. It is a development adapter, not a
shared or durable production volume strategy.

The R2 adapter uses AWS SDK for Go v2, bounded retries, path-style addressing, and presigned PUT/GET
operations. Grant headers omit browser-controlled or forbidden headers such as `Host`,
`Authorization`, `Cookie`, and `Content-Length`. Configure R2 CORS for the exact application
origins, methods, and returned headers. Apply a provider lifecycle rule to the `asset-uploads/`
prefix as defense in depth.

## Inspection And Threat Boundary

The default allowlist accepts JPEG, PNG, WebP, PDF, UTF-8 plain text, and UTF-8 CSV. The inspector
validates binary signatures and rejects invalid UTF-8 or NUL-containing text. Declared
`Content-Type`, filename extension, provider metadata, detected bytes, and configured size must
agree.

This is a content-integrity baseline, not an antivirus or document-sandbox claim. A downstream app
that accepts hostile documents must compose malware scanning, quarantine, and any preview
transformation before readiness. Never add HTML or SVG merely because a browser can display it.

## Lifecycle, Cleanup, And Recovery

The state machine is `pending -> ready|rejected`, with any active state able to become `deleted`.
Repository claims use bounded operation leases so duplicate completion, deletion, and cleanup do
not concurrently own one row. Completion also recognizes a final object left by a crash between
promotion and database update. Cleanup may clear expired staging metadata for a ready asset but
never deletes or clears its final object.

Run cleanup as a bounded scheduled job:

```bash
/app/luas asset:prune --batch=100
```

The command is safe to repeat and reports counts without names, keys, URLs, hashes, or content. It
removes stale staging/final objects before closing expired pending and rejected records. Deleted
tombstones retain private cleanup keys until the pending lifetime ends so the command can also remove
a provider PUT that arrives after user-visible deletion, then clear those keys. Provider lifecycle
remains the backstop for objects left by process or database failure.

Delete removes object bytes before recording a tombstone. The starter registers an account-deletion
guard, resolves it through the user deletion transaction, and locks the user row before asset
creation, so concurrent intent creation cannot orphan provider objects under a soft-deleted user. A
downstream retention policy still owns tombstone retention, backups, and legal requirements.

## Replacement Or Removal

To replace object storage, implement the capability interfaces and keep provider types inside
`internal/infra/storage`. To replace inspection, implement the private inspector seam and preserve
bounded reads and fail-closed readiness. Do not expose a generic bucket client to modules.

To remove the starter:

1. stop new uploads and cleanup jobs;
2. remove `asset` from every `OPTIONAL_STARTERS` value;
3. deploy without asset routes, provider/manifest contributions, migration ownership, console
   command, contract, and matching Web feature;
4. delete retained objects under an explicit provider policy, then drop metadata according to the
   downstream retention plan;
5. remove object-storage secrets if no other downstream capability owns them.

## Verification

```bash
cd api
go test ./internal/modules/asset ./internal/capabilities/storage ./internal/infra/storage
OPTIONAL_STARTERS=asset go run ./cmd/luas route:list
go run ./cmd/luas route:list
go run ./cmd/luas asset:prune --batch=1
```

The default route list must contain no asset route. The selected route list must contain the five
authenticated management routes and two token-authenticated local transfer routes. Migration tests
must exercise `Up` and `Down` against the production database driver; Compose verification must
exercise intent, byte upload, completion, download, deletion, and post-delete non-disclosure.

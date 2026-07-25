# Web SPA Deployment

## Artifact Contract

Run:

```bash
corepack pnpm install --frozen-lockfile
corepack pnpm build
```

`dist/` is the complete production artifact. It contains `index.html`,
content-hashed assets, and a build manifest. It contains no server runtime,
source maps, secrets, or runtime environment loader.

`VITE_*` values are compiled into the artifact. Build once per public
configuration, or keep deployment-specific behavior at the CDN/gateway layer.

## OSS And CDN Routing

Configure rules in this order:

1. Serve an existing object exactly.
2. Route an allowlisted `/api/*` prefix to the browser gateway or Go API when
   the application uses same-origin API calls.
3. Rewrite every other missing application path to `/index.html` with HTTP
   `200`.

The fallback must be an internal rewrite, not a `301`/`302` redirect, so a
request for `/console/preferences` keeps its URL while TanStack Router renders
the route.

When deploying under a subpath, set `VITE_BASE_PATH=/subpath/` before the
build and scope the CDN fallback to that path.

## Cache Policy

| Object                     | Recommended policy                                                                        |
| -------------------------- | ----------------------------------------------------------------------------------------- |
| `index.html`               | `Cache-Control: no-cache` or a very short reviewed TTL                                    |
| `/assets/<content-hash>.*` | `Cache-Control: public, max-age=31536000, immutable`                                      |
| `.vite/manifest.json`      | Build evidence; private or omitted from public upload if the deploy tool does not need it |

Upload hashed assets before `index.html`. This prevents a newly cached HTML
document from referencing assets that are not available yet. Keep prior hashed
assets during the rollback window.

## Security Headers

The CDN or static host owns response headers. Start from:

```text
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'
Referrer-Policy: strict-origin-when-cross-origin
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

Add explicit API origins to `connect-src` only when same-origin routing is not
possible. TLS/HSTS policy belongs to the production domain and CDN.

## Release And Rollback

1. Build from a frozen lockfile.
2. Run static-output and bundle-budget checks through `pnpm build`.
3. Upload assets to a versioned prefix or release bucket.
4. Smoke-test the release origin.
5. Switch the CDN origin/version atomically.
6. Purge or revalidate `index.html`; do not purge immutable hashed assets.
7. Roll back by restoring the previous HTML/version pointer.

The build manifest and commit SHA should remain attached to deployment
evidence even when the public host serves only runtime files.

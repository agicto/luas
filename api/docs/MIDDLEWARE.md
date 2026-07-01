# HTTP Middleware Catalog

This catalog names the HTTP middleware that belongs to the Luas API core, the default starter set, optional route/starter behavior, and the deployment layer.

Use this document when changing `api/internal/bootstrap/http.go`, adding starter-owned middleware, or deciding whether a cross-cutting concern should become part of the default scaffold.

## Ownership

| Owner | Meaning | Examples |
|---|---|---|
| Core default | Always part of the API HTTP kernel or enabled by safe production defaults. | request ID, recovery, metrics, security headers, body limit, timeout, CORS, production rate limit |
| Starter-owned | Registered by a default or optional starter because it needs starter dependencies or domain rules. | JWT auth, API key auth, audit logging |
| Route/starter opt-in | Available in the scaffold, but only specific routes or starters should choose it. | compression, version middleware, custom throttles |
| Deployment-owned | Better handled by the gateway, CDN, WAF, load balancer, or hosting platform. | global compression, distributed rate limits, TLS termination, bot protection |

## Default Kernel

`api/internal/bootstrap/http.go` wires the core HTTP middleware in this order:

1. `RequestID`
2. `GinLogger`
3. `Recovery`
4. `metrics.Middleware`
5. `Helmet`
6. `BodyLimit`
7. `Timeout`
8. `CORS`
9. `RateLimit` when enabled

Order matters:

- `RequestID` runs before anything that might log or emit an error response.
- `Recovery` wraps later middleware and routes.
- `Helmet`, `BodyLimit`, and `Timeout` protect every request.
- `CORS` runs before `RateLimit` so browser preflight requests do not consume rate limit quota.
- `RateLimit` runs before routes and after CORS. Health and metrics paths are skipped by default.

## Default Guardrails

| Middleware | Default behavior | Contract |
|---|---|---|
| `Helmet` | Sends baseline security headers. | Response headers only. |
| `BodyLimit` | Limits request bodies to `MIDDLEWARE_BODY_LIMIT_MB`, default 10 MB. | `413` + `COMMON.REQUEST_TOO_LARGE`. |
| `Timeout` | Adds a cooperative request context deadline from `MIDDLEWARE_REQUEST_TIMEOUT`, default 180 seconds. | `503` + `COMMON.TIMEOUT` when the handler respects context and has not written a response. |
| `CORS` | Allows configured browser origins. Production must use explicit origins. | CORS headers and preflight handling. |
| `RateLimit` | Enabled by default only when `APP_ENV=production`; default `600` requests per minute per client IP. | `429` + `COMMON.RATE_LIMITED`, `Retry-After`, and `X-RateLimit-*` headers. |

Environment knobs:

```bash
MIDDLEWARE_REQUEST_TIMEOUT=180
MIDDLEWARE_BODY_LIMIT_MB=10
MIDDLEWARE_RATE_LIMIT_ENABLED=true
MIDDLEWARE_RATE_LIMIT_MAX=600
MIDDLEWARE_RATE_LIMIT_WINDOW=1m
MIDDLEWARE_RATE_LIMIT_SKIP_PATHS=/health,/health/live,/health/ready,/metrics,/v1/health
```

## Starter-Owned Middleware

Starter-owned middleware is registered through the starter registry, not the core HTTP kernel.

| Middleware | Owner | Why |
|---|---|---|
| JWT auth | `user` starter | Needs the JWT service and auth domain errors. |
| API key auth | `apikey` starter | Needs API key hashing, lookup, expiry, and revocation rules. |
| Audit logging | `audit` starter | Needs audit persistence and starter-specific write-side behavior. |

Keep starter-owned middleware route-scoped or starter-scoped. Do not move it into the core kernel just because many routes use it.

## Opt-In Middleware

| Middleware | Preferred owner | Reason |
|---|---|---|
| `Compress` | Deployment/CDN first; route/starter opt-in second | Global API compression can duplicate CDN behavior and changes response encoding. |
| `VersionMiddleware` | Route group | API versioning is a contract boundary, not a global transport default. |
| Custom throttles | Route/starter | Some routes need stricter limits than the default production guardrail. |

## Change Checklist

Before moving middleware between categories:

1. Decide the owner: core default, starter-owned, route/starter opt-in, or deployment-owned.
2. Update this catalog and any affected README entry.
3. If response behavior changes, update `../../contracts/README.md`.
4. Add or update tests at the public seam:
   - default kernel behavior: `api/internal/bootstrap/http_test.go`
   - middleware contract behavior: middleware or `api/tests/unit`
   - configuration defaults: `api/internal/infra/config`
5. Run targeted tests and `golangci-lint run ./...`.

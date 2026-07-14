# HTTP Middleware Catalog

This catalog names the HTTP middleware that belongs to the Luas API core, the default starter set, optional route/starter behavior, and the deployment layer.

Use this document when changing `api/internal/bootstrap/http.go`, adding starter-owned middleware, or deciding whether a cross-cutting concern should become part of the default scaffold.

## Ownership

| Owner | Meaning | Examples |
|---|---|---|
| Core default | Always part of the API HTTP kernel or enabled by safe production defaults. | request ID, recovery, security headers, body limit, timeout, CORS, production rate limit |
| Core opt-in | Core operational behavior with explicit runtime configuration. | Prometheus request instrumentation and `/metrics` |
| Starter-owned | Registered by a default or optional starter because it needs starter dependencies or domain rules. | JWT auth, authentication abuse guard, API key auth, audit logging |
| Route/starter opt-in | Available in the scaffold, but only specific routes or starters should choose it. | compression, version middleware, custom throttles |
| Deployment-owned | Better handled by the gateway, CDN, WAF, load balancer, or hosting platform. | global compression, distributed rate limits, TLS termination, bot protection |

## Default Kernel

`api/internal/bootstrap/http.go` wires the core HTTP middleware in this order:

1. `RequestID`
2. `GinLogger`
3. `Recovery`
4. `metrics.Middleware` when `METRICS_ENABLED=true`
5. `Helmet`
6. `BodyLimit`
7. `Timeout`
8. `CORS`
9. `RateLimit` when enabled

Order matters:

- `RequestID` runs before anything that might log or emit an error response.
- `Recovery` wraps later middleware and routes.
- `metrics.Middleware` is absent when metrics are disabled, avoiding per-request instrumentation overhead.
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
| `Metrics` | Enabled by default outside production and disabled by default in production. Unmatched routes share the bounded `unmatched` path label. | Prometheus text endpoint at `/metrics` when enabled. |

Environment knobs:

```bash
MIDDLEWARE_REQUEST_TIMEOUT=180
MIDDLEWARE_BODY_LIMIT_MB=10
MIDDLEWARE_RATE_LIMIT_ENABLED=true
MIDDLEWARE_RATE_LIMIT_MAX=600
MIDDLEWARE_RATE_LIMIT_WINDOW=1m
MIDDLEWARE_RATE_LIMIT_SKIP_PATHS=/health,/health/live,/health/ready,/metrics,/v1/health
METRICS_ENABLED=true
```

## Proxy Trust Boundary

The kernel calls Gin's `SetTrustedProxies` before installing middleware. With the default empty
`SERVER_TRUSTED_PROXIES`, `ClientIP()` uses the direct network peer and ignores
`X-Forwarded-For`/`X-Real-IP`. This makes IP-based logging and rate limits resistant to spoofed
forwarding headers.

Set `SERVER_TRUSTED_PROXIES` only to the exact load balancer, ingress, or private proxy IPs/CIDRs
that append or sanitize forwarding headers:

```bash
SERVER_TRUSTED_PROXIES=10.20.0.0/16,192.0.2.10
```

Invalid values and trust-all networks (`0.0.0.0/0`, `::/0`) fail configuration validation. The
optional `RealIP` middleware follows the same deny-by-default rule when used by downstream apps.

## Operational Routes

- `/health/live` and `/health/ready` are always registered for deployment orchestration.
- `/metrics` is registered only when `METRICS_ENABLED=true`. Production deployments must restrict it with network policy, a gateway, or a private listener.
- The previous `/monitor` dashboard and `/swagger` route are not default surfaces. They were removed because the monitor depended on an unassembled global container and no generated Swagger contract existed.
- A future machine-readable OpenAPI contract or dedicated management listener should be added as a separate verified slice instead of restoring placeholder routes.

## Starter-Owned Middleware

Starter-owned middleware is registered through the starter registry, not the core HTTP kernel.

| Middleware | Owner | Why |
|---|---|---|
| JWT auth | `user` starter | Needs the JWT service and auth domain errors. |
| Authentication abuse guard | `user` starter | Public login, registration, and reset operations need endpoint-specific per-IP and per-subject quotas. |
| API key auth | `apikey` starter | Needs API key hashing, lookup, expiry, and revocation rules. |
| Audit logging | `audit` starter | Needs audit persistence and starter-specific write-side behavior. |

Keep starter-owned middleware route-scoped or starter-scoped. Do not move it into the core kernel just because many routes use it.

The authentication guard is enabled by default in production and disabled by default elsewhere.
It uses separate buckets per endpoint, then independent source-IP and normalized/hashed subject
buckets where configured. A single `IP+subject` combined key is intentionally avoided because it
does not stop one source from sweeping accounts or many sources from targeting one account.
Sensitive auth responses return the canonical `429` + `COMMON.RATE_LIMITED` envelope without
quota diagnostics or a bucket-specific reason.

```bash
AUTH_RATE_LIMIT_ENABLED=true
AUTH_RATE_LIMIT_LOGIN_IP_MAX=20
AUTH_RATE_LIMIT_LOGIN_IP_WINDOW=5m
AUTH_RATE_LIMIT_LOGIN_SUBJECT_MAX=10
AUTH_RATE_LIMIT_LOGIN_SUBJECT_WINDOW=15m
AUTH_RATE_LIMIT_PASSWORD_RESET_IP_MAX=10
AUTH_RATE_LIMIT_PASSWORD_RESET_IP_WINDOW=1h
AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_MAX=3
AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_WINDOW=1h
```

These starter limits use process-local memory. Multi-replica deployments must enforce equivalent
distributed buckets in a WAF/gateway or replace the store with Redis. Per-IP limiting is a baseline,
not a complete credential-stuffing defense; downstream products should add MFA, risk signals, and
graduated challenges according to their threat model.

## Opt-In Middleware

| Middleware | Preferred owner | Reason |
|---|---|---|
| `Compress` | Deployment/CDN first; route/starter opt-in second | Global API compression can duplicate CDN behavior and changes response encoding. |
| `VersionMiddleware` | Route group | API versioning is a contract boundary, not a global transport default. |
| Custom throttles | Route/starter | Some routes need stricter limits than the default production guardrail. |

## Performance Baseline

Run the representative steady-state middleware benchmark before and after changing the core chain:

```bash
make benchmark-http
```

The benchmark sends a `GET` to a parameterized route through request ID, Helmet, body limit,
cooperative timeout, and CORS middleware. It runs the same path with Prometheus instrumentation
disabled and enabled. Logging, tracing, rate limiting, network I/O, and handler serialization are
excluded, and request/response shells are reused between iterations, so the comparison stays focused
on steady-state middleware work owned by the kernel.

The API test suite also guards a steady-state allocation budget of at most `21 allocs/request` for
both variants. The budget deliberately leaves headroom above the current `18 allocs/request` and is
stable across repeated runs on the pinned Go toolchain. Nanosecond timings are measurement evidence,
not a CI service-level objective; compare them on the same host and toolchain.

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
6. Run `make benchmark-http` when the core chain or request metrics change, and report before/after
   timing, bytes, and allocations from the same machine.

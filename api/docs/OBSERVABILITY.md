# Observability And Sensitive Data

Luas treats logs, traces, local diagnostics, and audit records as durable data boundaries. They are
useful for correlation and operations, but they are not a second copy of requests or secrets.

## Request Logs

The default Gin request logger records status, latency, client IP, method, route-template path, and
`request_id`. It deliberately:

- records `/users/:id` rather than the concrete path parameter;
- records `unmatched` instead of attacker-controlled paths when no route matches;
- never records raw query strings or query values;
- never records request or response bodies;
- passes structured context through `pkg/redact` before any handler receives it.

Route templates and the unmatched-route sentinel reduce credential/PII exposure and bound log
cardinality. Do not place credentials in URLs even though Luas minimizes their telemetry surface.

## Redaction Contract

`pkg/redact` is the reusable, standard-library-only boundary for credential-shaped fields. It
normalizes key spelling and replaces password, secret, token, authorization, cookie, API-key,
session, OTP, private/signing/encryption-key, credential, and signature values with `[REDACTED]`.
Query-only `code`, `key`, and `sig` parameters are also treated as credentials. Nested string-key
maps and slices are copied and sanitized to a bounded depth; caller-owned values are not mutated.

This is defense in depth, not permission to log secrets. A secret hidden under an ambiguous key or
embedded in a free-form message cannot be classified reliably. Callers must still avoid credentials,
full email addresses, phone numbers, payment data, and raw provider payloads.

## Exception Center

The HTML exception center is development-only. It collects no body, redacts sensitive headers and
query values, uses the route template in place of concrete path parameters, and HTML-escapes every
dynamic field before rendering. `Authorization`, `Proxy-Authorization`, Cookie, `Set-Cookie`,
`X-API-Key`, CSRF tokens, OAuth codes, and similarly named values must never appear in the page.

Production must keep `APP_DEBUG=false`; production panic responses remain the normal JSON
`COMMON.INTERNAL` boundary and do not expose stack, request, SQL, or environment details.

## Database And Tracing

GORM logging always enables `ParameterizedQueries`, and the observed logger preserves GORM's
optional `ParamsFilter` interface while forcing bound parameters out before `Trace`. GORM's special
`Scan` recorder is also parameterized because it otherwise expands values before calling the
configured logger. SQL diagnostics therefore preserve statement shape while keeping values out of
stdout and the local exception timeline. OpenTelemetry uses GORM's parameterized statement shape as
`db.statement`. Do not disable parameterization to improve a debug screenshot; reproduce locally
with explicit non-secret fixtures instead.

HTTP server spans use the route template for the span name and `http.route`. Luas deliberately omits
the concrete `url.path`, raw query values, and free-form error messages from exported spans; traces
carry status and stable `error.type` summaries while structured logs own reviewed diagnostic detail.
Unmatched requests use only the HTTP method as the span name.

## Audit Records

Audit records intentionally persist actor, target, action, result, request correlation, and reviewed
business changes. The audit starter redacts sensitive `Changes` fields and recursively sanitizes
`Metadata` again at the service boundary before persistence. Feature code should record stable IDs
and state transitions, never bearer values, passwords, invitation tokens, provider payloads, or PII.

## Logging APIs

- Business and event operations use the existing `log/slog` seams.
- HTTP runtime, request, exception-center, and configured output handlers use `pkg/logger`.
- Do not add another logging library or silently mix the two seams inside one operation.
- Use stable event names and structured fields; never interpolate a secret into the message.

## Verification

```bash
go test ./pkg/redact ./pkg/logger ./pkg/errors ./internal/infra/exception ./internal/infra/database ./internal/infra/tracing ./internal/modules/audit
go test ./pkg/redact -run '^$' -bench '^BenchmarkMap$' -benchmem -count=5
python3 ../.agents/skills/luas-framework-review/scripts/check-sensitive-telemetry.py
```

---
name: logging-standards
description: Design or change Luas structured events, request correlation, log levels, or redaction. Use when telemetry behavior changes, not for routine code that emits no new signal.
---

# Logging Standards

## Purpose

Add production-useful telemetry without duplicating facts, leaking private
data, or creating unbounded cardinality and storage cost.

Read `docs/OBSERVABILITY.md` for the canonical sensitive-data and request-log
boundary. Follow nearby `log/slog`, `pkg/logger`, or event middleware usage
before introducing a new helper.

## Event Design

Prefer one wide event at the operation boundary over many narrow progress
logs. A useful event has:

- a stable dotted name such as `user.login` or `webhook.delivery_failed`;
- outcome and duration where meaningful;
- bounded identifiers needed for correlation;
- an `err` field for unexpected failure;
- no raw request/response body, credential, token, or private payload.

Log once at the outermost layer that knows the operation outcome. Wrapping and
returning an error is not another logging site.

## Field And Cardinality Rules

| Include when owned | Avoid |
|---|---|
| `request_id`, `correlation_id`, `job_id` | passwords, tokens, secrets, cookie values |
| stable internal actor/resource IDs | email, phone, full name, address |
| route template, method, status | concrete path/query values or bodies |
| bounded outcome/type/attempt | stack traces or provider bodies in user-visible fields |
| duration/size counts | arbitrary map keys or user-generated field names |

Use `pkg/redact` for credential-shaped values. Diagnostic HTML must escape
rendered fields, and SQL logs remain parameterized.

## Level Criteria

| Level | Use |
|---|---|
| `Debug` | Temporary or opt-in diagnostic detail with understood volume |
| `Info` | Retained business/operational outcome worth querying |
| `Warn` | Recoverable degradation or fallback |
| `Error` | Unexpected failed operation that needs investigation |

Expected validation failures, permission denials, and not-found outcomes are
not automatically errors. Routine cache hits and per-row progress do not need
retained info logs.

## Runtime Seams

- HTTP: request middleware owns request IDs, route templates, status, latency,
  and request completion.
- Business operation: the service/use-case boundary owns the final outcome.
- Events: use `events.LoggingMiddleware`; do not log every handler step.
- Background work: use `job_id` and `correlation_id`, not a fabricated request
  ID.
- Infrastructure startup/shutdown: log lifecycle once at the owner.

Use `InfoContext`/`ErrorContext` with structured key/value fields. Do not use
`fmt.Sprintf` inside the event name. The example under `examples/` shows
minimal request-scoped `slog` wiring; load it only when nearby code lacks a
pattern.

## Review Workflow

1. State the operator question the new event answers.
2. Find the existing owner of the same request/operation lifecycle.
3. Define a stable name, level, bounded fields, and redaction behavior.
4. Remove duplicate lower-layer logging.
5. Add focused assertions when redaction, correlation, route templates, or
   level behavior changed.

## Verification

```bash
bash .agents/skills/logging-standards/scripts/validate-logging.sh
go test ./pkg/logger ./pkg/redact ./internal/infra/events/...
```

Choose only packages touched by the change. Do not run the full repository
gate for a local event-name or field adjustment.

## Checklist

- Stable event name, correct level, and one logical owner.
- Correlation fields are present when available.
- No secrets, raw private payloads, concrete query/path data, or unbounded keys.
- Error is a structured `err` field rather than interpolated text.
- Request/event middleware is reused instead of reimplemented.
- Tests cover any changed redaction or structured-field contract.

## Legacy Calls

Existing `log.Printf` sites under infrastructure packages are migration debt.
Replace one only when touching its owning behavior; do not create a sweeping
logging-only refactor.

## Related Skills

- Root `luas-code-review`: explicit diff review.
- Root `systematic-debugging`: temporary diagnostic evidence for an unclear
  failure.

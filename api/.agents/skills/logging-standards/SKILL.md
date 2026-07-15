---
name: logging-standards
description: Structured logging across Luas business/event and HTTP runtime seams, including redaction, stable events, correlation, and level criteria.
---

# Logging Standards

## Purpose

Logs only earn their cost when they can answer a future question. This skill enforces structured
logging with enough context to be useful in production and a hard boundary against credentials and
unbounded request data.

Luas currently has two deliberate logging seams:

- business and event operations use Go's `log/slog` through existing module/event patterns;
- HTTP requests, configured output handlers, the recent-log buffer, and exception diagnostics use
  `pkg/logger`.

Do not introduce `logrus`, `zap`, `zerolog`, or a third seam. Do not migrate between the two in an
unrelated change. See `docs/OBSERVABILITY.md` for redaction and request-shape ownership.

## When to Use

- Adding new logs to a handler, service, or background job.
- Reviewing a PR where new logs were added.
- Debugging: structuring repro logs so you can grep them later.
- Setting up logging in a new module.

## Core Principles

### 1. Wide events, not narrow events

Log **once per logical operation** with **all the context you have** at that moment. Avoid sprinkling `slog.Info("starting X")` / `slog.Info("X done")` pairs around the same code path — they make searching harder, not easier.

Wrong:

```go
slog.Info("login attempt")
slog.Info("password checked")
slog.Info("token generated")
slog.Info("login success")
```

Right:

```go
slog.InfoContext(ctx, "user.login",
    "user_id", user.ID,
    "email_domain", emailDomain,
    "ip", clientIP,
    "duration_ms", time.Since(start).Milliseconds(),
    "outcome", "success",
)
```

### 2. High-cardinality context is the goal

Every log line should carry the identifiers that let you correlate it with other systems and other logs. Default keys:

| Key | Where it comes from | Example |
|---|---|---|
| `request_id` | gin middleware | `c.GetString("request_id")` |
| `user_id` | jwt claims | `c.GetUint("user_id")` |
| `correlation_id` | event/job metadata | `meta.CorrelationID` |
| `module` | static, per-package | `"user"`, `"audit"` |
| `outcome` | the result | `"success"`, `"denied"`, `"timeout"` |

If a line has no identifying context, ask why it exists.

### 3. Use stable event names

Use a stable, dotted event name as the first arg — never a free-form sentence. The name is what you grep/filter on; the args answer follow-up questions.

```go
slog.InfoContext(ctx, "user.login", ...)            // good
slog.InfoContext(ctx, "audit.change.recorded", ...) // good
slog.InfoContext(ctx, "User logged in successfully") // bad — un-greppable
```

Naming convention: `<module>.<noun>[.verb]` — `user.login`, `apikey.revoked`, `audit.change.recorded`.

### 4. Never log secrets or PII

- Passwords, tokens, API keys, session cookies → **never**, even at DEBUG.
- Emails, phone numbers, addresses → only hashed prefix or domain in normal logs; full value only behind a debug feature flag.
- Credit card data → never.

If you must log a token for debugging, log its prefix (`token[:8]`) and length, never the value.

`pkg/logger` recursively redacts credential-shaped context keys as defense in depth. That does not
protect secrets embedded in free-form messages or hidden under vague keys. Request logs use the Gin
route template (or the bounded `unmatched` sentinel), never concrete path parameters, raw query
strings, or bodies. HTTP traces omit concrete paths and free-form error text. GORM logs remain
parameterized, including the special `Scan` recorder path.

## Level Criteria

| Level | When | Example |
|---|---|---|
| `Debug` | Verbose, off in prod, useful for repro | "Resolved route to handler", "Cache hit" |
| `Info` | Normal business outcomes worth keeping | "user.login", "apikey.created" |
| `Warn` | Recoverable problem, system still works | "external API slow", "fallback path used" |
| `Error` | Operation failed, user-visible or data-affecting | "user.login failed: db down", "payment.charge failed" |

Use `slog.LevelDebug` only when you can justify the per-line cost in production volume. Most "debug" logs should be deleted before merge — production-time signal is what you grep months later.

## Implementation Patterns

### Pattern 1: Request-scoped logger via context

Attach request-scoped fields once in middleware; downstream code calls `slog.InfoContext(ctx, ...)`.

```go
// internal/infra/middleware/logging.go
type loggerKey struct{}

func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        reqID := c.GetHeader("X-Request-Id")
        if reqID == "" {
            reqID = uuid.NewString()
        }
        c.Set("request_id", reqID)

        logger := slog.Default().With(
            "request_id", reqID,
            "method", c.Request.Method,
            "path", c.FullPath(),
        )
        ctx := context.WithValue(c.Request.Context(), loggerKey{}, logger)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}

func LoggerFrom(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}
```

Downstream:

```go
LoggerFrom(ctx).InfoContext(ctx, "user.login", "user_id", user.ID, "outcome", "success")
```

### Pattern 2: Error logging

Always log the error value as a structured field, not interpolated into the message.

```go
// bad
slog.Error(fmt.Sprintf("failed to charge: %v", err))

// good
slog.ErrorContext(ctx, "payment.charge_failed",
    "user_id", userID,
    "amount_cents", amount,
    "err", err,
)
```

For wrapped errors, log at the **outer** layer with full context — not at every wrap-and-return.

### Pattern 3: Event handler logging

`events.LoggingMiddleware` in `internal/infra/events/middleware.go` already provides structured logging around handlers — use it, don't roll your own per-handler logs.

```go
events.WithMiddleware(
    events.LoggingMiddleware(slog.Default(), events.LogLevelInfo),
    events.RecoveryMiddleware(slog.Default()),
)
```

### Pattern 4: Background jobs / scheduled tasks

Background work has no `request_id`. Use `job_id` and `correlation_id` instead, attached when the job is enqueued.

```go
logger := slog.Default().With(
    "job_id", job.ID,
    "job_type", job.Type,
    "correlation_id", job.CorrelationID,
)
logger.InfoContext(ctx, "job.started", "attempt", job.Attempt)
// ... do work
logger.InfoContext(ctx, "job.completed", "duration_ms", time.Since(start).Milliseconds())
```

## Anti-patterns

- **String interpolation in the message**: `slog.Info(fmt.Sprintf("user %d", id))` defeats structured logging. Use args.
- **Logging the same fact at multiple levels**: pick one site (usually the outermost) and log there.
- **Logging without context**: a log with no identifiers is worse than no log — it costs storage and doesn't help triage.
- **Excessive DEBUG**: DEBUG that fires on every request becomes line noise. If it fires every request, it's INFO; if it never fires, delete it.
- **Logging the raw request/response body**: PII risk + huge cost. Log shape (length, content-type, ID), not contents.
- **Logging concrete path parameters or query values**: they create privacy risk and unbounded cardinality. Use the route template and stable identifiers.
- **Adding `logrus` or `zap`**: the existing `log/slog` and `pkg/logger` seams are sufficient; new dependencies need explicit justification.
- **Using `log.Printf`** for new code: it goes to stderr without structure. Replace with `slog.InfoContext` / `slog.ErrorContext`.

## Verification Checklist

When reviewing a PR that adds logs:

- [ ] Uses the owning `log/slog` or `pkg/logger` seam, not `log.Printf`, `fmt.Println`, `logrus`, or `zap`.
- [ ] First arg is a stable event name (`module.verb`), not a sentence.
- [ ] Includes `request_id` / `correlation_id` / `user_id` where the operation has them.
- [ ] No secrets, tokens, passwords, or full PII in log args.
- [ ] Request logs contain route templates and no concrete path parameters, query values, or bodies.
- [ ] SQL logging remains parameterized and diagnostic HTML is escaped.
- [ ] Error logs include the `err` field, not just an interpolated message.
- [ ] Level matches criteria above (no INFO for routine cache hits; no ERROR for expected 404s).
- [ ] One log per logical operation, not per micro-step.

## Pair With

- `code-review-guide` — log review is part of the review checklist.
- `systematic-debugging` — debug logs should match the events you grep for later.
- `verification-before-completion` — confirm logs render correctly (`go run ./cmd/api` and trigger the path; inspect JSON output).

## Reference: Project Files

- `docs/OBSERVABILITY.md` — canonical sensitive-data and request-log boundary.
- `pkg/redact` — shared credential-shaped redaction.
- `pkg/logger/gin.go` — HTTP request logger and route-template behavior.
- `internal/infra/events/middleware.go` — canonical slog event middleware pattern.
- `internal/infra/events/middleware_test.go` — how to assert on structured logs in tests.
- `examples/logging-setup.go` (this skill) — minimal slog wiring example.

## Migration Notes (existing `log.Printf` usage)

There are existing `log.Printf` call sites in `internal/infra/{database,events,middleware,queue,schedule}/`. These predate this skill. Treat them as legacy: leave them alone in unrelated PRs, replace with `slog` when you touch the surrounding code for another reason. Do not file a sweeping "migrate all logs" PR — too much surface area, too little value per line.

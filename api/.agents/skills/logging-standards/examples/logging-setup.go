// Package main shows a minimal slog wiring example matching the
// logging-standards skill. This file is illustrative; the real wiring
// lives in internal/infra/middleware/ and internal/infra/events/.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================================
// Logger Setup
// ============================================================================

// initLogger configures slog with a JSON handler at INFO level and sets it
// as the default. Call this once at program startup.
func initLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Rename the default "time" key to "timestamp" so it matches
			// the rest of the observability pipeline.
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))
}

// ============================================================================
// Request-Scoped Logger (Pattern 1)
// ============================================================================

type loggerKey struct{}

// RequestLogger attaches request-scoped fields once at the edge so every
// downstream slog call inherits request_id, method, and path.
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

// LoggerFrom returns the request-scoped logger if present, else the default.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ============================================================================
// Usage Examples
// ============================================================================

// userLoginExample shows a wide-event log: one line per logical operation
// with all the context we have at that point.
func userLoginExample(ctx context.Context, userID uint, emailDomain, ip string) {
	start := time.Now()

	// ... auth work happens here ...

	LoggerFrom(ctx).InfoContext(ctx, "user.login",
		"user_id", userID,
		"email_domain", emailDomain,
		"ip", ip,
		"duration_ms", time.Since(start).Milliseconds(),
		"outcome", "success",
	)
}

// paymentFailureExample shows error logging — err is a structured field,
// the message is a stable event name, no fmt.Sprintf into the message.
func paymentFailureExample(ctx context.Context, userID uint, amountCents int64, err error) {
	LoggerFrom(ctx).ErrorContext(ctx, "payment.charge_failed",
		"user_id", userID,
		"amount_cents", amountCents,
		"err", err,
	)
}

// backgroundJobExample shows job-scoped logging using With() instead of
// per-call args, because every line in this job shares the same identifiers.
func backgroundJobExample(ctx context.Context, jobID, jobType, correlationID string) {
	logger := slog.Default().With(
		"job_id", jobID,
		"job_type", jobType,
		"correlation_id", correlationID,
	)

	start := time.Now()
	logger.InfoContext(ctx, "job.started", "attempt", 1)

	// ... do work ...

	logger.InfoContext(ctx, "job.completed",
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func main() {
	initLogger()
	r := gin.New()
	r.Use(RequestLogger())
	// register routes...
	_ = r.Run(":8080")
}

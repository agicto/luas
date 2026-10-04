package user

import (
	"context"
	"log/slog"
	"time"
)

const (
	// passwordResetConcurrency bounds in-flight reset deliveries per process. Requests beyond it are
	// dropped and logged; the per-IP and per-email quotas keep legitimate traffic far below it.
	passwordResetConcurrency = 32
	passwordResetTimeout     = 30 * time.Second
)

// backgroundRunner runs best-effort work after the request returns, detached from its cancellation
// but keeping its values (request ID, trace) for logs. A task in flight when the process exits is
// lost, which is acceptable only for work the caller can safely repeat.
type backgroundRunner struct {
	name    string
	slots   chan struct{}
	timeout time.Duration
}

func newBackgroundRunner(name string, concurrency int, timeout time.Duration) *backgroundRunner {
	return &backgroundRunner{name: name, slots: make(chan struct{}, concurrency), timeout: timeout}
}

func (r *backgroundRunner) run(ctx context.Context, task func(context.Context)) {
	select {
	case r.slots <- struct{}{}:
	default:
		slog.WarnContext(ctx, r.name+"_dropped", "reason", "too many tasks in flight")
		return
	}
	detached := context.WithoutCancel(ctx)
	go func() {
		defer func() { <-r.slots }()
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(detached, r.name+"_panicked", "panic", recovered)
			}
		}()
		taskCtx, cancel := context.WithTimeout(detached, r.timeout)
		defer cancel()
		task(taskCtx)
	}()
}

// runInline runs the task on the caller's goroutine. Tests use it to observe effects directly.
func runInline(ctx context.Context, task func(context.Context)) {
	task(ctx)
}

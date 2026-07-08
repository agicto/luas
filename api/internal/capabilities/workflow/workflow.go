package workflow

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

var (
	// ErrNilJob is returned when attempting to dispatch a nil background job.
	ErrNilJob = errors.New("workflow: nil job")
	// ErrNilTask is returned when attempting to execute a nil task.
	ErrNilTask = errors.New("workflow: nil task")
	// ErrMaxAttemptsExceeded is returned when a retried task exhausts its attempts.
	ErrMaxAttemptsExceeded = errors.New("max retry attempts exceeded")
)

// TaskFunc is a small, retry-friendly unit of work.
type TaskFunc func(ctx context.Context) error

// ShouldRetry decides whether a workflow task error should be retried.
type ShouldRetry func(err error) bool

// RetryPolicy describes retry behavior for synchronous workflow steps.
type RetryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64
	ShouldRetry  ShouldRetry
}

// Manager is the deep seam over queue, scheduler, and retry behavior.
type Manager struct {
	queue     *QueueManager
	scheduler *Scheduler
}

var defaultManager = NewManager()

// Default returns the shared workflow manager.
func Default() *Manager {
	return defaultManager
}

// NewManager creates a workflow manager backed by the global queue and scheduler.
func NewManager() *Manager {
	return NewManagerWith(GlobalQueue(), GlobalScheduler())
}

// NewManagerWith creates a workflow manager backed by explicit infra instances.
func NewManagerWith(queueManager *QueueManager, scheduler *Scheduler) *Manager {
	if queueManager == nil {
		queueManager = GlobalQueue()
	}
	if scheduler == nil {
		scheduler = GlobalScheduler()
	}
	return &Manager{
		queue:     queueManager,
		scheduler: scheduler,
	}
}

// Register makes a background job deserializable for queue workers.
func (m *Manager) Register(job Job) *Manager {
	if job == nil {
		return m
	}
	m.queue.RegisterJob(job)
	return m
}

// Dispatch queues a background job, optionally targeting a named queue.
func (m *Manager) Dispatch(ctx context.Context, job Job, opts ...DispatchOption) error {
	if job == nil {
		return ErrNilJob
	}

	cfg := dispatchConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	if cfg.queue != "" {
		return m.queue.DispatchTo(ctx, cfg.queue, job)
	}
	return m.queue.Dispatch(ctx, job)
}

// DispatchAfter queues a background job to run after a delay.
func (m *Manager) DispatchAfter(ctx context.Context, delay time.Duration, job Job, opts ...DispatchOption) error {
	if job == nil {
		return ErrNilJob
	}

	cfg := dispatchConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	if cfg.queue != "" {
		return m.queue.LaterTo(ctx, cfg.queue, delay, job)
	}
	return m.queue.Later(ctx, delay, job)
}

// Run executes a synchronous workflow step with optional retry policy.
func (m *Manager) Run(ctx context.Context, name string, task TaskFunc, opts ...RunOption) error {
	if task == nil {
		return ErrNilTask
	}

	cfg := runConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	run := func(runCtx context.Context) error {
		return task(runCtx)
	}
	if cfg.retryPolicy == nil {
		if err := run(ctx); err != nil {
			return wrapTaskError(name, err)
		}
		return nil
	}

	if err := runWithRetry(ctx, run, *cfg.retryPolicy); err != nil {
		return wrapTaskError(name, err)
	}
	return nil
}

// Schedule starts a schedule plan that can be configured and registered.
func (m *Manager) Schedule(name string, task TaskFunc) *SchedulePlan {
	return &SchedulePlan{
		manager: m,
		event: Call(strings.TrimSpace(name), func(ctx context.Context) error {
			if task == nil {
				return ErrNilTask
			}
			return task(ctx)
		}),
	}
}

// RunDue executes any due scheduled work at the provided timestamp.
func (m *Manager) RunDue(ctx context.Context, now time.Time) error {
	for _, event := range m.scheduler.DueEvents(now) {
		if err := event.Run(ctx); err != nil {
			return fmt.Errorf("workflow schedule %q failed: %w", event.Name(), err)
		}
	}
	return nil
}

// StartScheduler starts the scheduler loop.
func (m *Manager) StartScheduler(ctx context.Context) {
	m.scheduler.Start(ctx)
}

// StopScheduler stops the scheduler loop.
func (m *Manager) StopScheduler() {
	m.scheduler.Stop()
}

// ClearSchedules removes registered schedules.
func (m *Manager) ClearSchedules() {
	m.scheduler.Clear()
}

// NewWorker creates a worker bound to the manager's default driver.
func (m *Manager) NewWorker(config WorkerConfig) *Worker {
	return m.queue.NewWorker(config)
}

// StartWorker creates and starts a worker.
func (m *Manager) StartWorker(ctx context.Context, config WorkerConfig) (*Worker, error) {
	worker := m.NewWorker(config)
	if err := worker.Start(ctx); err != nil {
		return nil, err
	}
	return worker, nil
}

type dispatchConfig struct {
	queue string
}

// DispatchOption customizes a background dispatch request.
type DispatchOption func(*dispatchConfig)

// WithQueue dispatches the job onto a named queue.
func WithQueue(name string) DispatchOption {
	return func(cfg *dispatchConfig) {
		cfg.queue = strings.TrimSpace(name)
	}
}

type runConfig struct {
	retryPolicy *RetryPolicy
}

// RunOption customizes a synchronous workflow step.
type RunOption func(*runConfig)

// WithRetryPolicy applies retry behavior to a synchronous workflow step.
func WithRetryPolicy(policy RetryPolicy) RunOption {
	return func(cfg *runConfig) {
		policyCopy := policy.normalize()
		cfg.retryPolicy = &policyCopy
	}
}

// DefaultRetryPolicy returns the framework's default retry behavior.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		ShouldRetry:  func(err error) bool { return err != nil },
	}
}

func (p RetryPolicy) normalize() RetryPolicy {
	defaults := DefaultRetryPolicy()
	if p.MaxAttempts < 1 {
		p.MaxAttempts = defaults.MaxAttempts
	}
	if p.InitialDelay < 0 {
		p.InitialDelay = defaults.InitialDelay
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = defaults.MaxDelay
	}
	if p.Multiplier <= 0 {
		p.Multiplier = defaults.Multiplier
	}
	if p.Jitter < 0 {
		p.Jitter = defaults.Jitter
	}
	if p.ShouldRetry == nil {
		p.ShouldRetry = defaults.ShouldRetry
	}
	return p
}

func runWithRetry(ctx context.Context, task TaskFunc, policy RetryPolicy) error {
	policy = policy.normalize()

	var lastErr error
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := task(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		if !policy.ShouldRetry(err) {
			return err
		}

		if attempt < policy.MaxAttempts-1 {
			delay := calculateRetryDelay(attempt, policy)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return errors.Join(ErrMaxAttemptsExceeded, lastErr)
}

func calculateRetryDelay(attempt int, policy RetryPolicy) time.Duration {
	delay := float64(policy.InitialDelay) * math.Pow(policy.Multiplier, float64(attempt))
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	if policy.Jitter > 0 {
		jitter := delay * policy.Jitter * (rand.Float64()*2 - 1)
		delay += jitter
	}
	return time.Duration(delay)
}

func wrapTaskError(name string, err error) error {
	if strings.TrimSpace(name) == "" {
		return err
	}
	return fmt.Errorf("workflow task %q failed: %w", name, err)
}

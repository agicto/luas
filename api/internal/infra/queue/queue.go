package queue

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/capabilities/workflow"
)

// Job represents a queueable job.
type Job = workflow.Job

// JobWithQueue allows a job to specify its queue.
type JobWithQueue = workflow.JobWithQueue

// JobWithDelay allows a job to specify a delay.
type JobWithDelay = workflow.JobWithDelay

// JobWithRetry allows a job to specify retry behavior.
type JobWithRetry = workflow.JobWithRetry

type JobWithIdempotencyKey = workflow.JobWithIdempotencyKey

// Driver defines the queue driver interface.
type Driver = workflow.Driver

var (
	// ErrQueueEmpty indicates that a non-blocking queue has no jobs available.
	ErrQueueEmpty = workflow.ErrQueueEmpty
	// ErrDriverClosed indicates that the queue driver no longer accepts work.
	ErrDriverClosed = workflow.ErrDriverClosed
)

// JobPayload represents serialized job data.
type JobPayload = workflow.JobPayload

// Manager manages queue operations.
type Manager = workflow.QueueManager

// SyncDriver executes jobs synchronously.
type SyncDriver = workflow.SyncDriver

// MemoryDriver stores jobs in memory for worker processing.
type MemoryDriver = workflow.MemoryDriver

type PostgresDriver = workflow.PostgresDriver

// WorkerConfig holds worker configuration.
type WorkerConfig = workflow.WorkerConfig

// Worker processes jobs from a queue.
type Worker = workflow.Worker

// Global returns the global queue manager.
func Global() *Manager {
	return workflow.GlobalQueue()
}

// New creates a queue manager with the default sync driver.
func New() *Manager {
	return workflow.NewQueueManager()
}

// NewSyncDriver creates a sync driver.
func NewSyncDriver() *SyncDriver {
	return workflow.NewSyncDriver()
}

// NewMemoryDriver creates a memory driver.
func NewMemoryDriver(bufferSize int) *MemoryDriver {
	return workflow.NewMemoryDriver(bufferSize)
}

// NewPostgresDriver creates the durable PostgreSQL queue driver.
func NewPostgresDriver(db *gorm.DB) (*PostgresDriver, error) {
	return workflow.NewPostgresDriver(db)
}

// Dispatch dispatches a job using the global manager.
func Dispatch(ctx context.Context, job Job) error {
	return workflow.Dispatch(ctx, job)
}

// DispatchTo dispatches a job to a specific queue.
func DispatchTo(ctx context.Context, queue string, job Job) error {
	return workflow.DispatchTo(ctx, queue, job)
}

// DispatchTask dispatches a job and returns its stable task ID.
func DispatchTask(ctx context.Context, job Job) (string, error) {
	return workflow.DispatchTask(ctx, job)
}

// DispatchTaskTo dispatches a job to a named queue and returns its stable task ID.
func DispatchTaskTo(ctx context.Context, queue string, job Job) (string, error) {
	return workflow.DispatchTaskTo(ctx, queue, job)
}

// Later dispatches a job with a delay.
func Later(ctx context.Context, delay time.Duration, job Job) error {
	return workflow.Later(ctx, delay, job)
}

// LaterTo dispatches a job to a specific queue with a delay.
func LaterTo(ctx context.Context, queue string, delay time.Duration, job Job) error {
	return workflow.LaterTo(ctx, queue, delay, job)
}

// RegisterJob registers a job type.
func RegisterJob(job Job) {
	workflow.RegisterJob(job)
}

// Size returns the size of a queue.
func Size(ctx context.Context, queue string) (int64, error) {
	return workflow.Size(ctx, queue)
}

// Clear clears a queue.
func Clear(ctx context.Context, queue string) error {
	return workflow.Clear(ctx, queue)
}

// Cancel requests cancellation of a durable task.
func Cancel(ctx context.Context, taskID string) error {
	return workflow.Cancel(ctx, taskID)
}

// DefaultWorkerConfig returns default worker configuration.
func DefaultWorkerConfig() WorkerConfig {
	return workflow.DefaultWorkerConfig()
}

// NewWorker creates a worker.
func NewWorker(config WorkerConfig) *Worker {
	return workflow.NewWorker(config)
}

// Daemon starts a worker as a daemon process.
func Daemon(ctx context.Context, config WorkerConfig) error {
	return workflow.Daemon(ctx, config)
}

// Work processes jobs from the default queue.
func Work(ctx context.Context, queue string, concurrency int) error {
	return workflow.Work(ctx, queue, concurrency)
}

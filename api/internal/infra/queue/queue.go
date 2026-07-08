package queue

import (
	"context"
	"time"

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

// Driver defines the queue driver interface.
type Driver = workflow.Driver

// JobPayload represents serialized job data.
type JobPayload = workflow.JobPayload

// Manager manages queue operations.
type Manager = workflow.QueueManager

// SyncDriver executes jobs synchronously.
type SyncDriver = workflow.SyncDriver

// MemoryDriver stores jobs in memory for worker processing.
type MemoryDriver = workflow.MemoryDriver

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

// Dispatch dispatches a job using the global manager.
func Dispatch(ctx context.Context, job Job) error {
	return workflow.Dispatch(ctx, job)
}

// DispatchTo dispatches a job to a specific queue.
func DispatchTo(ctx context.Context, queue string, job Job) error {
	return workflow.DispatchTo(ctx, queue, job)
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

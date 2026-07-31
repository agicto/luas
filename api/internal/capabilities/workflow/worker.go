package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	// Queue name to process
	Queue string

	// Number of concurrent workers
	Concurrency int

	// Sleep duration when queue is empty
	Sleep time.Duration

	// Maximum job processing time
	Timeout time.Duration

	// Stop after processing this many jobs (0 = unlimited)
	MaxJobs int

	// LeaseDuration bounds ownership of a durable task.
	LeaseDuration time.Duration

	// HeartbeatInterval renews a durable lease and observes cancellation.
	HeartbeatInterval time.Duration

	// ObserveInterval controls durable queue depth and lag snapshots.
	ObserveInterval time.Duration

	// ObserveQueue receives bounded-cardinality durable queue snapshots.
	ObserveQueue func(queue string, stats QueueStats)

	// Handler for failed jobs
	FailedJobHandler func(ctx context.Context, payload *JobPayload, err error)

	// Handler called before job processing
	BeforeJob func(ctx context.Context, payload *JobPayload)

	// Handler called after job processing
	AfterJob func(ctx context.Context, payload *JobPayload, err error)
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Queue:             "default",
		Concurrency:       1,
		Sleep:             time.Second,
		Timeout:           60 * time.Second,
		MaxJobs:           0,
		LeaseDuration:     90 * time.Second,
		HeartbeatInterval: 20 * time.Second,
		ObserveInterval:   15 * time.Second,
	}
}

// Worker processes jobs from a queue
type Worker struct {
	config   WorkerConfig
	manager  *QueueManager
	driver   Driver
	stop     chan struct{}
	wg       sync.WaitGroup
	running  bool
	stopping bool
	cancel   context.CancelFunc
	mu       sync.Mutex
}

// NewWorker creates a new worker
func NewWorker(config WorkerConfig) *Worker {
	return GlobalQueue().NewWorker(config)
}

// NewWorker creates a new worker bound to this queue manager.
func (m *QueueManager) NewWorker(config WorkerConfig) *Worker {
	if config.Concurrency < 1 {
		config.Concurrency = 1
	}
	if config.Sleep == 0 {
		config.Sleep = time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.LeaseDuration <= 0 {
		config.LeaseDuration = config.Timeout + 30*time.Second
	}
	if config.HeartbeatInterval <= 0 || config.HeartbeatInterval >= config.LeaseDuration {
		config.HeartbeatInterval = config.LeaseDuration / 3
	}
	if config.ObserveInterval <= 0 {
		config.ObserveInterval = 15 * time.Second
	}

	return &Worker{
		config:  config,
		manager: m,
		driver:  m.DefaultDriver(),
		stop:    make(chan struct{}),
	}
}

// SetDriver sets the driver for the worker
func (w *Worker) SetDriver(driver Driver) {
	if driver == nil {
		return
	}
	w.mu.Lock()
	w.driver = driver
	w.mu.Unlock()
}

func (w *Worker) currentDriver() Driver {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.driver
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	stop := make(chan struct{})
	w.stop = stop
	w.cancel = cancel
	w.running = true
	w.stopping = false

	for i := 0; i < w.config.Concurrency; i++ {
		w.wg.Add(1)
		go w.work(runCtx, stop, i)
	}
	if durable, ok := w.driver.(DurableDriver); ok && w.config.ObserveQueue != nil {
		w.wg.Add(1)
		go w.observe(runCtx, stop, durable)
	}
	w.mu.Unlock()

	return nil
}

func (w *Worker) observe(ctx context.Context, stop <-chan struct{}, driver DurableDriver) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.config.ObserveInterval)
	defer ticker.Stop()
	for {
		stats, err := driver.Stats(ctx, w.config.Queue, time.Now().UTC())
		if err == nil {
			w.config.ObserveQueue(w.config.Queue, stats)
		}
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
		}
	}
}

// Stop stops the worker gracefully
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	if w.stopping {
		w.mu.Unlock()
		w.wg.Wait()
		return
	}
	w.stopping = true
	cancel := w.cancel
	stop := w.stop
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	close(stop)
	w.wg.Wait()

	w.mu.Lock()
	w.running = false
	w.stopping = false
	w.cancel = nil
	w.mu.Unlock()
}

// Wait blocks until all worker goroutines exit.
func (w *Worker) Wait() {
	w.wg.Wait()
}

// work is the main worker loop. id is reserved for future per-worker logging
// labels.
func (w *Worker) work(ctx context.Context, stop <-chan struct{}, _ int) {
	defer w.wg.Done()

	jobsProcessed := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		default:
			// Check max jobs
			if w.config.MaxJobs > 0 && jobsProcessed >= w.config.MaxJobs {
				return
			}

			driver := w.currentDriver()
			if durable, ok := driver.(DurableDriver); ok {
				claim, err := durable.Claim(ctx, w.config.Queue, w.config.LeaseDuration)
				if err != nil {
					if ctx.Err() != nil || errors.Is(err, ErrDriverClosed) {
						return
					}
					if !waitForWorker(ctx, stop, w.config.Sleep) {
						return
					}
					continue
				}
				w.processDurableJob(ctx, durable, claim)
				jobsProcessed++
				continue
			}

			// Try to get a job
			payload, err := driver.Pop(ctx, w.config.Queue)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, ErrDriverClosed) {
					return
				}
				if !waitForWorker(ctx, stop, w.config.Sleep) {
					return
				}
				continue
			}

			// Process the job
			w.processJob(ctx, payload)
			jobsProcessed++
		}
	}
}

func waitForWorker(ctx context.Context, stop <-chan struct{}, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	case <-stop:
		return false
	}
}

// processJob processes a single job
func (w *Worker) processJob(ctx context.Context, data []byte) {
	payload, retryDelay, retry, err := w.executeJob(ctx, data, 0)
	if payload == nil {
		return
	}
	if err != nil && retry {
		if newPayload, marshalErr := json.Marshal(payload); marshalErr == nil {
			_ = w.currentDriver().PushDelayed(ctx, w.config.Queue, newPayload, retryDelay) //nolint:errcheck
		}
		return
	}
	if err != nil {
		w.handleFailedJob(ctx, payload, err)
	}
}

func (w *Worker) processDurableJob(ctx context.Context, driver DurableDriver, claim *DurableClaim) {
	jobCtx, cancel := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(w.config.HeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-jobCtx.Done():
				return
			case <-ticker.C:
				canceled, err := driver.Heartbeat(jobCtx, claim, w.config.LeaseDuration)
				if err != nil || canceled {
					cancel()
					return
				}
			}
		}
	}()

	payload, retryDelay, retry, err := w.executeJob(jobCtx, claim.Payload, claim.Attempts)
	cancel()
	<-heartbeatDone
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer finishCancel()
	if payload == nil {
		_ = driver.Fail(finishCtx, claim, err) //nolint:errcheck
		return
	}
	if err == nil {
		_ = driver.Complete(finishCtx, claim) //nolint:errcheck
		return
	}
	if retry {
		serialized, marshalErr := json.Marshal(payload)
		if marshalErr == nil {
			_ = driver.Retry(finishCtx, claim, serialized, time.Now().UTC().Add(retryDelay), err) //nolint:errcheck
			return
		}
		err = errors.Join(err, marshalErr)
	}
	_ = driver.Fail(finishCtx, claim, err) //nolint:errcheck
	w.handleFailedJob(finishCtx, payload, err)
}

func (w *Worker) executeJob(ctx context.Context, data []byte, claimedAttempt int) (*JobPayload, time.Duration, bool, error) {
	var payload JobPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("Failed to unmarshal job payload: %v", err)
		return nil, 0, false, err
	}
	carrier := propagation.MapCarrier{"traceparent": payload.TraceParent, "tracestate": payload.TraceState}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	// Call before handler
	if w.config.BeforeJob != nil {
		w.config.BeforeJob(ctx, &payload)
	}

	// Create job instance
	job, err := w.manager.createJob(payload.Type)
	if err != nil {
		log.Printf("Failed to create job instance: %v", err)
		return &payload, 0, false, err
	}

	// Unmarshal job data
	if unmarshalErr := json.Unmarshal(payload.Data, job); unmarshalErr != nil {
		log.Printf("Failed to unmarshal job data: %v", unmarshalErr)
		return &payload, 0, false, unmarshalErr
	}

	// Create context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, w.config.Timeout)
	defer cancel()

	// Execute the job
	if claimedAttempt > 0 {
		payload.Attempts = claimedAttempt
	} else {
		payload.Attempts++
	}
	err = job.Handle(jobCtx)

	// Call after handler
	if w.config.AfterJob != nil {
		w.config.AfterJob(ctx, &payload, err)
	}

	if err == nil {
		return &payload, 0, false, nil
	}
	retryDelay := time.Second * time.Duration(payload.Attempts)
	if jr, ok := job.(JobWithRetry); ok {
		retryDelay = jr.RetryDelay()
	}
	return &payload, retryDelay, payload.Attempts < payload.MaxRetries, err
}

// handleFailedJob handles a failed job
func (w *Worker) handleFailedJob(ctx context.Context, payload *JobPayload, err error) {
	if w.config.FailedJobHandler != nil {
		w.config.FailedJobHandler(ctx, payload, err)
	} else {
		log.Printf("Job %s failed after %d attempts: %v", payload.Type, payload.Attempts, err)
	}
}

// ProcessNext processes the next job in queue (useful for testing)
func (w *Worker) ProcessNext(ctx context.Context) error {
	payload, err := w.driver.Pop(ctx, w.config.Queue)
	if err != nil {
		return err
	}

	w.processJob(ctx, payload)
	return nil
}

// Daemon starts the worker as a daemon process
func Daemon(ctx context.Context, config WorkerConfig) error {
	worker := NewWorker(config)
	return worker.Start(ctx)
}

// Work processes jobs from the default queue
func Work(ctx context.Context, queue string, concurrency int) error {
	config := DefaultWorkerConfig()
	config.Queue = queue
	config.Concurrency = concurrency

	worker := NewWorker(config)
	return worker.Start(ctx)
}

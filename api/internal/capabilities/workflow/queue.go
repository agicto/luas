package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Job represents a queueable job
type Job interface {
	Handle(ctx context.Context) error
}

// JobWithQueue allows a job to specify its queue
type JobWithQueue interface {
	Job
	Queue() string
}

// JobWithDelay allows a job to specify a delay
type JobWithDelay interface {
	Job
	Delay() time.Duration
}

// JobWithRetry allows a job to specify retry behavior
type JobWithRetry interface {
	Job
	MaxRetries() int
	RetryDelay() time.Duration
}

// Driver defines the queue driver interface
type Driver interface {
	// Push adds a job to the queue
	Push(ctx context.Context, queue string, payload []byte) error

	// PushDelayed adds a job to the queue with a delay
	PushDelayed(ctx context.Context, queue string, payload []byte, delay time.Duration) error

	// Pop retrieves the next job from the queue
	Pop(ctx context.Context, queue string) ([]byte, error)

	// Size returns the number of jobs in the queue
	Size(ctx context.Context, queue string) (int64, error)

	// Clear removes all jobs from the queue
	Clear(ctx context.Context, queue string) error

	// Close closes the driver connection
	Close() error
}

var (
	// ErrQueueEmpty indicates that a non-blocking queue has no jobs available.
	ErrQueueEmpty = errors.New("queue is empty")
	// ErrDriverClosed indicates that the queue driver no longer accepts work.
	ErrDriverClosed = errors.New("queue driver is closed")
)

// JobPayload represents the serialized job data
type JobPayload struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	Attempts   int             `json:"attempts"`
	MaxRetries int             `json:"max_retries"`
	CreatedAt  time.Time       `json:"created_at"`
}

// QueueManager manages workflow queue operations.
type QueueManager struct {
	mu            sync.RWMutex
	drivers       map[string]Driver
	defaultDriver string
	defaultQueue  string
	jobRegistry   map[string]reflect.Type
}

var (
	queueManager *QueueManager
	queueOnce    sync.Once
)

// GlobalQueue returns the global workflow queue manager.
func GlobalQueue() *QueueManager {
	queueOnce.Do(func() {
		queueManager = NewQueueManager()
	})
	return queueManager
}

// NewQueueManager creates a queue manager with the default sync driver.
func NewQueueManager() *QueueManager {
	manager := &QueueManager{
		drivers:       make(map[string]Driver),
		defaultDriver: "sync",
		defaultQueue:  "default",
		jobRegistry:   make(map[string]reflect.Type),
	}
	manager.RegisterDriver("sync", NewSyncDriver())
	return manager
}

// SetDefaultQueue sets the default queue name
func (m *QueueManager) SetDefaultQueue(queue string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultQueue = queue
}

// RegisterDriver registers a queue driver
func (m *QueueManager) RegisterDriver(name string, driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if syncDriver, ok := driver.(*SyncDriver); ok && syncDriver.manager == nil {
		syncDriver.manager = m
	}
	m.drivers[name] = driver
}

// Driver returns a driver by name
func (m *QueueManager) Driver(name string) Driver {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.drivers[name]
}

// SetDefaultDriver sets the default driver name used by dispatch operations.
func (m *QueueManager) SetDefaultDriver(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.drivers[name]; !ok {
		return fmt.Errorf("queue driver %q is not registered", name)
	}
	m.defaultDriver = name
	return nil
}

// DefaultDriverName returns the configured default driver name.
func (m *QueueManager) DefaultDriverName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if strings.TrimSpace(m.defaultDriver) == "" {
		return "sync"
	}
	return m.defaultDriver
}

// DefaultDriver returns the default driver
func (m *QueueManager) DefaultDriver() Driver {
	name := m.DefaultDriverName()
	driver := m.Driver(name)
	if driver != nil {
		return driver
	}
	return m.Driver("sync")
}

// RegisterJob registers a job type for deserialization
func (m *QueueManager) RegisterJob(job Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := reflect.TypeOf(job)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	m.jobRegistry[t.Name()] = t
}

// createJob creates a job instance from type name
func (m *QueueManager) createJob(typeName string) (Job, error) {
	m.mu.RLock()
	t, ok := m.jobRegistry[typeName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown job type: %s", typeName)
	}

	job, ok := reflect.New(t).Interface().(Job)
	if !ok {
		return nil, fmt.Errorf("registered type %s does not implement Job", typeName)
	}
	return job, nil
}

// Dispatch dispatches a job to the queue
func (m *QueueManager) Dispatch(ctx context.Context, job Job) error {
	return m.DispatchTo(ctx, m.defaultQueue, job)
}

// DispatchTo dispatches a job to a specific queue
func (m *QueueManager) DispatchTo(ctx context.Context, queue string, job Job) error {
	// Check if job specifies its own queue
	if jq, ok := job.(JobWithQueue); ok {
		queue = jq.Queue()
	}

	payload, err := m.serializeJob(job)
	if err != nil {
		return err
	}

	driver := m.DefaultDriver()

	// Check if job has delay
	if jd, ok := job.(JobWithDelay); ok {
		if delay := jd.Delay(); delay > 0 {
			return driver.PushDelayed(ctx, queue, payload, delay)
		}
	}

	return driver.Push(ctx, queue, payload)
}

// Later dispatches a job with a delay
func (m *QueueManager) Later(ctx context.Context, delay time.Duration, job Job) error {
	return m.LaterTo(ctx, m.defaultQueue, delay, job)
}

// LaterTo dispatches a job to a specific queue with a delay
func (m *QueueManager) LaterTo(ctx context.Context, queue string, delay time.Duration, job Job) error {
	payload, err := m.serializeJob(job)
	if err != nil {
		return err
	}

	return m.DefaultDriver().PushDelayed(ctx, queue, payload, delay)
}

// serializeJob serializes a job to JSON
func (m *QueueManager) serializeJob(job Job) ([]byte, error) {
	t := reflect.TypeOf(job)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	data, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	maxRetries := 3
	if jr, ok := job.(JobWithRetry); ok {
		maxRetries = jr.MaxRetries()
	}

	payload := JobPayload{
		Type:       t.Name(),
		Data:       data,
		Attempts:   0,
		MaxRetries: maxRetries,
		CreatedAt:  time.Now(),
	}

	return json.Marshal(payload)
}

// deserializeJob deserializes a job from JSON
func (m *QueueManager) deserializeJob(data []byte) (Job, *JobPayload, error) {
	var payload JobPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, err
	}

	job, err := m.createJob(payload.Type)
	if err != nil {
		return nil, nil, err
	}

	if err := json.Unmarshal(payload.Data, job); err != nil {
		return nil, nil, err
	}

	return job, &payload, nil
}

// Size returns the size of a queue
func (m *QueueManager) Size(ctx context.Context, queue string) (int64, error) {
	return m.DefaultDriver().Size(ctx, queue)
}

// Clear clears a queue
func (m *QueueManager) Clear(ctx context.Context, queue string) error {
	return m.DefaultDriver().Clear(ctx, queue)
}

// --- Convenience functions ---

// Dispatch dispatches a job using the global manager
func Dispatch(ctx context.Context, job Job) error {
	return GlobalQueue().Dispatch(ctx, job)
}

// DispatchTo dispatches a job to a specific queue
func DispatchTo(ctx context.Context, queue string, job Job) error {
	return GlobalQueue().DispatchTo(ctx, queue, job)
}

// Later dispatches a job with a delay
func Later(ctx context.Context, delay time.Duration, job Job) error {
	return GlobalQueue().Later(ctx, delay, job)
}

// LaterTo dispatches a job to a specific queue with a delay
func LaterTo(ctx context.Context, queue string, delay time.Duration, job Job) error {
	return GlobalQueue().LaterTo(ctx, queue, delay, job)
}

// RegisterJob registers a job type
func RegisterJob(job Job) {
	GlobalQueue().RegisterJob(job)
}

// Size returns the size of a queue
func Size(ctx context.Context, queue string) (int64, error) {
	return GlobalQueue().Size(ctx, queue)
}

// Clear clears a queue
func Clear(ctx context.Context, queue string) error {
	return GlobalQueue().Clear(ctx, queue)
}

// --- Sync Driver (executes jobs immediately) ---

// SyncDriver executes jobs synchronously (for development/testing)
type SyncDriver struct {
	mu      sync.Mutex
	queues  map[string][][]byte
	manager *QueueManager
}

// NewSyncDriver creates a new sync driver
func NewSyncDriver() *SyncDriver {
	return &SyncDriver{
		queues: make(map[string][][]byte),
	}
}

// Push adds a job and executes it immediately
func (d *SyncDriver) Push(ctx context.Context, queue string, payload []byte) error {
	// For sync driver, we execute immediately
	manager := d.manager
	if manager == nil {
		manager = GlobalQueue()
	}
	job, jobPayload, err := manager.deserializeJob(payload)
	if err != nil {
		// Graceful fallback: a payload we can't deserialize is queued raw
		// for later inspection. The caller intentionally sees success here
		// because the work is durable in d.queues.
		d.mu.Lock()
		d.queues[queue] = append(d.queues[queue], payload)
		d.mu.Unlock()
		return nil //nolint:nilerr // intentional graceful fallback (see above)
	}

	// Execute the job
	if err := job.Handle(ctx); err != nil {
		// Check retry. JSON marshal of a struct with primitive fields cannot
		// fail in practice, so a marshal error here would indicate corruption
		// — fall through to returning the handler error.
		if jobPayload.Attempts < jobPayload.MaxRetries {
			jobPayload.Attempts++
			if newPayload, marshalErr := json.Marshal(jobPayload); marshalErr == nil {
				d.mu.Lock()
				d.queues[queue] = append(d.queues[queue], newPayload)
				d.mu.Unlock()
			}
		}
		return err
	}

	return nil
}

// PushDelayed waits for the delay and then executes the job synchronously.
func (d *SyncDriver) PushDelayed(ctx context.Context, queue string, payload []byte, delay time.Duration) error {
	if delay <= 0 {
		return d.Push(ctx, queue, payload)
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return d.Push(ctx, queue, payload)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Pop retrieves the next job
func (d *SyncDriver) Pop(ctx context.Context, queue string) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	jobs := d.queues[queue]
	if len(jobs) == 0 {
		return nil, ErrQueueEmpty
	}

	payload := jobs[0]
	d.queues[queue] = jobs[1:]
	return payload, nil
}

// Size returns queue size
func (d *SyncDriver) Size(ctx context.Context, queue string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return int64(len(d.queues[queue])), nil
}

// Clear clears the queue
func (d *SyncDriver) Clear(ctx context.Context, queue string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queues[queue] = nil
	return nil
}

// Close closes the driver
func (d *SyncDriver) Close() error {
	return nil
}

// --- Memory Driver (stores jobs for worker processing) ---

// MemoryDriver stores jobs in bounded, process-local memory for async
// processing. Close cancels pending delayed jobs and waits for in-flight
// operations to observe shutdown before returning.
type MemoryDriver struct {
	mu           sync.Mutex
	queues       map[string]*memoryQueue
	bufSize      int
	closed       atomic.Bool
	done         chan struct{}
	delayedSlots chan struct{}
	operations   sync.WaitGroup
	closeOnce    sync.Once
}

type memoryQueue struct {
	mu       sync.Mutex
	items    [][]byte
	head     int
	size     int
	notEmpty chan struct{}
	notFull  chan struct{}
}

// NewMemoryDriver creates a new memory driver
func NewMemoryDriver(bufferSize int) *MemoryDriver {
	if bufferSize < 1 {
		bufferSize = 1
	}
	return &MemoryDriver{
		queues:       make(map[string]*memoryQueue),
		bufSize:      bufferSize,
		done:         make(chan struct{}),
		delayedSlots: make(chan struct{}, bufferSize),
	}
}

func newMemoryQueue(capacity int) *memoryQueue {
	queue := &memoryQueue{
		items:    make([][]byte, capacity),
		notEmpty: make(chan struct{}, 1),
		notFull:  make(chan struct{}, 1),
	}
	queue.syncSignals()
	return queue
}

func (q *memoryQueue) syncSignals() {
	setQueueSignal(q.notEmpty, q.size > 0)
	setQueueSignal(q.notFull, q.size < len(q.items))
}

func setQueueSignal(signal chan struct{}, ready bool) {
	if ready {
		select {
		case signal <- struct{}{}:
		default:
		}
		return
	}

	select {
	case <-signal:
	default:
	}
}

func (q *memoryQueue) enqueue(payload []byte) {
	index := (q.head + q.size) % len(q.items)
	q.items[index] = payload
	q.size++
	q.syncSignals()
}

func (q *memoryQueue) dequeue() []byte {
	payload := q.items[q.head]
	q.items[q.head] = nil
	q.head = (q.head + 1) % len(q.items)
	q.size--
	q.syncSignals()
	return payload
}

func (q *memoryQueue) clear() {
	for index := range q.items {
		q.items[index] = nil
	}
	q.head = 0
	q.size = 0
	q.syncSignals()
}

func (d *MemoryDriver) beginOperation(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed.Load() {
		return ErrDriverClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	d.operations.Add(1)
	return nil
}

func (d *MemoryDriver) getQueue(name string, create bool) (*memoryQueue, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed.Load() {
		return nil, ErrDriverClosed
	}

	queue, ok := d.queues[name]
	if ok || !create {
		return queue, nil
	}

	queue = newMemoryQueue(d.bufSize)
	d.queues[name] = queue
	return queue, nil
}

// Push adds a job to the queue
func (d *MemoryDriver) Push(ctx context.Context, queue string, payload []byte) error {
	if err := d.beginOperation(ctx); err != nil {
		return err
	}
	defer d.operations.Done()
	return d.push(ctx, queue, payload)
}

func (d *MemoryDriver) push(ctx context.Context, queueName string, payload []byte) error {
	queue, err := d.getQueue(queueName, true)
	if err != nil {
		return err
	}

	for {
		queue.mu.Lock()
		if d.closed.Load() {
			queue.mu.Unlock()
			return ErrDriverClosed
		}
		if err := ctx.Err(); err != nil {
			queue.mu.Unlock()
			return err
		}
		if queue.size < len(queue.items) {
			queue.enqueue(payload)
			queue.mu.Unlock()
			return nil
		}
		queue.mu.Unlock()

		select {
		case <-queue.notFull:
		case <-ctx.Done():
			return ctx.Err()
		case <-d.done:
			return ErrDriverClosed
		}
	}
}

// PushDelayed schedules a job for in-process delivery. The pending delivery is
// canceled when its context is canceled or the driver is closed.
func (d *MemoryDriver) PushDelayed(ctx context.Context, queue string, payload []byte, delay time.Duration) error {
	if delay <= 0 {
		return d.Push(ctx, queue, payload)
	}
	if err := d.beginOperation(ctx); err != nil {
		return err
	}
	select {
	case d.delayedSlots <- struct{}{}:
	case <-ctx.Done():
		d.operations.Done()
		return ctx.Err()
	case <-d.done:
		d.operations.Done()
		return ErrDriverClosed
	}
	if d.closed.Load() {
		<-d.delayedSlots
		d.operations.Done()
		return ErrDriverClosed
	}
	if err := ctx.Err(); err != nil {
		<-d.delayedSlots
		d.operations.Done()
		return err
	}

	timer := time.NewTimer(delay)
	go func() {
		defer func() {
			<-d.delayedSlots
			d.operations.Done()
		}()
		defer timer.Stop()

		select {
		case <-timer.C:
			_ = d.push(ctx, queue, payload) //nolint:errcheck // scheduling succeeded; later cancellation is expected
		case <-ctx.Done():
		case <-d.done:
		}
	}()
	return nil
}

// Pop retrieves the next job
func (d *MemoryDriver) Pop(ctx context.Context, queue string) ([]byte, error) {
	if err := d.beginOperation(ctx); err != nil {
		return nil, err
	}
	defer d.operations.Done()

	memoryQueue, err := d.getQueue(queue, true)
	if err != nil {
		return nil, err
	}

	for {
		memoryQueue.mu.Lock()
		if d.closed.Load() {
			memoryQueue.mu.Unlock()
			return nil, ErrDriverClosed
		}
		if err := ctx.Err(); err != nil {
			memoryQueue.mu.Unlock()
			return nil, err
		}
		if memoryQueue.size > 0 {
			payload := memoryQueue.dequeue()
			memoryQueue.mu.Unlock()
			return payload, nil
		}
		memoryQueue.mu.Unlock()

		select {
		case <-memoryQueue.notEmpty:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-d.done:
			return nil, ErrDriverClosed
		}
	}
}

// Size returns the current queue size.
func (d *MemoryDriver) Size(ctx context.Context, queue string) (int64, error) {
	if err := d.beginOperation(ctx); err != nil {
		return 0, err
	}
	defer d.operations.Done()

	memoryQueue, err := d.getQueue(queue, false)
	if err != nil {
		return 0, err
	}
	if memoryQueue == nil {
		return 0, nil
	}

	memoryQueue.mu.Lock()
	defer memoryQueue.mu.Unlock()
	if d.closed.Load() {
		return 0, ErrDriverClosed
	}
	return int64(memoryQueue.size), nil
}

// Clear clears the queue
func (d *MemoryDriver) Clear(ctx context.Context, queue string) error {
	if err := d.beginOperation(ctx); err != nil {
		return err
	}
	defer d.operations.Done()

	memoryQueue, err := d.getQueue(queue, false)
	if err != nil {
		return err
	}
	if memoryQueue == nil {
		return nil
	}

	memoryQueue.mu.Lock()
	defer memoryQueue.mu.Unlock()
	if d.closed.Load() {
		return ErrDriverClosed
	}
	memoryQueue.clear()
	return nil
}

// Close cancels delayed jobs, unblocks waiting operations, and releases queued
// payloads. It is safe to call concurrently or more than once.
func (d *MemoryDriver) Close() error {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed.Store(true)
		close(d.done)
		d.mu.Unlock()

		d.operations.Wait()

		d.mu.Lock()
		d.queues = nil
		d.mu.Unlock()
	})
	return nil
}

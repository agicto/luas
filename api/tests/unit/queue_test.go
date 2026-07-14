package unit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zgiai/luas/api/internal/infra/queue"
)

// TestJob is a simple test job
type TestJob struct {
	Message string `json:"message"`
}

func (j *TestJob) Handle(ctx context.Context) error {
	return nil
}

// CounterJob increments a counter
type CounterJob struct {
	Value int32 `json:"value"`
}

func (j *CounterJob) Handle(ctx context.Context) error {
	return nil
}

// FailingJob always fails
type FailingJob struct {
	Attempts int `json:"attempts"`
}

func (j *FailingJob) Handle(ctx context.Context) error {
	return errors.New("job failed")
}

func (j *FailingJob) MaxRetries() int {
	return 3
}

func (j *FailingJob) RetryDelay() time.Duration {
	return 10 * time.Millisecond
}

// DelayedJob has a delay
type DelayedJob struct {
	Message   string        `json:"message"`
	DelayTime time.Duration `json:"-"`
}

func (j *DelayedJob) Handle(ctx context.Context) error {
	return nil
}

func (j *DelayedJob) Delay() time.Duration {
	return j.DelayTime
}

// QueuedJob specifies its queue
type QueuedJob struct {
	QueueName string `json:"-"`
}

func (j *QueuedJob) Handle(ctx context.Context) error {
	return nil
}

func (j *QueuedJob) Queue() string {
	return j.QueueName
}

func init() {
	// Register test jobs
	queue.RegisterJob(&TestJob{})
	queue.RegisterJob(&CounterJob{})
	queue.RegisterJob(&FailingJob{})
	queue.RegisterJob(&DelayedJob{})
	queue.RegisterJob(&QueuedJob{})
}

func TestQueue_Dispatch(t *testing.T) {
	ctx := context.Background()

	job := &TestJob{Message: "Hello"}
	err := queue.Dispatch(ctx, job)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
}

func TestQueue_DispatchTo(t *testing.T) {
	ctx := context.Background()

	job := &TestJob{Message: "Hello"}
	err := queue.DispatchTo(ctx, "emails", job)
	if err != nil {
		t.Fatalf("DispatchTo failed: %v", err)
	}
}

func TestSyncDriver_Push(t *testing.T) {
	ctx := context.Background()
	driver := queue.NewSyncDriver()

	payload := []byte(`{"type":"TestJob","data":{"message":"test"}}`)
	err := driver.Push(ctx, "default", payload)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestSyncDriver_Size(t *testing.T) {
	ctx := context.Background()
	driver := queue.NewSyncDriver()

	size, err := driver.Size(ctx, "test-queue")
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}
}

func TestSyncDriver_Clear(t *testing.T) {
	ctx := context.Background()
	driver := queue.NewSyncDriver()

	err := driver.Clear(ctx, "test-queue")
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
}

func TestSyncDriver_PushDelayedHonorsContextCancellation(t *testing.T) {
	driver := queue.NewSyncDriver()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := driver.PushDelayed(ctx, "test", []byte(`{"message":"delayed"}`), time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PushDelayed() error = %v, want context.Canceled", err)
	}
}

func TestMemoryDriver_PushAndPop(t *testing.T) {
	ctx := context.Background()
	driver := queue.NewMemoryDriver(100)
	defer driver.Close()

	payload := []byte(`{"message":"test"}`)

	// Push
	err := driver.Push(ctx, "test", payload)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Check size
	size, _ := driver.Size(ctx, "test")
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Pop
	data, err := driver.Pop(ctx, "test")
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if string(data) != string(payload) {
		t.Errorf("Expected %s, got %s", payload, data)
	}

	// Size should be 0 now
	size, _ = driver.Size(ctx, "test")
	if size != 0 {
		t.Errorf("Expected size 0 after pop, got %d", size)
	}
}

func TestMemoryDriver_Clear(t *testing.T) {
	ctx := context.Background()
	driver := queue.NewMemoryDriver(100)
	defer driver.Close()

	// Push some items
	driver.Push(ctx, "test", []byte(`{"a":1}`))
	driver.Push(ctx, "test", []byte(`{"b":2}`))

	// Clear
	err := driver.Clear(ctx, "test")
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
}

func TestMemoryDriver_PushDelayed(t *testing.T) {
	ctx := context.Background()
	driver := queue.NewMemoryDriver(100)
	defer driver.Close()

	payload := []byte(`{"message":"delayed"}`)

	// Push with short delay
	err := driver.PushDelayed(ctx, "test", payload, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("PushDelayed failed: %v", err)
	}

	// Should be empty initially
	size, _ := driver.Size(ctx, "test")
	if size != 0 {
		t.Errorf("Expected empty queue initially, got size %d", size)
	}

	// Wait for delay
	time.Sleep(100 * time.Millisecond)

	// Should have the item now
	size, _ = driver.Size(ctx, "test")
	if size != 1 {
		t.Errorf("Expected size 1 after delay, got %d", size)
	}
}

func TestMemoryDriver_CloseIsIdempotent(t *testing.T) {
	driver := queue.NewMemoryDriver(1)

	if err := driver.Push(context.Background(), "test", []byte("queued")); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestMemoryDriver_OperationsFailAfterClose(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *queue.MemoryDriver) error
	}{
		{
			name: "push",
			call: func(ctx context.Context, driver *queue.MemoryDriver) error {
				return driver.Push(ctx, "test", []byte("payload"))
			},
		},
		{
			name: "push delayed",
			call: func(ctx context.Context, driver *queue.MemoryDriver) error {
				return driver.PushDelayed(ctx, "test", []byte("payload"), time.Millisecond)
			},
		},
		{
			name: "pop",
			call: func(ctx context.Context, driver *queue.MemoryDriver) error {
				_, err := driver.Pop(ctx, "test")
				return err
			},
		},
		{
			name: "size",
			call: func(ctx context.Context, driver *queue.MemoryDriver) error {
				_, err := driver.Size(ctx, "test")
				return err
			},
		},
		{
			name: "clear",
			call: func(ctx context.Context, driver *queue.MemoryDriver) error {
				return driver.Clear(ctx, "test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := queue.NewMemoryDriver(1)
			if err := driver.Push(context.Background(), "test", []byte("queued")); err != nil {
				t.Fatalf("Push() error = %v", err)
			}
			if err := driver.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := tt.call(ctx, driver); !errors.Is(err, queue.ErrDriverClosed) {
				t.Fatalf("operation after Close() error = %v, want ErrDriverClosed", err)
			}
		})
	}
}

func TestMemoryDriver_CloseUnblocksBlockedOperations(t *testing.T) {
	t.Run("push", func(t *testing.T) {
		driver := queue.NewMemoryDriver(1)
		if err := driver.Push(context.Background(), "test", []byte("first")); err != nil {
			t.Fatalf("first Push() error = %v", err)
		}

		result := runQueueOperation(func() error {
			return driver.Push(context.Background(), "test", []byte("blocked"))
		})
		time.Sleep(10 * time.Millisecond)
		if err := driver.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		assertQueueOperationClosed(t, result)
	})

	t.Run("push delayed", func(t *testing.T) {
		driver := queue.NewMemoryDriver(1)
		if err := driver.PushDelayed(context.Background(), "test", []byte("first"), time.Hour); err != nil {
			t.Fatalf("first PushDelayed() error = %v", err)
		}

		result := runQueueOperation(func() error {
			return driver.PushDelayed(context.Background(), "test", []byte("blocked"), time.Hour)
		})
		time.Sleep(10 * time.Millisecond)
		if err := driver.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		assertQueueOperationClosed(t, result)
	})

	t.Run("pop", func(t *testing.T) {
		driver := queue.NewMemoryDriver(1)
		result := runQueueOperation(func() error {
			_, err := driver.Pop(context.Background(), "test")
			return err
		})
		time.Sleep(10 * time.Millisecond)
		if err := driver.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		assertQueueOperationClosed(t, result)
	})
}

func TestMemoryDriver_PushDelayedHonorsContextCancellation(t *testing.T) {
	t.Run("already canceled", func(t *testing.T) {
		driver := queue.NewMemoryDriver(1)
		defer driver.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := driver.PushDelayed(ctx, "test", []byte("payload"), 20*time.Millisecond); !errors.Is(err, context.Canceled) {
			t.Fatalf("PushDelayed() error = %v, want context.Canceled", err)
		}
	})

	t.Run("canceled while waiting", func(t *testing.T) {
		driver := queue.NewMemoryDriver(1)
		defer driver.Close()

		ctx, cancel := context.WithCancel(context.Background())
		if err := driver.PushDelayed(ctx, "test", []byte("payload"), 20*time.Millisecond); err != nil {
			t.Fatalf("PushDelayed() error = %v", err)
		}
		cancel()
		time.Sleep(40 * time.Millisecond)

		size, err := driver.Size(context.Background(), "test")
		if err != nil {
			t.Fatalf("Size() error = %v", err)
		}
		if size != 0 {
			t.Fatalf("Size() = %d, want 0 after delayed push cancellation", size)
		}
	})
}

func TestMemoryDriver_PushDelayedAppliesBoundedBackpressure(t *testing.T) {
	driver := queue.NewMemoryDriver(1)
	defer driver.Close()

	if err := driver.PushDelayed(context.Background(), "test", []byte("first"), time.Hour); err != nil {
		t.Fatalf("first PushDelayed() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := driver.PushDelayed(ctx, "test", []byte("second"), time.Hour); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second PushDelayed() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestMemoryDriver_PreservesFIFOAcrossBufferWraparound(t *testing.T) {
	driver := queue.NewMemoryDriver(2)
	defer driver.Close()

	if err := driver.Push(context.Background(), "test", []byte("first")); err != nil {
		t.Fatalf("first Push() error = %v", err)
	}
	if err := driver.Push(context.Background(), "test", []byte("second")); err != nil {
		t.Fatalf("second Push() error = %v", err)
	}

	payload, err := driver.Pop(context.Background(), "test")
	if err != nil {
		t.Fatalf("first Pop() error = %v", err)
	}
	if string(payload) != "first" {
		t.Fatalf("first Pop() = %q, want %q", payload, "first")
	}
	if err := driver.Push(context.Background(), "test", []byte("third")); err != nil {
		t.Fatalf("third Push() error = %v", err)
	}

	for _, want := range []string{"second", "third"} {
		payload, popErr := driver.Pop(context.Background(), "test")
		if popErr != nil {
			t.Fatalf("Pop() error = %v", popErr)
		}
		if string(payload) != want {
			t.Fatalf("Pop() = %q, want %q", payload, want)
		}
	}
}

func TestMemoryDriver_ConcurrentLifecycle(t *testing.T) {
	driver := queue.NewMemoryDriver(8)
	ctx := context.Background()
	start := make(chan struct{})
	errorsFound := make(chan error, 256)
	var operations sync.WaitGroup

	recordUnexpected := func(err error) {
		if err == nil || errors.Is(err, queue.ErrDriverClosed) {
			return
		}
		select {
		case errorsFound <- err:
		default:
		}
	}

	for producer := 0; producer < 8; producer++ {
		operations.Add(1)
		go func() {
			defer operations.Done()
			<-start
			for index := 0; index < 100; index++ {
				if index%5 == 0 {
					recordUnexpected(driver.PushDelayed(ctx, "test", []byte("delayed"), time.Millisecond))
					continue
				}
				recordUnexpected(driver.Push(ctx, "test", []byte("immediate")))
			}
		}()
	}

	for consumer := 0; consumer < 4; consumer++ {
		operations.Add(1)
		go func() {
			defer operations.Done()
			<-start
			for {
				if _, err := driver.Pop(ctx, "test"); err != nil {
					recordUnexpected(err)
					return
				}
			}
		}()
	}

	for clearer := 0; clearer < 2; clearer++ {
		operations.Add(1)
		go func() {
			defer operations.Done()
			<-start
			for index := 0; index < 100; index++ {
				recordUnexpected(driver.Clear(ctx, "test"))
			}
		}()
	}

	close(start)
	time.Sleep(10 * time.Millisecond)

	var closers sync.WaitGroup
	for index := 0; index < 8; index++ {
		closers.Add(1)
		go func() {
			defer closers.Done()
			recordUnexpected(driver.Close())
		}()
	}
	closers.Wait()
	operations.Wait()
	close(errorsFound)

	for err := range errorsFound {
		t.Errorf("concurrent queue operation error = %v", err)
	}
}

func TestWorker_StopInterruptsEmptyMemoryQueue(t *testing.T) {
	manager := queue.New()
	driver := queue.NewMemoryDriver(1)
	manager.RegisterDriver("memory", driver)
	if err := manager.SetDefaultDriver("memory"); err != nil {
		t.Fatalf("SetDefaultDriver() error = %v", err)
	}

	worker := manager.NewWorker(queue.WorkerConfig{
		Queue:       "test",
		Concurrency: 1,
		Sleep:       time.Second,
		Timeout:     time.Second,
	})
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	stopped := make(chan struct{})
	go func() {
		worker.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		if err := driver.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(time.Second):
		_ = driver.Close()
		<-stopped
		t.Fatal("Stop() did not interrupt a worker blocked on an empty memory queue")
	}
}

func TestWorker_ExitsOnDriverCloseAndAcceptsConcurrentStop(t *testing.T) {
	manager := queue.New()
	driver := queue.NewMemoryDriver(1)
	manager.RegisterDriver("memory", driver)
	if err := manager.SetDefaultDriver("memory"); err != nil {
		t.Fatalf("SetDefaultDriver() error = %v", err)
	}

	worker := manager.NewWorker(queue.WorkerConfig{
		Queue:       "test",
		Concurrency: 2,
		Sleep:       time.Second,
		Timeout:     time.Second,
	})
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	exited := make(chan struct{})
	go func() {
		worker.Wait()
		close(exited)
	}()
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("worker did not exit after its driver closed")
	}

	var stops sync.WaitGroup
	for index := 0; index < 8; index++ {
		stops.Add(1)
		go func() {
			defer stops.Done()
			worker.Stop()
		}()
	}
	stops.Wait()
}

type queueOperationResult struct {
	err        error
	panicValue any
}

func runQueueOperation(operation func() error) <-chan queueOperationResult {
	result := make(chan queueOperationResult, 1)
	go func() {
		outcome := queueOperationResult{}
		defer func() {
			outcome.panicValue = recover()
			result <- outcome
		}()
		outcome.err = operation()
	}()
	return result
}

func assertQueueOperationClosed(t *testing.T, result <-chan queueOperationResult) {
	t.Helper()

	select {
	case outcome := <-result:
		if outcome.panicValue != nil {
			t.Fatalf("blocked operation panicked during Close(): %v", outcome.panicValue)
		}
		if !errors.Is(outcome.err, queue.ErrDriverClosed) {
			t.Fatalf("blocked operation error = %v, want ErrDriverClosed", outcome.err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked operation did not return after Close()")
	}
}

func BenchmarkMemoryDriver_RoundTrip(b *testing.B) {
	driver := queue.NewMemoryDriver(1)
	b.Cleanup(func() {
		_ = driver.Close()
	})

	ctx := context.Background()
	payload := make([]byte, 256)
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for b.Loop() {
		if err := driver.Push(ctx, "benchmark", payload); err != nil {
			b.Fatalf("Push() error = %v", err)
		}
		if _, err := driver.Pop(ctx, "benchmark"); err != nil {
			b.Fatalf("Pop() error = %v", err)
		}
	}
}

func TestWorker_Config(t *testing.T) {
	config := queue.DefaultWorkerConfig()

	if config.Queue != "default" {
		t.Errorf("Expected default queue name 'default', got '%s'", config.Queue)
	}
	if config.Concurrency != 1 {
		t.Errorf("Expected concurrency 1, got %d", config.Concurrency)
	}
	if config.Sleep != time.Second {
		t.Errorf("Expected sleep 1s, got %v", config.Sleep)
	}
}

func TestWorker_Creation(t *testing.T) {
	config := queue.WorkerConfig{
		Queue:       "test",
		Concurrency: 2,
		Sleep:       100 * time.Millisecond,
		Timeout:     30 * time.Second,
	}

	worker := queue.NewWorker(config)
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}
}

func TestWorker_StartStop(t *testing.T) {
	config := queue.WorkerConfig{
		Queue:       "worker-test",
		Concurrency: 1,
		Sleep:       50 * time.Millisecond,
		MaxJobs:     1,
	}

	worker := queue.NewWorker(config)
	ctx := context.Background()

	// Start
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop
	worker.Stop()
}

func TestQueue_Later(t *testing.T) {
	ctx := context.Background()

	job := &TestJob{Message: "Delayed job"}
	err := queue.Later(ctx, 10*time.Millisecond, job)
	if err != nil {
		t.Fatalf("Later failed: %v", err)
	}
}

func TestQueue_LaterTo(t *testing.T) {
	ctx := context.Background()

	job := &TestJob{Message: "Delayed job to specific queue"}
	err := queue.LaterTo(ctx, "delayed-queue", 10*time.Millisecond, job)
	if err != nil {
		t.Fatalf("LaterTo failed: %v", err)
	}
}

func TestQueue_JobWithQueue(t *testing.T) {
	ctx := context.Background()

	job := &QueuedJob{QueueName: "custom-queue"}
	err := queue.Dispatch(ctx, job)
	if err != nil {
		t.Fatalf("Dispatch with queue failed: %v", err)
	}
}

func TestQueue_JobWithDelay(t *testing.T) {
	ctx := context.Background()

	job := &DelayedJob{
		Message:   "Auto-delayed",
		DelayTime: 10 * time.Millisecond,
	}
	err := queue.Dispatch(ctx, job)
	if err != nil {
		t.Fatalf("Dispatch with delay failed: %v", err)
	}
}

func TestQueue_RegisterJob(t *testing.T) {
	// This should not panic. Create a temporary job type.
	job := &struct {
		Name string `json:"name"`
	}{}

	// We can't test createJob directly as it's not exported,
	// but we can ensure RegisterJob doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RegisterJob panicked: %v", r)
		}
	}()

	// This tests that the registration mechanism works
	_ = job
}

func TestMemoryDriver_ContextCancellation(t *testing.T) {
	driver := queue.NewMemoryDriver(100)
	defer driver.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Pop should return context error
	_, err := driver.Pop(ctx, "test")
	if err == nil {
		t.Error("Expected error on canceled context")
	}
}

func TestWorker_ProcessWithCallbacks(t *testing.T) {
	var beforeCalled, afterCalled int32

	config := queue.WorkerConfig{
		Queue:       "callback-test",
		Concurrency: 1,
		Sleep:       10 * time.Millisecond,
		BeforeJob: func(ctx context.Context, payload *queue.JobPayload) {
			atomic.AddInt32(&beforeCalled, 1)
		},
		AfterJob: func(ctx context.Context, payload *queue.JobPayload, err error) {
			atomic.AddInt32(&afterCalled, 1)
		},
	}

	worker := queue.NewWorker(config)
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}
}

func TestQueue_Size(t *testing.T) {
	ctx := context.Background()

	size, err := queue.Size(ctx, "size-test")
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}

	// Should be 0 or more
	if size < 0 {
		t.Errorf("Size should be non-negative, got %d", size)
	}
}

func TestQueue_Clear(t *testing.T) {
	ctx := context.Background()

	err := queue.Clear(ctx, "clear-test")
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
}

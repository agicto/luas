package workflow

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/queue"
)

var counterJobCalls atomic.Int32

type counterJob struct{}

func (j *counterJob) Handle(ctx context.Context) error {
	counterJobCalls.Add(1)
	return nil
}

func TestManagerRunRetriesAndEventuallySucceeds(t *testing.T) {
	manager := NewManagerWith(queue.Global(), NewScheduler())

	attempts := 0
	err := manager.Run(context.Background(), "sync.retry", func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 0,
		MaxDelay:     time.Millisecond,
		Multiplier:   1,
		Jitter:       0,
	}))
	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}

func TestManagerRunStopsWhenShouldRetryRejectsError(t *testing.T) {
	manager := NewManagerWith(queue.Global(), NewScheduler())

	expectedErr := errors.New("do not retry")
	attempts := 0
	err := manager.Run(context.Background(), "sync.retry.reject", func(ctx context.Context) error {
		attempts++
		return expectedErr
	}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 0,
		MaxDelay:     time.Millisecond,
		Multiplier:   1,
		Jitter:       0,
		ShouldRetry:  func(err error) bool { return false },
	}))
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, 1, attempts)
}

func TestManagerRunReturnsMaxAttemptsError(t *testing.T) {
	manager := NewManagerWith(queue.Global(), NewScheduler())

	expectedErr := errors.New("still failing")
	attempts := 0
	err := manager.Run(context.Background(), "sync.retry.exhausted", func(ctx context.Context) error {
		attempts++
		return expectedErr
	}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:  2,
		InitialDelay: 0,
		MaxDelay:     time.Millisecond,
		Multiplier:   1,
		Jitter:       0,
	}))
	require.ErrorIs(t, err, expectedErr)
	require.ErrorIs(t, err, ErrMaxAttemptsExceeded)
	require.Equal(t, 2, attempts)
}

func TestManagerDispatchAfterExecutesRegisteredJob(t *testing.T) {
	manager := NewManagerWith(queue.Global(), NewScheduler())

	counterJobCalls.Store(0)
	job := &counterJob{}
	manager.Register(job)

	err := manager.DispatchAfter(context.Background(), 5*time.Millisecond, job)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return counterJobCalls.Load() == 1
	}, 200*time.Millisecond, 5*time.Millisecond)
}

func TestSchedulePlanRegistersAndRunsDueWork(t *testing.T) {
	scheduler := NewScheduler()
	manager := NewManagerWith(queue.Global(), scheduler)

	ran := 0
	manager.Schedule("nightly.sync", func(ctx context.Context) error {
		ran++
		return nil
	}).DailyAt(10, 30).Register()

	err := manager.RunDue(context.Background(), time.Date(2026, time.April, 27, 10, 30, 0, 0, time.Local))
	require.NoError(t, err)
	require.Equal(t, 1, ran)
}

func TestBootstrapConfiguresMemoryQueueDriver(t *testing.T) {
	t.Cleanup(func() {
		queue.Global().RegisterDriver("sync", queue.NewSyncDriver())
		require.NoError(t, queue.Global().SetDefaultDriver("sync"))
		queue.Global().SetDefaultQueue("default")
	})

	manager, err := Bootstrap(QueueRuntimeConfig{
		Driver:       "memory",
		DefaultQueue: "jobs",
		BufferSize:   32,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)
	require.Equal(t, "memory", queue.Global().DefaultDriverName())
}

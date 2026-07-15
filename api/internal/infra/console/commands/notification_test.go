package commands

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeNotificationDispatcher struct {
	limits []int
	counts []int
	errors []error
}

func (f *fakeNotificationDispatcher) DispatchDue(_ context.Context, limit int) (int, error) {
	f.limits = append(f.limits, limit)
	count := 0
	if len(f.counts) > 0 {
		count = f.counts[0]
		f.counts = f.counts[1:]
	}
	if len(f.errors) == 0 {
		return count, nil
	}
	err := f.errors[0]
	f.errors = f.errors[1:]
	return count, err
}

func TestParseNotificationWorkerArgs(t *testing.T) {
	cfg, err := parseNotificationWorkerArgs([]string{
		"--batch=40",
		"--poll",
		"500ms",
		"--max-attempts=75",
		"--once",
	})
	if err != nil {
		t.Fatalf("parseNotificationWorkerArgs() error = %v", err)
	}
	if cfg.Batch != 40 || cfg.Poll != 500*time.Millisecond || cfg.MaxAttempts != 75 || !cfg.Once {
		t.Fatalf("worker config = %+v", cfg)
	}
}

func TestParseNotificationWorkerArgsRejectsUnsafeBounds(t *testing.T) {
	tests := [][]string{
		{"--batch=0"},
		{"--batch=101"},
		{"--poll=10ms"},
		{"--poll=2m"},
		{"--max-attempts=-1"},
		{"--unknown"},
	}
	for _, args := range tests {
		if _, err := parseNotificationWorkerArgs(args); err == nil {
			t.Fatalf("parseNotificationWorkerArgs(%v) returned no error", args)
		}
	}
}

func TestRunNotificationWorkerHonorsMaximumWithoutOverClaiming(t *testing.T) {
	dispatcher := &fakeNotificationDispatcher{counts: []int{3, 2}}
	processed, err := runNotificationWorker(context.Background(), dispatcher, notificationWorkerConfig{
		Batch:       3,
		Poll:        100 * time.Millisecond,
		MaxAttempts: 5,
	})
	if err != nil {
		t.Fatalf("runNotificationWorker() error = %v", err)
	}
	if processed != 5 {
		t.Fatalf("processed = %d, want 5", processed)
	}
	if len(dispatcher.limits) != 2 || dispatcher.limits[0] != 3 || dispatcher.limits[1] != 2 {
		t.Fatalf("dispatch limits = %v, want [3 2]", dispatcher.limits)
	}
}

func TestRunNotificationWorkerOnceReturnsDispatchErrorAndCompletedCount(t *testing.T) {
	dispatchError := errors.New("database unavailable")
	dispatcher := &fakeNotificationDispatcher{
		counts: []int{2},
		errors: []error{dispatchError},
	}
	processed, err := runNotificationWorker(context.Background(), dispatcher, notificationWorkerConfig{
		Batch: 10,
		Poll:  time.Second,
		Once:  true,
	})
	if processed != 2 || !errors.Is(err, dispatchError) || len(dispatcher.limits) != 1 {
		t.Fatalf("run once error = processed %d, limits %v, error %v", processed, dispatcher.limits, err)
	}
}

func TestRunNotificationWorkerOnceRunsOneBatch(t *testing.T) {
	dispatcher := &fakeNotificationDispatcher{counts: []int{4}}
	processed, err := runNotificationWorker(context.Background(), dispatcher, notificationWorkerConfig{
		Batch: 10,
		Poll:  time.Second,
		Once:  true,
	})
	if err != nil || processed != 4 || len(dispatcher.limits) != 1 {
		t.Fatalf("run once = processed %d, limits %v, error %v", processed, dispatcher.limits, err)
	}
}

func TestRunNotificationWorkerReturnsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processed, err := runNotificationWorker(ctx, &fakeNotificationDispatcher{}, notificationWorkerConfig{
		Batch: 10,
		Poll:  time.Second,
	})
	if processed != 0 || err != context.Canceled {
		t.Fatalf("canceled worker = processed %d, error %v", processed, err)
	}
}

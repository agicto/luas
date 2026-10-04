package logger_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zgiai/luas/api/pkg/logger"
)

func TestSlogHandlerRoutesRecordsIntoTheLogger(t *testing.T) {
	cfg := logger.DefaultConfig()
	cfg.Level = logger.LevelDebug
	l := logger.New(cfg)
	memory := logger.NewMemoryHandler(10)
	l.AddHandler(memory)

	log := slog.New(logger.NewSlogHandler(l)).With("component", "worker").WithGroup("job")
	log.Warn("job.retry",
		"attempt", 3,
		"err", errors.New("upstream timeout"),
		"backoff", 1500*time.Millisecond,
		slog.Group("queue", "name", "default"),
	)

	entries := memory.Recent(1)
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Level != logger.LevelWarning || entry.Message != "job.retry" {
		t.Fatalf("unexpected entry level/message: %v %q", entry.Level, entry.Message)
	}
	want := map[string]any{
		"component":      "worker",
		"job.attempt":    int64(3),
		"job.err":        "upstream timeout",
		"job.backoff":    "1.5s",
		"job.queue.name": "default",
	}
	for key, value := range want {
		if entry.Context[key] != value {
			t.Errorf("context[%q] = %#v, want %#v", key, entry.Context[key], value)
		}
	}
}

func TestSlogHandlerCopiesRequestIdentifiers(t *testing.T) {
	cfg := logger.DefaultConfig()
	cfg.Level = logger.LevelDebug
	l := logger.New(cfg)
	memory := logger.NewMemoryHandler(10)
	l.AddHandler(memory)

	slog.New(logger.NewSlogHandler(l)).Error("request.failed", "request_id", "req-slog-2")

	entries := memory.RecentByRequest("req-slog-2", "", 1)
	if len(entries) != 1 || entries[0].Level != logger.LevelError {
		t.Fatalf("expected one correlated error entry, got %+v", entries)
	}
}

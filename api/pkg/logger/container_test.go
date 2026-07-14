package logger

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoggerSupportsStructuredStdoutWithoutFileHandler(t *testing.T) {
	cfg := ProductionConfig()
	cfg.Level = LevelInfo
	cfg.StdoutPrint = true
	cfg.FileEnabled = false
	cfg.JSON = true

	logger := New(cfg)
	var console *ConsoleHandler
	for _, handler := range logger.handlers {
		switch typed := handler.(type) {
		case *ConsoleHandler:
			console = typed
		case *FileHandler:
			t.Fatal("file handler registered while file logging is disabled")
		}
	}
	if console == nil {
		t.Fatal("console handler is missing")
	}

	var output bytes.Buffer
	console.writer = &output
	logger.Info("container.request", map[string]any{
		"request_id": "req_container",
		"status":     200,
	})

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("stdout is not JSON: %v; output = %q", err, output.String())
	}
	if record["message"] != "container.request" {
		t.Fatalf("message = %v, want container.request", record["message"])
	}
	if record["request_id"] != "req_container" {
		t.Fatalf("request_id = %v, want req_container", record["request_id"])
	}
}

func TestDefaultLoggerUsesExplicitOutputConfig(t *testing.T) {
	cfg := ProductionConfig()
	cfg.Level = LevelInfo
	cfg.StdoutPrint = true
	cfg.FileEnabled = false
	cfg.JSON = true

	runtimeLogger := BootWithConfig(cfg)
	t.Cleanup(func() {
		SetDefault(New(DefaultConfig()))
	})

	if Default() != runtimeLogger {
		t.Fatal("explicit logger was not installed as the process default")
	}
	if !runtimeLogger.config.StdoutPrint {
		t.Fatal("explicit config did not enable stdout")
	}
	if runtimeLogger.config.FileEnabled {
		t.Fatal("explicit config did not disable file logging")
	}
	if !runtimeLogger.config.JSON {
		t.Fatal("explicit config did not enable structured output")
	}
}

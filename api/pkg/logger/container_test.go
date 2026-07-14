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

func TestBootReadsContainerOutputEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_STDOUT", "true")
	t.Setenv("LOG_FILE_ENABLED", "false")
	t.Setenv("LOG_JSON", "true")

	logger := Boot()
	t.Cleanup(func() {
		SetDefault(New(DefaultConfig()))
	})

	if !logger.config.StdoutPrint {
		t.Fatal("LOG_STDOUT=true did not enable stdout")
	}
	if logger.config.FileEnabled {
		t.Fatal("LOG_FILE_ENABLED=false did not disable file logging")
	}
	if !logger.config.JSON {
		t.Fatal("LOG_JSON=true did not enable structured output")
	}
}

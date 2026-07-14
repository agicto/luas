package bootstrap

import (
	"path/filepath"
	"testing"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/logger"
)

func TestBuildLoggerConfigUsesTypedApplicationSnapshot(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Debug: false},
		Log: config.LogConfig{
			Level:       "warning",
			File:        filepath.Join("var", "log", "luas", "app.log"),
			Stdout:      true,
			FileEnabled: false,
			JSON:        true,
		},
	}

	got := buildLoggerConfig(cfg)

	if got.Level != logger.LevelWarning {
		t.Fatalf("level = %v, want warning", got.Level)
	}
	if got.Path != filepath.Join("var", "log", "luas") || got.File != "app.log" {
		t.Fatalf("file target = %q/%q, want var/log/luas/app.log", got.Path, got.File)
	}
	if !got.StdoutPrint || got.FileEnabled || !got.JSON {
		t.Fatalf("outputs = stdout:%v file:%v json:%v", got.StdoutPrint, got.FileEnabled, got.JSON)
	}
	if got.ColorEnabled {
		t.Fatal("JSON logging must disable terminal colors")
	}
}

func TestInitLoggerRequiresConfig(t *testing.T) {
	if err := InitLogger(nil); err == nil {
		t.Fatal("InitLogger(nil) error = nil, want validation error")
	}
}

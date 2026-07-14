package commands

import (
	"testing"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestParseServeOptions(t *testing.T) {
	options, err := parseServeOptions([]string{
		"--migrate",
		"--port=9025",
		"--env=development",
		"--env-file",
		".env.local",
	})
	if err != nil {
		t.Fatalf("parseServeOptions() error = %v", err)
	}
	if !options.migrate {
		t.Fatal("migrate = false, want true")
	}
	if options.port != 9025 {
		t.Fatalf("port = %d, want 9025", options.port)
	}
}

func TestParseServeOptionsValidatesPort(t *testing.T) {
	for _, args := range [][]string{
		{"--port"},
		{"--port=invalid"},
		{"--port=0"},
		{"--port=65536"},
	} {
		if _, err := parseServeOptions(args); err == nil {
			t.Fatalf("parseServeOptions(%q) error = nil, want validation error", args)
		}
	}
}

func TestValidateServeRuntimeRejectsProductionStartupMigrations(t *testing.T) {
	production := &config.Config{App: config.AppConfig{Env: "release"}}
	if err := validateServeRuntime(serveOptions{migrate: true}, production); err == nil {
		t.Fatal("production startup migration error = nil, want rejection")
	}

	development := &config.Config{App: config.AppConfig{Env: "development"}}
	if err := validateServeRuntime(serveOptions{migrate: true}, development); err != nil {
		t.Fatalf("development startup migration error = %v", err)
	}
}

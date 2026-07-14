package commands

import (
	"strings"
	"testing"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestMigrationProductionGuardUsesTypedConfigurationSnapshot(t *testing.T) {
	tests := []struct {
		environment string
		want        bool
	}{
		{environment: "production", want: true},
		{environment: "PROD", want: true},
		{environment: " release ", want: true},
		{environment: "development", want: false},
		{environment: "testing", want: false},
	}

	for _, test := range tests {
		t.Run(test.environment, func(t *testing.T) {
			cfg := &config.Config{App: config.AppConfig{Env: test.environment}}
			if got := cfg.IsProduction(); got != test.want {
				t.Fatalf("Config.IsProduction(%q) = %v, want %v", test.environment, got, test.want)
			}
		})
	}
}

func TestRequireProductionForce(t *testing.T) {
	production := &config.Config{App: config.AppConfig{Env: "release"}}
	development := &config.Config{App: config.AppConfig{Env: "development"}}

	if err := requireProductionForce(production, false, "rollback migrations"); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("production guard error = %v, want --force error", err)
	}
	if err := requireProductionForce(production, true, "rollback migrations"); err != nil {
		t.Fatalf("forced production operation error = %v", err)
	}
	if err := requireProductionForce(development, false, "rollback migrations"); err != nil {
		t.Fatalf("development operation error = %v", err)
	}
}

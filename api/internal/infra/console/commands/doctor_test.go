package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestReadEnvKeysRejectsDuplicateKeys(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\nAPP_NAME=Duplicate\n")

	_, err := readEnvKeys(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate key APP_NAME") {
		t.Fatalf("readEnvKeys() error = %v, want duplicate-key error", err)
	}
}

func TestReadEnvKeysRejectsMalformedEntries(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\nTHIS IS NOT AN ENV ENTRY\n")

	_, err := readEnvKeys(path)
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("readEnvKeys() error = %v, want malformed line error", err)
	}
}

func TestAuditDoctorDoesNotRequireOptionalExampleKeys(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\nOPTIONAL_ADAPTER_KEY=\n")
	report := auditDoctor(path, func() (*config.Config, error) {
		return healthyDoctorConfig(), nil
	})

	if report.failures() != 0 {
		t.Fatalf("auditDoctor() failures = %d, checks = %#v", report.failures(), report.checks)
	}
	for _, check := range report.checks {
		if strings.Contains(strings.ToLower(check.label+" "+check.detail), "unset") {
			t.Fatalf("optional example key produced an unset warning: %#v", check)
		}
	}
}

func TestAuditDoctorTreatsProviderModelIDsAsOpaque(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\n")
	cfg := healthyDoctorConfig()
	cfg.AI.Enabled = true
	cfg.AI.DefaultProvider = "openai"
	cfg.AI.DefaultModel = "gpt-5.4"
	cfg.AI.OpenAI.APIKey = "test-key"

	report := auditDoctor(path, func() (*config.Config, error) { return cfg, nil })

	for _, check := range report.checks {
		text := strings.ToLower(check.label + " " + check.detail)
		if strings.Contains(text, "not a real") || strings.Contains(text, "invalid model") {
			t.Fatalf("provider-owned model id was rejected: %#v", check)
		}
	}
	if !report.has(checkOK, "AI_DEFAULT_MODEL=gpt-5.4") {
		t.Fatalf("model check missing from report: %#v", report.checks)
	}
}

func TestAuditDoctorWarnsForProviderWithoutBuiltInAdapter(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\n")
	cfg := healthyDoctorConfig()
	cfg.AI.Enabled = true
	cfg.AI.DefaultProvider = "downstream-provider"
	cfg.AI.DefaultModel = "downstream-model"

	report := auditDoctor(path, func() (*config.Config, error) { return cfg, nil })

	if !report.has(checkWarning, "requires a downstream adapter") {
		t.Fatalf("adapter warning missing from report: %#v", report.checks)
	}
}

func TestAuditDoctorReportsConfigLoadFailure(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\n")
	report := auditDoctor(path, func() (*config.Config, error) {
		return nil, errors.New("invalid configuration")
	})

	if !report.has(checkFailure, "config.Load()") {
		t.Fatalf("config failure missing from report: %#v", report.checks)
	}
}

func TestAuditDoctorPreservesProductionAliasInReport(t *testing.T) {
	path := writeDoctorEnvExample(t, "APP_NAME=Luas\n")
	cfg := healthyDoctorConfig()
	cfg.App.Env = "release"

	report := auditDoctor(path, func() (*config.Config, error) { return cfg, nil })

	if !report.has(checkOK, "APP_ENV=release (production defaults active)") {
		t.Fatalf("production alias check missing from report: %#v", report.checks)
	}
}

func healthyDoctorConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{Env: "development"},
		Authentication: config.AuthenticationConfig{
			SessionTTL:           config.DefaultAuthenticationSessionTTL,
			SessionIdleTimeout:   config.DefaultAuthenticationSessionIdleTimeout,
			SessionTouchInterval: config.DefaultAuthenticationSessionTouchInterval,
			SessionRetention:     config.DefaultAuthenticationSessionRetention,
		},
		CORS: config.CORSConfig{
			AllowOrigins:     []string{"http://localhost:3000"},
			AllowCredentials: true,
		},
	}
}

func writeDoctorEnvExample(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env.example")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env example: %v", err)
	}
	return path
}

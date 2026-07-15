package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAIConfig(t *testing.T) {
	valid := func() AIConfig {
		return AIConfig{
			Enabled:             true,
			DefaultProvider:     "openai",
			DefaultModel:        "provider-model",
			RequestTimeout:      DefaultAIRequestTimeout,
			MaxInputBytes:       DefaultAIMaxInputBytes,
			MaxResponseBytes:    DefaultAIMaxResponseBytes,
			MaxStreamEventBytes: DefaultAIMaxStreamEventBytes,
			OpenAI: AIProviderConfig{
				APIKey:  "test-key",
				BaseURL: "https://provider.example/v1",
			},
		}
	}

	tests := []struct {
		name       string
		production bool
		mutate     func(*AIConfig)
		wantError  string
	}{
		{name: "valid production", production: true},
		{name: "valid development HTTP", mutate: func(cfg *AIConfig) { cfg.OpenAI.BaseURL = "http://127.0.0.1:8080/v1" }},
		{name: "timeout required", mutate: func(cfg *AIConfig) { cfg.RequestTimeout = 0 }, wantError: "AI_REQUEST_TIMEOUT"},
		{name: "timeout capped", mutate: func(cfg *AIConfig) { cfg.RequestTimeout = 15*time.Minute + time.Second }, wantError: "AI_REQUEST_TIMEOUT"},
		{name: "input minimum", mutate: func(cfg *AIConfig) { cfg.MaxInputBytes = 1023 }, wantError: "AI_MAX_INPUT_BYTES"},
		{name: "response maximum", mutate: func(cfg *AIConfig) { cfg.MaxResponseBytes = 32*1024*1024 + 1 }, wantError: "AI_MAX_RESPONSE_BYTES"},
		{name: "event no larger than response", mutate: func(cfg *AIConfig) { cfg.MaxResponseBytes = 64 * 1024 }, wantError: "must not exceed"},
		{name: "provider required", mutate: func(cfg *AIConfig) { cfg.DefaultProvider = "" }, wantError: "AI_DEFAULT_PROVIDER"},
		{name: "provider registered", mutate: func(cfg *AIConfig) { cfg.DefaultProvider = "downstream" }, wantError: "not registered"},
		{name: "model explicit", mutate: func(cfg *AIConfig) { cfg.DefaultModel = "" }, wantError: "AI_DEFAULT_MODEL"},
		{name: "key required", mutate: func(cfg *AIConfig) { cfg.OpenAI.APIKey = "" }, wantError: "OPENAI_API_KEY"},
		{name: "endpoint credentials rejected", mutate: func(cfg *AIConfig) { cfg.OpenAI.BaseURL = "https://user:pass@provider.example/v1" }, wantError: "without credentials"},
		{name: "endpoint query rejected", mutate: func(cfg *AIConfig) { cfg.OpenAI.BaseURL = "https://provider.example/v1?secret=value" }, wantError: "query"},
		{name: "production HTTPS", production: true, mutate: func(cfg *AIConfig) { cfg.OpenAI.BaseURL = "http://provider.example/v1" }, wantError: "https in production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			err := validateAIConfig(cfg, tt.production)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateAIConfig() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateAIConfig() error = %v, want %q", err, tt.wantError)
			}
		})
	}

	if err := validateAIConfig(AIConfig{}, true); err != nil {
		t.Fatalf("disabled zero-value AI config error = %v", err)
	}
}

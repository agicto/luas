package config

import (
	"strings"
	"testing"
)

// baseValidConfig returns a config that passes validate() — tests then
// mutate the field under test.
func baseValidConfig(env string) *Config {
	return &Config{
		App: AppConfig{Env: env},
		Database: DatabaseConfig{
			Enabled:  false,
			Driver:   "sqlite",
			Password: "",
		},
		JWT: JWTConfig{
			Secret: strings.Repeat("a", 64),
		},
		CORS: CORSConfig{
			AllowOrigins:     []string{"https://app.example.com"},
			AllowCredentials: true,
		},
	}
}

func TestValidate_AcceptsValidConfig(t *testing.T) {
	if err := validate(baseValidConfig("production")); err != nil {
		t.Fatalf("validate() error = %v, want nil", err)
	}
	if err := validate(baseValidConfig("development")); err != nil {
		t.Fatalf("validate() error = %v, want nil", err)
	}
}

func TestValidate_RejectsEmptyJWTSecret(t *testing.T) {
	cfg := baseValidConfig("development")
	cfg.JWT.Secret = ""
	if err := validate(cfg); err == nil {
		t.Fatal("expected error for empty JWT_SECRET")
	}
}

func TestValidate_RejectsPlaceholderJWTSecretInProduction(t *testing.T) {
	for _, secret := range []string{
		"replace_me_with_a_long_random_secret_at_least_32_chars",
		"your_jwt_secret_key_here",
		"replace-me",
	} {
		cfg := baseValidConfig("production")
		cfg.JWT.Secret = secret
		err := validate(cfg)
		if err == nil || !strings.Contains(err.Error(), "placeholder") {
			t.Fatalf("expected placeholder error for %q, got %v", secret, err)
		}
	}
}

func TestValidate_AllowsPlaceholderInDevelopment(t *testing.T) {
	cfg := baseValidConfig("development")
	cfg.JWT.Secret = "your_jwt_secret_key_here"
	if err := validate(cfg); err != nil {
		t.Fatalf("dev mode should tolerate placeholder, got %v", err)
	}
}

func TestValidate_RejectsShortJWTSecretInProduction(t *testing.T) {
	cfg := baseValidConfig("production")
	cfg.JWT.Secret = strings.Repeat("a", 16)
	err := validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "32 characters") {
		t.Fatalf("expected length error, got %v", err)
	}
}

func TestValidate_RejectsWildcardWithCredentials(t *testing.T) {
	cfg := baseValidConfig("development")
	cfg.CORS.AllowOrigins = []string{"*"}
	cfg.CORS.AllowCredentials = true
	err := validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "'*'") {
		t.Fatalf("expected wildcard+credentials error, got %v", err)
	}
}

func TestValidate_AllowsWildcardWithoutCredentials(t *testing.T) {
	cfg := baseValidConfig("development")
	cfg.CORS.AllowOrigins = []string{"*"}
	cfg.CORS.AllowCredentials = false
	if err := validate(cfg); err != nil {
		t.Fatalf("wildcard with credentials=false should be OK, got %v", err)
	}
}

func TestValidate_RejectsLocalhostOriginInProduction(t *testing.T) {
	cfg := baseValidConfig("production")
	cfg.CORS.AllowOrigins = []string{"http://localhost:3000"}
	err := validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "localhost") {
		t.Fatalf("expected localhost error, got %v", err)
	}
}

package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func withoutEnv(t *testing.T, keys ...string) {
	t.Helper()

	previous := make(map[string]string, len(keys))
	existed := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		previous[key] = value
		existed[key] = ok
		os.Unsetenv(key)
	}

	t.Cleanup(func() {
		for _, key := range keys {
			if existed[key] {
				os.Setenv(key, previous[key])
			} else {
				os.Unsetenv(key)
			}
		}
		LoadFresh()
	})
}

func loadConfigForRateLimitDefault(t *testing.T, appEnv string) *Config {
	t.Helper()

	withoutEnv(
		t,
		"MIDDLEWARE_RATE_LIMIT_ENABLED",
		"MIDDLEWARE_RATE_LIMIT_MAX",
		"MIDDLEWARE_RATE_LIMIT_WINDOW",
		"MIDDLEWARE_RATE_LIMIT_SKIP_PATHS",
	)
	t.Setenv("APP_ENV", appEnv)
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	return cfg
}

func TestLoad_RateLimitDefaultsByEnvironment(t *testing.T) {
	prod := loadConfigForRateLimitDefault(t, "production")
	if !prod.Middleware.RateLimit.Enabled {
		t.Fatal("production should enable rate limit by default")
	}
	if prod.Middleware.RateLimit.Max != 600 {
		t.Fatalf("production rate limit max = %d, want 600", prod.Middleware.RateLimit.Max)
	}
	if prod.Middleware.RateLimit.Window != time.Minute {
		t.Fatalf("production rate limit window = %s, want 1m", prod.Middleware.RateLimit.Window)
	}

	dev := loadConfigForRateLimitDefault(t, "development")
	if dev.Middleware.RateLimit.Enabled {
		t.Fatal("development should disable rate limit by default")
	}
}

func TestLoad_RateLimitExplicitEnvOverridesDefault(t *testing.T) {
	withoutEnv(
		t,
		"MIDDLEWARE_RATE_LIMIT_ENABLED",
		"MIDDLEWARE_RATE_LIMIT_MAX",
		"MIDDLEWARE_RATE_LIMIT_WINDOW",
		"MIDDLEWARE_RATE_LIMIT_SKIP_PATHS",
	)
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")
	t.Setenv("MIDDLEWARE_RATE_LIMIT_ENABLED", "true")
	t.Setenv("MIDDLEWARE_RATE_LIMIT_MAX", "42")
	t.Setenv("MIDDLEWARE_RATE_LIMIT_WINDOW", "30s")
	t.Setenv("MIDDLEWARE_RATE_LIMIT_SKIP_PATHS", "/health,/metrics")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}

	if !cfg.Middleware.RateLimit.Enabled {
		t.Fatal("explicit env should enable rate limit")
	}
	if cfg.Middleware.RateLimit.Max != 42 {
		t.Fatalf("rate limit max = %d, want 42", cfg.Middleware.RateLimit.Max)
	}
	if cfg.Middleware.RateLimit.Window != 30*time.Second {
		t.Fatalf("rate limit window = %s, want 30s", cfg.Middleware.RateLimit.Window)
	}
	if got := strings.Join(cfg.Middleware.RateLimit.SkipPaths, ","); got != "/health,/metrics" {
		t.Fatalf("rate limit skip paths = %q, want /health,/metrics", got)
	}
}

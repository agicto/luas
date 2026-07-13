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

func authRateLimitEnvKeys() []string {
	return []string{
		"AUTH_RATE_LIMIT_ENABLED",
		"AUTH_RATE_LIMIT_LOGIN_IP_MAX",
		"AUTH_RATE_LIMIT_LOGIN_IP_WINDOW",
		"AUTH_RATE_LIMIT_LOGIN_SUBJECT_MAX",
		"AUTH_RATE_LIMIT_LOGIN_SUBJECT_WINDOW",
		"AUTH_RATE_LIMIT_REGISTER_IP_MAX",
		"AUTH_RATE_LIMIT_REGISTER_IP_WINDOW",
		"AUTH_RATE_LIMIT_REGISTER_SUBJECT_MAX",
		"AUTH_RATE_LIMIT_REGISTER_SUBJECT_WINDOW",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_IP_MAX",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_IP_WINDOW",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_MAX",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_WINDOW",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_IP_MAX",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_IP_WINDOW",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_SUBJECT_MAX",
		"AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_SUBJECT_WINDOW",
	}
}

func loadConfigForAuthRateLimitDefault(t *testing.T, appEnv string) *Config {
	t.Helper()

	withoutEnv(t, authRateLimitEnvKeys()...)
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

func TestLoad_AuthenticationRateLimitDefaultsByEnvironment(t *testing.T) {
	prod := loadConfigForAuthRateLimitDefault(t, "production")
	got := prod.Middleware.AuthenticationRateLimit
	if !got.Enabled {
		t.Fatal("production should enable authentication rate limits by default")
	}
	if got.Login.PerIP.Max != 20 || got.Login.PerIP.Window != 5*time.Minute {
		t.Fatalf("login per-IP default = %#v, want 20/5m", got.Login.PerIP)
	}
	if got.Login.PerSubject.Max != 10 || got.Login.PerSubject.Window != 15*time.Minute {
		t.Fatalf("login per-subject default = %#v, want 10/15m", got.Login.PerSubject)
	}
	if got.PasswordReset.PerSubject.Max != 3 || got.PasswordReset.PerSubject.Window != time.Hour {
		t.Fatalf("password reset per-subject default = %#v, want 3/1h", got.PasswordReset.PerSubject)
	}

	dev := loadConfigForAuthRateLimitDefault(t, "development")
	if dev.Middleware.AuthenticationRateLimit.Enabled {
		t.Fatal("development should disable authentication rate limits by default")
	}
}

func TestLoad_AuthenticationRateLimitExplicitEnvOverridesDefault(t *testing.T) {
	withoutEnv(t, authRateLimitEnvKeys()...)
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")
	t.Setenv("AUTH_RATE_LIMIT_ENABLED", "true")
	t.Setenv("AUTH_RATE_LIMIT_LOGIN_IP_MAX", "7")
	t.Setenv("AUTH_RATE_LIMIT_LOGIN_IP_WINDOW", "45s")
	t.Setenv("AUTH_RATE_LIMIT_LOGIN_SUBJECT_MAX", "4")
	t.Setenv("AUTH_RATE_LIMIT_LOGIN_SUBJECT_WINDOW", "2m")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}

	got := cfg.Middleware.AuthenticationRateLimit
	if !got.Enabled {
		t.Fatal("explicit env should enable authentication rate limits")
	}
	if got.Login.PerIP.Max != 7 || got.Login.PerIP.Window != 45*time.Second {
		t.Fatalf("login per-IP override = %#v, want 7/45s", got.Login.PerIP)
	}
	if got.Login.PerSubject.Max != 4 || got.Login.PerSubject.Window != 2*time.Minute {
		t.Fatalf("login per-subject override = %#v, want 4/2m", got.Login.PerSubject)
	}
}

func TestLoad_TrustedProxyDefaultsAndOverride(t *testing.T) {
	withoutEnv(t, "SERVER_TRUSTED_PROXIES")
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if len(cfg.Server.TrustedProxies) != 0 {
		t.Fatalf("trusted proxy default = %v, want none", cfg.Server.TrustedProxies)
	}

	t.Setenv("SERVER_TRUSTED_PROXIES", "10.0.0.0/8,192.0.2.10")
	cfg, err = LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() with trusted proxies error = %v", err)
	}
	if got := strings.Join(cfg.Server.TrustedProxies, ","); got != "10.0.0.0/8,192.0.2.10" {
		t.Fatalf("trusted proxies = %q, want configured CIDRs", got)
	}
}

func loadConfigForMetricsDefault(t *testing.T, appEnv string) *Config {
	t.Helper()

	withoutEnv(t, "METRICS_ENABLED")
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

func TestLoad_MetricsDefaultsByEnvironment(t *testing.T) {
	prod := loadConfigForMetricsDefault(t, "production")
	if prod.Metrics.Enabled {
		t.Fatal("production should disable metrics by default")
	}

	dev := loadConfigForMetricsDefault(t, "development")
	if !dev.Metrics.Enabled {
		t.Fatal("development should enable metrics by default")
	}
}

func TestLoad_MetricsExplicitEnvOverridesDefault(t *testing.T) {
	withoutEnv(t, "METRICS_ENABLED")
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")
	t.Setenv("METRICS_ENABLED", "true")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if !cfg.Metrics.Enabled {
		t.Fatal("explicit env should enable metrics")
	}
}

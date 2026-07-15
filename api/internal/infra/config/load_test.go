package config

import (
	"os"
	"strings"
	"testing"
	"time"

	pkgenv "github.com/zgiai/luas/api/pkg/env"
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

func TestLoad_AuthenticationSessionDefaultsAndLegacyJWTRejection(t *testing.T) {
	withoutEnv(
		t,
		"AUTH_SESSION_TTL",
		"AUTH_SESSION_IDLE_TIMEOUT",
		"AUTH_SESSION_TOUCH_INTERVAL",
		"AUTH_SESSION_RETENTION",
		"JWT_SECRET",
		"JWT_EXPIRE_DAYS",
	)
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if cfg.Authentication.SessionTTL != DefaultAuthenticationSessionTTL ||
		cfg.Authentication.SessionIdleTimeout != DefaultAuthenticationSessionIdleTimeout ||
		cfg.Authentication.SessionTouchInterval != DefaultAuthenticationSessionTouchInterval ||
		cfg.Authentication.SessionRetention != DefaultAuthenticationSessionRetention {
		t.Fatalf("authentication defaults = %#v", cfg.Authentication)
	}

	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	if _, err := LoadFresh(); err == nil || !strings.Contains(err.Error(), "no longer supported") {
		t.Fatalf("LoadFresh() legacy JWT error = %v", err)
	}
}

func TestLoad_OptionalStartersAreAdditiveAndEmptyByDefault(t *testing.T) {
	withoutEnv(t, "OPTIONAL_STARTERS")
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if len(cfg.Starters.Optional) != 0 {
		t.Fatalf("default optional starters = %v, want none", cfg.Starters.Optional)
	}

	t.Setenv("OPTIONAL_STARTERS", "organization,notification")
	cfg, err = LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() with optional starters error = %v", err)
	}
	if got := strings.Join(cfg.Starters.Optional, ","); got != "organization,notification" {
		t.Fatalf("optional starters = %q, want organization,notification", got)
	}
}

func TestLoad_EmailRequestTimeout(t *testing.T) {
	withoutEnv(t, "MAIL_FROM", "RESEND_API_KEY", "MAIL_REQUEST_TIMEOUT")
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if cfg.Email.RequestTimeout != DefaultEmailRequestTimeout {
		t.Fatalf("default email timeout = %s, want %s", cfg.Email.RequestTimeout, DefaultEmailRequestTimeout)
	}

	t.Setenv("MAIL_FROM", "Luas <noreply@example.com>")
	t.Setenv("RESEND_API_KEY", "resend-key")
	t.Setenv("MAIL_REQUEST_TIMEOUT", "3s")
	cfg, err = LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() with email config error = %v", err)
	}
	if cfg.Email.RequestTimeout != 3*time.Second {
		t.Fatalf("email timeout = %s, want 3s", cfg.Email.RequestTimeout)
	}
}

func TestLoad_OrganizationInvitationTTL(t *testing.T) {
	withoutEnv(t, "ORGANIZATION_INVITATION_TTL")
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if cfg.Organization.InvitationTTL != DefaultOrganizationInvitationTTL {
		t.Fatalf("default invitation TTL = %s, want %s", cfg.Organization.InvitationTTL, DefaultOrganizationInvitationTTL)
	}

	t.Setenv("ORGANIZATION_INVITATION_TTL", "72h")
	cfg, err = LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() with invitation TTL error = %v", err)
	}
	if cfg.Organization.InvitationTTL != 72*time.Hour {
		t.Fatalf("invitation TTL = %s, want 72h", cfg.Organization.InvitationTTL)
	}
}

func TestLoad_WebhookDefaultsAndOverrides(t *testing.T) {
	withoutEnv(
		t,
		"WEBHOOK_ENCRYPTION_KEY",
		"WEBHOOK_REQUEST_TIMEOUT",
		"WEBHOOK_MAX_RESPONSE_BYTES",
		"WEBHOOK_SECRET_OVERLAP",
		"WEBHOOK_EVENT_RETENTION",
		"WEBHOOK_ALLOW_INSECURE_HTTP",
		"WEBHOOK_ALLOW_PRIVATE_TARGETS",
	)
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if cfg.Webhook.RequestTimeout != DefaultWebhookRequestTimeout ||
		cfg.Webhook.MaxResponseBytes != DefaultWebhookMaxResponseBytes ||
		cfg.Webhook.SecretOverlap != DefaultWebhookSecretOverlap ||
		cfg.Webhook.EventRetention != DefaultWebhookEventRetention {
		t.Fatalf("unexpected webhook defaults: %+v", cfg.Webhook)
	}

	t.Setenv("WEBHOOK_REQUEST_TIMEOUT", "5s")
	t.Setenv("WEBHOOK_MAX_RESPONSE_BYTES", "8192")
	t.Setenv("WEBHOOK_SECRET_OVERLAP", "2h")
	t.Setenv("WEBHOOK_EVENT_RETENTION", "168h")
	t.Setenv("WEBHOOK_ALLOW_INSECURE_HTTP", "true")
	t.Setenv("WEBHOOK_ALLOW_PRIVATE_TARGETS", "true")
	cfg, err = LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() with webhook overrides error = %v", err)
	}
	if cfg.Webhook.RequestTimeout != 5*time.Second || cfg.Webhook.MaxResponseBytes != 8192 ||
		cfg.Webhook.SecretOverlap != 2*time.Hour || cfg.Webhook.EventRetention != 7*24*time.Hour ||
		!cfg.Webhook.AllowInsecureHTTP || !cfg.Webhook.AllowPrivateTargets {
		t.Fatalf("unexpected webhook overrides: %+v", cfg.Webhook)
	}
}

func TestLoad_EnvironmentAliasUsesProductionDefaults(t *testing.T) {
	withoutEnv(
		t,
		"APP_ENV",
		"APP_DEBUG",
		"GO_ENV",
		"GIN_MODE",
		"SERVER_MODE",
		"LOG_LEVEL",
		"LOG_STDOUT",
		"LOG_FILE_ENABLED",
		"LOG_JSON",
		"METRICS_ENABLED",
		"MIDDLEWARE_RATE_LIMIT_ENABLED",
		"AUTH_RATE_LIMIT_ENABLED",
	)
	t.Setenv("GO_ENV", " release ")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if cfg.App.Env != "release" || !cfg.IsProduction() {
		t.Fatalf("environment = %q, production = %v", cfg.App.Env, cfg.IsProduction())
	}
	if cfg.App.Debug || cfg.Server.Mode != "release" {
		t.Fatalf("production runtime = debug:%v server-mode:%q", cfg.App.Debug, cfg.Server.Mode)
	}
	if !cfg.Middleware.RateLimit.Enabled || !cfg.Middleware.AuthenticationRateLimit.Enabled {
		t.Fatal("production alias did not enable default abuse controls")
	}
	if cfg.Metrics.Enabled {
		t.Fatal("production alias did not disable metrics by default")
	}
	if cfg.Log.Level != "info" {
		t.Fatalf("production alias log level = %q, want info", cfg.Log.Level)
	}
	if !cfg.Log.Stdout || cfg.Log.FileEnabled || !cfg.Log.JSON {
		t.Fatalf(
			"production alias log outputs = stdout:%v file:%v json:%v",
			cfg.Log.Stdout,
			cfg.Log.FileEnabled,
			cfg.Log.JSON,
		)
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

func serverTransportEnvKeys() []string {
	return []string{
		"SERVER_HOST",
		"SERVER_PORT",
		"SERVER_READ_TIMEOUT",
		"SERVER_READ_HEADER_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
		"SERVER_MAX_HEADER_BYTES",
		"MIDDLEWARE_REQUEST_TIMEOUT",
	}
}

func TestLoad_ServerTransportDefaultsAreSafeAndCoherent(t *testing.T) {
	withoutEnv(t, serverTransportEnvKeys()...)
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("server host = %q, want loopback", cfg.Server.Host)
	}
	if cfg.Server.ReadTimeout != 60 {
		t.Fatalf("read timeout = %d, want 60", cfg.Server.ReadTimeout)
	}
	if cfg.Server.ReadHeaderTimeout != 10 {
		t.Fatalf("read header timeout = %d, want 10", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Server.WriteTimeout != 190 {
		t.Fatalf("write timeout = %d, want 190", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 120 {
		t.Fatalf("idle timeout = %d, want 120", cfg.Server.IdleTimeout)
	}
	if cfg.Server.MaxHeaderBytes != 64*1024 {
		t.Fatalf("max header bytes = %d, want %d", cfg.Server.MaxHeaderBytes, 64*1024)
	}
	if cfg.Server.WriteTimeout <= cfg.Middleware.RequestTimeout {
		t.Fatalf(
			"write timeout = %d must exceed request timeout = %d",
			cfg.Server.WriteTimeout,
			cfg.Middleware.RequestTimeout,
		)
	}
}

func TestLoad_ServerTransportExplicitEnvOverridesDefaults(t *testing.T) {
	withoutEnv(t, serverTransportEnvKeys()...)
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")
	t.Setenv("SERVER_HOST", "::1")
	t.Setenv("SERVER_PORT", "9025")
	t.Setenv("SERVER_READ_TIMEOUT", "31")
	t.Setenv("SERVER_READ_HEADER_TIMEOUT", "7")
	t.Setenv("SERVER_WRITE_TIMEOUT", "91")
	t.Setenv("SERVER_IDLE_TIMEOUT", "47")
	t.Setenv("SERVER_MAX_HEADER_BYTES", "32768")
	t.Setenv("MIDDLEWARE_REQUEST_TIMEOUT", "90")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}

	if cfg.Server.Host != "::1" || cfg.Server.Port != 9025 {
		t.Fatalf("server endpoint = %s:%d, want [::1]:9025", cfg.Server.Host, cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 31 || cfg.Server.ReadHeaderTimeout != 7 {
		t.Fatalf("read budgets = %d/%d, want 31/7", cfg.Server.ReadTimeout, cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Server.WriteTimeout != 91 || cfg.Server.IdleTimeout != 47 {
		t.Fatalf("write/idle budgets = %d/%d, want 91/47", cfg.Server.WriteTimeout, cfg.Server.IdleTimeout)
	}
	if cfg.Server.MaxHeaderBytes != 32768 {
		t.Fatalf("max header bytes = %d, want 32768", cfg.Server.MaxHeaderBytes)
	}
}

func TestLoad_LogOutputEnvironment(t *testing.T) {
	withoutEnv(t, "LOG_LEVEL", "LOG_FILE", "LOG_STDOUT", "LOG_FILE_ENABLED", "LOG_JSON", "SENTRY_DSN")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("DB_ENABLED", "false")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example.com")
	t.Setenv("LOG_STDOUT", "true")
	t.Setenv("LOG_FILE_ENABLED", "false")
	t.Setenv("LOG_JSON", "true")
	t.Setenv("LOG_LEVEL", "warning")
	t.Setenv("LOG_FILE", "var/log/luas/app.log")
	t.Setenv("SENTRY_DSN", "https://public@example.invalid/1")

	cfg, err := LoadFresh()
	if err != nil {
		t.Fatalf("LoadFresh() error = %v", err)
	}
	if !cfg.Log.Stdout || cfg.Log.FileEnabled || !cfg.Log.JSON {
		t.Fatalf("log outputs = stdout:%v file:%v json:%v, want true/false/true", cfg.Log.Stdout, cfg.Log.FileEnabled, cfg.Log.JSON)
	}
	if cfg.Log.Level != "warning" || cfg.Log.File != "var/log/luas/app.log" {
		t.Fatalf("log target = level:%q file:%q", cfg.Log.Level, cfg.Log.File)
	}
	if cfg.Sentry.DSN != "https://public@example.invalid/1" {
		t.Fatalf("Sentry DSN = %q", cfg.Sentry.DSN)
	}
}

func TestLoadAIConfigDoesNotRequireServerRuntimeSecrets(t *testing.T) {
	withoutEnv(t, "AI_ENABLED", "AI_DEFAULT_PROVIDER", "AI_DEFAULT_MODEL", "AI_REQUEST_TIMEOUT", "AI_MAX_INPUT_BYTES", "AI_MAX_RESPONSE_BYTES", "AI_MAX_STREAM_EVENT_BYTES", "OPENAI_API_KEY", "OPENAI_BASE_URL", "DB_PASSWORD")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "provider-model")
	t.Setenv("AI_REQUEST_TIMEOUT", "45s")
	t.Setenv("AI_MAX_INPUT_BYTES", "262144")
	t.Setenv("AI_MAX_RESPONSE_BYTES", "2097152")
	t.Setenv("AI_MAX_STREAM_EVENT_BYTES", "131072")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://provider.example/v1")
	if err := pkgenv.LoadFresh(); err != nil {
		t.Fatalf("env.LoadFresh() error = %v", err)
	}

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig() error = %v", err)
	}
	if cfg.DefaultModel != "provider-model" || cfg.RequestTimeout != 45*time.Second {
		t.Fatalf("AI config = model:%q timeout:%s", cfg.DefaultModel, cfg.RequestTimeout)
	}
	if cfg.MaxInputBytes != 262144 || cfg.MaxResponseBytes != 2097152 || cfg.MaxStreamEventBytes != 131072 {
		t.Fatalf("AI limits = input:%d response:%d event:%d", cfg.MaxInputBytes, cfg.MaxResponseBytes, cfg.MaxStreamEventBytes)
	}
	if cfg.OpenAI.APIKey != "test-key" || cfg.OpenAI.BaseURL != "https://provider.example/v1" {
		t.Fatalf("OpenAI config = key:%q base:%q", cfg.OpenAI.APIKey, cfg.OpenAI.BaseURL)
	}
}

func TestLoadAIConfigDefaultsToDisabledWithoutModel(t *testing.T) {
	withoutEnv(t, "AI_ENABLED", "AI_DEFAULT_PROVIDER", "AI_DEFAULT_MODEL", "AI_REQUEST_TIMEOUT", "AI_MAX_INPUT_BYTES", "AI_MAX_RESPONSE_BYTES", "AI_MAX_STREAM_EVENT_BYTES", "OPENAI_API_KEY", "OPENAI_BASE_URL")
	if err := pkgenv.LoadFresh(); err != nil {
		t.Fatalf("env.LoadFresh() error = %v", err)
	}

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig() error = %v", err)
	}
	if cfg.Enabled || cfg.DefaultModel != "" {
		t.Fatalf("AI defaults = enabled:%v model:%q, want disabled with no model", cfg.Enabled, cfg.DefaultModel)
	}
}

func TestLoadAIConfigRejectsIncompleteEnabledProvider(t *testing.T) {
	withoutEnv(t, "APP_ENV", "AI_ENABLED", "AI_DEFAULT_PROVIDER", "AI_DEFAULT_MODEL", "AI_REQUEST_TIMEOUT", "AI_MAX_INPUT_BYTES", "AI_MAX_RESPONSE_BYTES", "AI_MAX_STREAM_EVENT_BYTES", "OPENAI_API_KEY", "OPENAI_BASE_URL")
	t.Setenv("APP_ENV", "development")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "provider-model")
	t.Setenv("OPENAI_API_KEY", "")
	if err := pkgenv.LoadFresh(); err != nil {
		t.Fatalf("env.LoadFresh() error = %v", err)
	}

	_, err := LoadAIConfig()
	if err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("LoadAIConfig() error = %v, want missing provider key error", err)
	}
}

func TestLoadAIConfigRejectsInsecureProductionEndpoint(t *testing.T) {
	withoutEnv(t, "APP_ENV", "AI_ENABLED", "AI_DEFAULT_PROVIDER", "AI_DEFAULT_MODEL", "AI_REQUEST_TIMEOUT", "AI_MAX_INPUT_BYTES", "AI_MAX_RESPONSE_BYTES", "AI_MAX_STREAM_EVENT_BYTES", "OPENAI_API_KEY", "OPENAI_BASE_URL")
	t.Setenv("APP_ENV", "production")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_PROVIDER", "openai")
	t.Setenv("AI_DEFAULT_MODEL", "provider-model")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "http://provider.example/v1")
	if err := pkgenv.LoadFresh(); err != nil {
		t.Fatalf("env.LoadFresh() error = %v", err)
	}

	_, err := LoadAIConfig()
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("LoadAIConfig() error = %v, want production HTTPS error", err)
	}
}

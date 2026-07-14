package config

import (
	"strings"
	"testing"
	"time"
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

func TestValidate_RejectsInvalidRateLimitWhenEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  RateLimitConfig
		want string
	}{
		{
			name: "zero max",
			cfg: RateLimitConfig{
				Enabled: true,
				Max:     0,
				Window:  time.Minute,
			},
			want: "MIDDLEWARE_RATE_LIMIT_MAX",
		},
		{
			name: "zero window",
			cfg: RateLimitConfig{
				Enabled: true,
				Max:     100,
				Window:  0,
			},
			want: "MIDDLEWARE_RATE_LIMIT_WINDOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseValidConfig("production")
			cfg.Middleware.RateLimit = tt.cfg
			err := validate(cfg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %s error, got %v", tt.want, err)
			}
		})
	}
}

func TestValidate_AllowsDisabledRateLimitWithoutValues(t *testing.T) {
	cfg := baseValidConfig("production")
	cfg.Middleware.RateLimit = RateLimitConfig{Enabled: false}
	if err := validate(cfg); err != nil {
		t.Fatalf("disabled rate limit should not require max/window, got %v", err)
	}
}

func TestValidate_RejectsUnsafeOrInvalidTrustedProxies(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid", value: "not-an-ip"},
		{name: "all IPv4", value: "0.0.0.0/0"},
		{name: "all IPv6", value: "::/0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseValidConfig("production")
			cfg.Server.TrustedProxies = []string{tt.value}
			err := validate(cfg)
			if err == nil || !strings.Contains(err.Error(), "SERVER_TRUSTED_PROXIES") {
				t.Fatalf("expected trusted proxy validation error for %q, got %v", tt.value, err)
			}
		})
	}
}

func TestValidate_AcceptsSpecificTrustedProxyRanges(t *testing.T) {
	cfg := baseValidConfig("production")
	cfg.Server.TrustedProxies = []string{"10.0.0.0/8", "192.0.2.10", "2001:db8::/32"}
	if err := validate(cfg); err != nil {
		t.Fatalf("specific trusted proxy ranges should be valid, got %v", err)
	}
}

func TestValidate_RejectsInvalidAuthenticationRateLimitWhenEnabled(t *testing.T) {
	validRule := RateLimitRuleConfig{Max: 10, Window: time.Minute}
	tests := []struct {
		name string
		edit func(*AuthenticationRateLimitConfig)
		want string
	}{
		{
			name: "zero login IP max",
			edit: func(cfg *AuthenticationRateLimitConfig) {
				cfg.Login.PerIP.Max = 0
			},
			want: "AUTH_RATE_LIMIT_LOGIN_IP_MAX",
		},
		{
			name: "zero login IP window",
			edit: func(cfg *AuthenticationRateLimitConfig) {
				cfg.Login.PerIP.Window = 0
			},
			want: "AUTH_RATE_LIMIT_LOGIN_IP_WINDOW",
		},
		{
			name: "subject max without window",
			edit: func(cfg *AuthenticationRateLimitConfig) {
				cfg.PasswordReset.PerSubject = RateLimitRuleConfig{Max: 3}
			},
			want: "AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_WINDOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := AuthenticationRateLimitConfig{
				Enabled: true,
				Login: AuthenticationEndpointRateLimitConfig{
					PerIP:      validRule,
					PerSubject: validRule,
				},
				Register: AuthenticationEndpointRateLimitConfig{PerIP: validRule},
				PasswordReset: AuthenticationEndpointRateLimitConfig{
					PerIP:      validRule,
					PerSubject: validRule,
				},
				PasswordResetConfirm: AuthenticationEndpointRateLimitConfig{
					PerIP:      validRule,
					PerSubject: validRule,
				},
			}
			tt.edit(&auth)

			cfg := baseValidConfig("production")
			cfg.Middleware.AuthenticationRateLimit = auth
			err := validate(cfg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %s error, got %v", tt.want, err)
			}
		})
	}
}

func TestValidate_AllowsDisabledAuthenticationRateLimitWithoutRules(t *testing.T) {
	cfg := baseValidConfig("production")
	cfg.Middleware.AuthenticationRateLimit = AuthenticationRateLimitConfig{Enabled: false}
	if err := validate(cfg); err != nil {
		t.Fatalf("disabled authentication rate limit should not require rules, got %v", err)
	}
}

func TestValidate_RejectsInvalidServerTransportBudgets(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{
			name: "negative read timeout",
			edit: func(cfg *Config) { cfg.Server.ReadTimeout = -1 },
			want: "SERVER_READ_TIMEOUT",
		},
		{
			name: "negative read header timeout",
			edit: func(cfg *Config) { cfg.Server.ReadHeaderTimeout = -1 },
			want: "SERVER_READ_HEADER_TIMEOUT",
		},
		{
			name: "negative write timeout",
			edit: func(cfg *Config) { cfg.Server.WriteTimeout = -1 },
			want: "SERVER_WRITE_TIMEOUT",
		},
		{
			name: "negative idle timeout",
			edit: func(cfg *Config) { cfg.Server.IdleTimeout = -1 },
			want: "SERVER_IDLE_TIMEOUT",
		},
		{
			name: "negative max header bytes",
			edit: func(cfg *Config) { cfg.Server.MaxHeaderBytes = -1 },
			want: "SERVER_MAX_HEADER_BYTES",
		},
		{
			name: "read header exceeds read timeout",
			edit: func(cfg *Config) {
				cfg.Server.ReadTimeout = 5
				cfg.Server.ReadHeaderTimeout = 6
			},
			want: "SERVER_READ_HEADER_TIMEOUT",
		},
		{
			name: "write does not outlive request",
			edit: func(cfg *Config) {
				cfg.Server.WriteTimeout = 180
				cfg.Middleware.RequestTimeout = 180
			},
			want: "SERVER_WRITE_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseValidConfig("production")
			tt.edit(cfg)

			err := validate(cfg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %s error, got %v", tt.want, err)
			}
		})
	}
}

func TestValidate_AllowsDisabledServerWriteTimeout(t *testing.T) {
	cfg := baseValidConfig("production")
	cfg.Server.WriteTimeout = 0
	cfg.Middleware.RequestTimeout = 180

	if err := validate(cfg); err != nil {
		t.Fatalf("disabled server write timeout should be explicit and valid, got %v", err)
	}
}

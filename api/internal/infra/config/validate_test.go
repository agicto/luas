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

func TestValidate_ProductionAliasesUseProductionRules(t *testing.T) {
	for _, environment := range []string{"production", "prod", "release", " RELEASE "} {
		t.Run(environment, func(t *testing.T) {
			cfg := baseValidConfig(environment)
			cfg.JWT.Secret = "short"
			err := validate(cfg)
			if err == nil || !strings.Contains(err.Error(), "32 characters") {
				t.Fatalf("validate() with APP_ENV=%q error = %v, want production secret error", environment, err)
			}
		})
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

func TestValidate_EmailConfigurationIsCompleteAndBounded(t *testing.T) {
	tests := []struct {
		name    string
		config  EmailConfig
		wantErr string
	}{
		{name: "disabled"},
		{
			name: "configured",
			config: EmailConfig{
				From:           "Luas <noreply@example.com>",
				ResendAPIKey:   "resend-key",
				RequestTimeout: 10 * time.Second,
			},
		},
		{
			name:    "missing sender",
			config:  EmailConfig{ResendAPIKey: "resend-key", RequestTimeout: 10 * time.Second},
			wantErr: "MAIL_FROM and RESEND_API_KEY",
		},
		{
			name:    "missing API key",
			config:  EmailConfig{From: "noreply@example.com", RequestTimeout: 10 * time.Second},
			wantErr: "MAIL_FROM and RESEND_API_KEY",
		},
		{
			name: "invalid sender",
			config: EmailConfig{
				From:           "not-an-email",
				ResendAPIKey:   "resend-key",
				RequestTimeout: 10 * time.Second,
			},
			wantErr: "MAIL_FROM",
		},
		{
			name: "zero timeout",
			config: EmailConfig{
				From:         "noreply@example.com",
				ResendAPIKey: "resend-key",
			},
			wantErr: "MAIL_REQUEST_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseValidConfig("production")
			cfg.Email = tt.config
			err := validate(cfg)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_OrganizationInvitationTTLWhenStarterIsSelected(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr bool
	}{
		{name: "positive", ttl: 7 * 24 * time.Hour},
		{name: "zero", wantErr: true},
		{name: "negative", ttl: -time.Hour, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseValidConfig("production")
			cfg.Starters.Optional = []string{"organization"}
			cfg.Organization.InvitationTTL = tt.ttl
			err := validate(cfg)
			if tt.wantErr && (err == nil || !strings.Contains(err.Error(), "ORGANIZATION_INVITATION_TTL")) {
				t.Fatalf("validate() error = %v, want invitation TTL error", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
		})
	}
}

func TestValidate_AssetStoragePolicy(t *testing.T) {
	validAsset := func(environment, driver string) *Config {
		cfg := baseValidConfig(environment)
		cfg.Starters.Optional = []string{"asset"}
		cfg.App.URL = "https://api.example.com"
		cfg.ObjectStorage = ObjectStorageConfig{
			Driver:         driver,
			LocalRoot:      "storage/objects",
			RequestTimeout: 30 * time.Second,
		}
		cfg.Asset = AssetConfig{
			MaxBytes:         DefaultAssetMaxBytes,
			UploadGrantTTL:   DefaultAssetUploadGrantTTL,
			DownloadGrantTTL: DefaultAssetDownloadGrantTTL,
			PendingTTL:       DefaultAssetPendingTTL,
		}
		if driver == "r2" {
			cfg.R2 = R2Config{
				AccessKeyID:     "access-key",
				SecretAccessKey: "secret-key",
				Bucket:          "private-bucket",
				Region:          "auto",
				Endpoint:        "https://account.r2.cloudflarestorage.com",
			}
		}
		return cfg
	}

	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{name: "development local", config: validAsset("development", "local")},
		{name: "production r2", config: validAsset("production", "r2")},
		{name: "disabled selected", config: validAsset("development", "disabled"), wantErr: "must be enabled"},
		{name: "production local", config: validAsset("production", "local"), wantErr: "must be r2"},
		{name: "unknown driver", config: validAsset("development", "s3"), wantErr: "OBJECT_STORAGE_DRIVER"},
		{name: "oversized policy", config: func() *Config {
			cfg := validAsset("development", "local")
			cfg.Asset.MaxBytes = 100*1024*1024 + 1
			return cfg
		}(), wantErr: "ASSET_MAX_BYTES"},
		{name: "pending shorter than grant", config: func() *Config {
			cfg := validAsset("development", "local")
			cfg.Asset.PendingTTL = time.Minute
			return cfg
		}(), wantErr: "ASSET_PENDING_TTL"},
		{name: "partial r2", config: func() *Config {
			cfg := validAsset("development", "local")
			cfg.R2.AccessKeyID = "accidental-key"
			return cfg
		}(), wantErr: "must be configured together"},
		{name: "insecure production endpoint", config: func() *Config {
			cfg := validAsset("production", "r2")
			cfg.R2.Endpoint = "http://r2.internal"
			return cfg
		}(), wantErr: "https in production"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validate(test.config)
			if test.wantErr == "" && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("validate() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestValidate_WebhookPolicy(t *testing.T) {
	validWebhook := func(environment string) *Config {
		cfg := baseValidConfig(environment)
		cfg.Starters.Optional = []string{"organization", "webhook"}
		cfg.Organization.InvitationTTL = DefaultOrganizationInvitationTTL
		cfg.Webhook = WebhookConfig{
			EncryptionKey:    strings.Repeat("w", 32),
			RequestTimeout:   DefaultWebhookRequestTimeout,
			MaxResponseBytes: DefaultWebhookMaxResponseBytes,
			SecretOverlap:    DefaultWebhookSecretOverlap,
			EventRetention:   DefaultWebhookEventRetention,
		}
		return cfg
	}
	tests := []struct {
		name    string
		edit    func(*Config)
		wantErr string
	}{
		{name: "valid production"},
		{name: "missing encryption key", edit: func(cfg *Config) { cfg.Webhook.EncryptionKey = "" }, wantErr: "WEBHOOK_ENCRYPTION_KEY"},
		{name: "request timeout too high", edit: func(cfg *Config) { cfg.Webhook.RequestTimeout = 31 * time.Second }, wantErr: "WEBHOOK_REQUEST_TIMEOUT"},
		{name: "response limit too high", edit: func(cfg *Config) { cfg.Webhook.MaxResponseBytes = 1024*1024 + 1 }, wantErr: "WEBHOOK_MAX_RESPONSE_BYTES"},
		{name: "overlap too long", edit: func(cfg *Config) { cfg.Webhook.SecretOverlap = 8 * 24 * time.Hour }, wantErr: "WEBHOOK_SECRET_OVERLAP"},
		{name: "retention too short", edit: func(cfg *Config) { cfg.Webhook.EventRetention = time.Hour }, wantErr: "WEBHOOK_EVENT_RETENTION"},
		{name: "production insecure HTTP", edit: func(cfg *Config) { cfg.Webhook.AllowInsecureHTTP = true }, wantErr: "WEBHOOK_ALLOW_INSECURE_HTTP"},
		{name: "production private target", edit: func(cfg *Config) { cfg.Webhook.AllowPrivateTargets = true }, wantErr: "WEBHOOK_ALLOW_PRIVATE_TARGETS"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validWebhook("production")
			if test.edit != nil {
				test.edit(cfg)
			}
			err := validate(cfg)
			if test.wantErr == "" && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("validate() error = %v, want %q", err, test.wantErr)
			}
		})
	}

	development := validWebhook("development")
	development.Webhook.AllowInsecureHTTP = true
	development.Webhook.AllowPrivateTargets = true
	if err := validate(development); err != nil {
		t.Fatalf("development overrides should be explicit but valid: %v", err)
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

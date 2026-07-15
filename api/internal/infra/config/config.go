package config

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/zgiai/luas/api/pkg/env"
)

// GlobalConfig stores the global configuration
var GlobalConfig *Config

const (
	// DefaultServerHost keeps local runs off external interfaces unless explicitly configured.
	DefaultServerHost = "127.0.0.1"
	// DefaultServerPort is shared by the HTTP server and local health probe.
	DefaultServerPort = 8025

	defaultServerReadTimeoutSeconds       = 60
	defaultServerReadHeaderTimeoutSeconds = 10
	defaultServerWriteTimeoutSeconds      = 190
	defaultServerIdleTimeoutSeconds       = 120
	defaultServerMaxHeaderBytes           = 64 * 1024

	// DefaultMiddlewareRequestTimeoutSeconds is shared by config loading and timeout middleware.
	DefaultMiddlewareRequestTimeoutSeconds = 180
	// DefaultEmailRequestTimeout caps one outbound provider call.
	DefaultEmailRequestTimeout = 10 * time.Second
	// DefaultAIRequestTimeout caps one complete provider call or streaming session.
	DefaultAIRequestTimeout = 120 * time.Second
	// DefaultAIMaxInputBytes bounds input plus instructions before provider serialization.
	DefaultAIMaxInputBytes = 1024 * 1024
	// DefaultAIMaxResponseBytes bounds a decompressed one-shot provider response.
	DefaultAIMaxResponseBytes int64 = 4 * 1024 * 1024
	// DefaultAIMaxStreamEventBytes bounds one provider SSE event line.
	DefaultAIMaxStreamEventBytes = 1024 * 1024
	// DefaultOrganizationInvitationTTL bounds one organization invitation token.
	DefaultOrganizationInvitationTTL = 7 * 24 * time.Hour
	// DefaultObjectStorageRequestTimeout bounds one provider metadata or object operation.
	DefaultObjectStorageRequestTimeout = 30 * time.Second
	// DefaultAssetMaxBytes bounds the first single-object asset workflow.
	DefaultAssetMaxBytes int64 = 10 * 1024 * 1024
	// DefaultAssetUploadGrantTTL keeps upload bearer credentials short-lived.
	DefaultAssetUploadGrantTTL = 10 * time.Minute
	// DefaultAssetDownloadGrantTTL keeps download bearer credentials short-lived.
	DefaultAssetDownloadGrantTTL = 5 * time.Minute
	// DefaultAssetPendingTTL bounds incomplete staging-object lifetime.
	DefaultAssetPendingTTL = time.Hour
	// DefaultWebhookRequestTimeout bounds one outbound receiver call.
	DefaultWebhookRequestTimeout = 15 * time.Second
	// DefaultWebhookMaxResponseBytes bounds receiver response draining.
	DefaultWebhookMaxResponseBytes int64 = 64 * 1024
	// DefaultWebhookSecretOverlap permits zero-downtime consumer key rotation.
	DefaultWebhookSecretOverlap = 24 * time.Hour
	// DefaultWebhookEventRetention bounds the built-in replay horizon.
	DefaultWebhookEventRetention = 30 * 24 * time.Hour
)

// Config holds all application configuration
type Config struct {
	App           AppConfig
	Starters      StarterConfig
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Queue         QueueConfig
	Scheduler     SchedulerConfig
	JWT           JWTConfig
	Log           LogConfig
	Sentry        SentryConfig
	CORS          CORSConfig
	Email         EmailConfig
	Organization  OrganizationConfig
	AI            AIConfig
	ObjectStorage ObjectStorageConfig
	Asset         AssetConfig
	Webhook       WebhookConfig
	R2            R2Config
	Middleware    MiddlewareConfig
	Metrics       MetricsConfig
	Tracing       TracingConfig
}

// StarterConfig controls additive activation of starters that are not part of the defaults.
type StarterConfig struct {
	Optional []string
}

// IsProduction reports whether this snapshot uses a production environment
// alias. Keep production-sensitive defaults and validation on this method.
func (c *Config) IsProduction() bool {
	return c != nil && isProductionEnvironment(c.App.Env)
}

type AppConfig struct {
	Name  string
	Env   string
	Debug bool
	URL   string
}

type ServerConfig struct {
	Host              string
	Port              int
	Mode              string
	ReadTimeout       int
	ReadHeaderTimeout int
	WriteTimeout      int
	IdleTimeout       int
	MaxHeaderBytes    int
	TrustedProxies    []string
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	RequestTimeout          int                           // Request timeout in seconds, default 180 (3 min)
	BodyLimit               int64                         // Max body size in bytes, default 10MB
	RateLimit               RateLimitConfig               // Production default request rate guardrail
	AuthenticationRateLimit AuthenticationRateLimitConfig // Public authentication abuse guardrails
}

// RateLimitConfig holds default HTTP rate limit configuration.
type RateLimitConfig struct {
	Enabled   bool
	Max       int
	Window    time.Duration
	SkipPaths []string
}

// RateLimitRuleConfig defines one fixed-window quota.
type RateLimitRuleConfig struct {
	Max    int
	Window time.Duration
}

// AuthenticationEndpointRateLimitConfig applies independent quotas to the
// request source and the normalized, hashed subject targeted by the request.
type AuthenticationEndpointRateLimitConfig struct {
	PerIP      RateLimitRuleConfig
	PerSubject RateLimitRuleConfig
}

// AuthenticationRateLimitConfig controls public authentication endpoint
// guardrails. Per-subject rules may be left at zero when they do not add a
// meaningful boundary for an endpoint.
type AuthenticationRateLimitConfig struct {
	Enabled              bool
	Login                AuthenticationEndpointRateLimitConfig
	Register             AuthenticationEndpointRateLimitConfig
	PasswordReset        AuthenticationEndpointRateLimitConfig
	PasswordResetConfirm AuthenticationEndpointRateLimitConfig
}

// MetricsConfig controls HTTP request instrumentation and the Prometheus endpoint.
type MetricsConfig struct {
	Enabled bool
}

type DatabaseConfig struct {
	Enabled              bool
	Driver               string
	Host                 string
	Port                 int
	Name                 string
	Username             string
	Password             string
	SSLMode              string
	Timezone             string
	MaxIdleConns         int
	MaxOpenConns         int
	ConnMaxLifetime      time.Duration
	Memory               bool
	LogLevel             string
	SlowThreshold        time.Duration
	IgnoreRecordNotFound bool
}

// DBName returns the database name (alias for Name)
func (d DatabaseConfig) DBName() string {
	return d.Name
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type QueueConfig struct {
	Driver            string
	DefaultQueue      string
	BufferSize        int
	WorkerConcurrency int
	WorkerSleep       time.Duration
	WorkerTimeout     time.Duration
}

type SchedulerConfig struct {
	Enabled bool
}

type JWTConfig struct {
	Secret     string
	ExpireDays int
	Expire     time.Duration
}

// ExpireDuration returns the expiration duration (alias for Expire)
func (j JWTConfig) ExpireDuration() time.Duration {
	return j.Expire
}

type LogConfig struct {
	Level       string
	File        string
	Stdout      bool
	FileEnabled bool
	JSON        bool
}

type SentryConfig struct {
	DSN string
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
}

type EmailConfig struct {
	From           string
	ResendAPIKey   string
	RequestTimeout time.Duration
}

// OrganizationConfig holds optional organization starter policy.
type OrganizationConfig struct {
	InvitationTTL time.Duration
}

type AIProviderConfig struct {
	APIKey  string
	BaseURL string
}

type AIConfig struct {
	Enabled             bool
	DefaultProvider     string
	DefaultModel        string
	RequestTimeout      time.Duration
	MaxInputBytes       int
	MaxResponseBytes    int64
	MaxStreamEventBytes int
	OpenAI              AIProviderConfig
	// To add a new provider: add a field here, wire it in config.Load,
	// and register the provider in ai.NewManager.
}

type R2Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	Endpoint        string
}

// ObjectStorageConfig selects a provider-neutral object adapter.
type ObjectStorageConfig struct {
	Driver         string
	LocalRoot      string
	RequestTimeout time.Duration
}

// AssetConfig owns policy for the optional asset starter rather than provider details.
type AssetConfig struct {
	MaxBytes         int64
	UploadGrantTTL   time.Duration
	DownloadGrantTTL time.Duration
	PendingTTL       time.Duration
}

// WebhookConfig owns outbound delivery safety and retention policy.
type WebhookConfig struct {
	EncryptionKey       string
	RequestTimeout      time.Duration
	MaxResponseBytes    int64
	SecretOverlap       time.Duration
	EventRetention      time.Duration
	AllowInsecureHTTP   bool
	AllowPrivateTargets bool
}

// TracingConfig holds OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled    bool
	Endpoint   string  // OTLP endpoint (e.g., "localhost:4317")
	Insecure   bool    // Use insecure connection
	SampleRate float64 // Sampling rate (0.0 to 1.0)
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	if err := env.Load(); err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}

	appEnv := env.AppEnv()
	isProd := isProductionEnvironment(appEnv)
	appDebug := env.GetBool("APP_DEBUG", !isProd)
	expireDays := env.GetInt("JWT_EXPIRE_DAYS", 7)

	cfg := &Config{
		App: AppConfig{
			Name:  env.Get("APP_NAME", "Luas"),
			Env:   appEnv,
			Debug: appDebug,
			URL:   env.Get("APP_URL", "http://localhost:8025"),
		},
		Starters: StarterConfig{
			Optional: env.GetSlice("OPTIONAL_STARTERS", []string{}),
		},
		Server: ServerConfig{
			Host:              env.Get("SERVER_HOST", DefaultServerHost),
			Port:              env.GetInt("SERVER_PORT", DefaultServerPort),
			Mode:              env.Get("SERVER_MODE", env.Get("GIN_MODE", defaultServerMode(isProd))),
			ReadTimeout:       env.GetInt("SERVER_READ_TIMEOUT", defaultServerReadTimeoutSeconds),
			ReadHeaderTimeout: env.GetInt("SERVER_READ_HEADER_TIMEOUT", defaultServerReadHeaderTimeoutSeconds),
			WriteTimeout:      env.GetInt("SERVER_WRITE_TIMEOUT", defaultServerWriteTimeoutSeconds),
			IdleTimeout:       env.GetInt("SERVER_IDLE_TIMEOUT", defaultServerIdleTimeoutSeconds),
			MaxHeaderBytes:    env.GetInt("SERVER_MAX_HEADER_BYTES", defaultServerMaxHeaderBytes),
			TrustedProxies:    env.GetSlice("SERVER_TRUSTED_PROXIES", []string{}),
		},
		Database: DatabaseConfig{
			Enabled:              env.GetBool("DB_ENABLED", true),
			Driver:               env.Get("DB_DRIVER", "postgres"),
			Host:                 env.Get("DB_HOST", "localhost"),
			Port:                 env.GetInt("DB_PORT", 5432),
			Name:                 env.Get("DB_NAME", ""),
			Username:             env.Get("DB_USERNAME", ""),
			Password:             env.Get("DB_PASSWORD", ""),
			SSLMode:              env.Get("DB_SSLMODE", "disable"),
			Timezone:             env.Get("DB_TIMEZONE", "UTC"),
			MaxIdleConns:         env.GetInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:         env.GetInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime:      time.Duration(env.GetInt("DB_CONN_MAX_LIFETIME", 3600)) * time.Second,
			LogLevel:             env.Get("DB_LOG_LEVEL", ""),
			SlowThreshold:        env.GetDuration("DB_SLOW_THRESHOLD", time.Second),
			IgnoreRecordNotFound: env.GetBool("DB_LOG_IGNORE_NOT_FOUND", true),
		},
		Redis: RedisConfig{
			Host:     env.Get("REDIS_HOST", "localhost"),
			Port:     env.GetInt("REDIS_PORT", 6379),
			Password: env.Get("REDIS_PASSWORD", ""),
			DB:       env.GetInt("REDIS_DB", 0),
		},
		Queue: QueueConfig{
			Driver:            env.Get("QUEUE_DRIVER", "sync"),
			DefaultQueue:      env.Get("QUEUE_DEFAULT", "default"),
			BufferSize:        env.GetInt("QUEUE_BUFFER_SIZE", 256),
			WorkerConcurrency: env.GetInt("QUEUE_WORKER_CONCURRENCY", 1),
			WorkerSleep:       env.GetDuration("QUEUE_WORKER_SLEEP", time.Second),
			WorkerTimeout:     env.GetDuration("QUEUE_WORKER_TIMEOUT", 60*time.Second),
		},
		Scheduler: SchedulerConfig{
			Enabled: env.GetBool("SCHEDULER_ENABLED", false),
		},
		JWT: JWTConfig{
			Secret:     env.Get("JWT_SECRET", ""),
			ExpireDays: expireDays,
			Expire:     time.Duration(expireDays) * 24 * time.Hour,
		},
		Log: LogConfig{
			Level:       env.Get("LOG_LEVEL", defaultLogLevel(isProd)),
			File:        env.Get("LOG_FILE", env.Get("LOG_FILENAME", "storage/logs/app.log")),
			Stdout:      env.GetBool("LOG_STDOUT", true),
			FileEnabled: env.GetBool("LOG_FILE_ENABLED", !isProd),
			JSON:        env.GetBool("LOG_JSON", isProd),
		},
		Sentry: SentryConfig{
			DSN: env.Get("SENTRY_DSN", ""),
		},
		CORS: CORSConfig{
			// Default to localhost-only. Production should set CORS_ALLOW_ORIGINS
			// explicitly to a comma-separated list. "*" is rejected by validate()
			// whenever AllowCredentials is true (browsers reject the combo anyway).
			AllowOrigins:     env.GetSlice("CORS_ALLOW_ORIGINS", []string{"http://localhost:3000"}),
			AllowMethods:     env.GetSlice("CORS_ALLOW_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowHeaders:     env.GetSlice("CORS_ALLOW_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization", "Organization-Id", "Idempotency-Key", "If-Match", "X-Request-ID"}),
			ExposeHeaders:    env.GetSlice("CORS_EXPOSE_HEADERS", []string{"Content-Length", "ETag", "X-Request-ID"}),
			AllowCredentials: env.GetBool("CORS_ALLOW_CREDENTIALS", true),
		},
		Email: EmailConfig{
			From:           env.Get("MAIL_FROM", ""),
			ResendAPIKey:   env.Get("RESEND_API_KEY", ""),
			RequestTimeout: env.GetDuration("MAIL_REQUEST_TIMEOUT", DefaultEmailRequestTimeout),
		},
		Organization: OrganizationConfig{
			InvitationTTL: env.GetDuration("ORGANIZATION_INVITATION_TTL", DefaultOrganizationInvitationTTL),
		},
		AI: loadAIConfig(),
		ObjectStorage: ObjectStorageConfig{
			Driver:         env.Get("OBJECT_STORAGE_DRIVER", defaultObjectStorageDriver(env.GetSlice("OPTIONAL_STARTERS", []string{}), isProd)),
			LocalRoot:      env.Get("OBJECT_STORAGE_LOCAL_ROOT", "storage/objects"),
			RequestTimeout: env.GetDuration("OBJECT_STORAGE_REQUEST_TIMEOUT", DefaultObjectStorageRequestTimeout),
		},
		Asset: AssetConfig{
			MaxBytes:         int64(env.GetInt("ASSET_MAX_BYTES", int(DefaultAssetMaxBytes))),
			UploadGrantTTL:   env.GetDuration("ASSET_UPLOAD_GRANT_TTL", DefaultAssetUploadGrantTTL),
			DownloadGrantTTL: env.GetDuration("ASSET_DOWNLOAD_GRANT_TTL", DefaultAssetDownloadGrantTTL),
			PendingTTL:       env.GetDuration("ASSET_PENDING_TTL", DefaultAssetPendingTTL),
		},
		Webhook: WebhookConfig{
			EncryptionKey:       env.Get("WEBHOOK_ENCRYPTION_KEY", ""),
			RequestTimeout:      env.GetDuration("WEBHOOK_REQUEST_TIMEOUT", DefaultWebhookRequestTimeout),
			MaxResponseBytes:    int64(env.GetInt("WEBHOOK_MAX_RESPONSE_BYTES", int(DefaultWebhookMaxResponseBytes))),
			SecretOverlap:       env.GetDuration("WEBHOOK_SECRET_OVERLAP", DefaultWebhookSecretOverlap),
			EventRetention:      env.GetDuration("WEBHOOK_EVENT_RETENTION", DefaultWebhookEventRetention),
			AllowInsecureHTTP:   env.GetBool("WEBHOOK_ALLOW_INSECURE_HTTP", false),
			AllowPrivateTargets: env.GetBool("WEBHOOK_ALLOW_PRIVATE_TARGETS", false),
		},
		R2: R2Config{
			AccessKeyID:     env.Get("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: env.Get("R2_SECRET_ACCESS_KEY", ""),
			Bucket:          env.Get("R2_BUCKET", ""),
			Region:          env.Get("R2_REGION", "auto"),
			Endpoint:        env.Get("R2_ENDPOINT", ""),
		},
		Middleware: MiddlewareConfig{
			RequestTimeout: env.GetInt("MIDDLEWARE_REQUEST_TIMEOUT", DefaultMiddlewareRequestTimeoutSeconds),
			BodyLimit:      int64(env.GetInt("MIDDLEWARE_BODY_LIMIT_MB", 10)) * 1024 * 1024,
			RateLimit: RateLimitConfig{
				Enabled: env.GetBool("MIDDLEWARE_RATE_LIMIT_ENABLED", isProd),
				Max:     env.GetInt("MIDDLEWARE_RATE_LIMIT_MAX", 600),
				Window:  env.GetDuration("MIDDLEWARE_RATE_LIMIT_WINDOW", time.Minute),
				SkipPaths: env.GetSlice("MIDDLEWARE_RATE_LIMIT_SKIP_PATHS", []string{
					"/health",
					"/health/live",
					"/health/ready",
					"/metrics",
					"/v1/health",
				}),
			},
			AuthenticationRateLimit: AuthenticationRateLimitConfig{
				Enabled: env.GetBool("AUTH_RATE_LIMIT_ENABLED", isProd),
				Login: AuthenticationEndpointRateLimitConfig{
					PerIP: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_LOGIN_IP_MAX", 20),
						Window: env.GetDuration("AUTH_RATE_LIMIT_LOGIN_IP_WINDOW", 5*time.Minute),
					},
					PerSubject: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_LOGIN_SUBJECT_MAX", 10),
						Window: env.GetDuration("AUTH_RATE_LIMIT_LOGIN_SUBJECT_WINDOW", 15*time.Minute),
					},
				},
				Register: AuthenticationEndpointRateLimitConfig{
					PerIP: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_REGISTER_IP_MAX", 10),
						Window: env.GetDuration("AUTH_RATE_LIMIT_REGISTER_IP_WINDOW", time.Hour),
					},
					PerSubject: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_REGISTER_SUBJECT_MAX", 0),
						Window: env.GetDuration("AUTH_RATE_LIMIT_REGISTER_SUBJECT_WINDOW", 0),
					},
				},
				PasswordReset: AuthenticationEndpointRateLimitConfig{
					PerIP: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_PASSWORD_RESET_IP_MAX", 10),
						Window: env.GetDuration("AUTH_RATE_LIMIT_PASSWORD_RESET_IP_WINDOW", time.Hour),
					},
					PerSubject: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_MAX", 3),
						Window: env.GetDuration("AUTH_RATE_LIMIT_PASSWORD_RESET_SUBJECT_WINDOW", time.Hour),
					},
				},
				PasswordResetConfirm: AuthenticationEndpointRateLimitConfig{
					PerIP: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_IP_MAX", 10),
						Window: env.GetDuration("AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_IP_WINDOW", 15*time.Minute),
					},
					PerSubject: RateLimitRuleConfig{
						Max:    env.GetInt("AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_SUBJECT_MAX", 5),
						Window: env.GetDuration("AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM_SUBJECT_WINDOW", 15*time.Minute),
					},
				},
			},
		},
		Metrics: MetricsConfig{
			Enabled: env.GetBool("METRICS_ENABLED", !isProd),
		},
		Tracing: TracingConfig{
			Enabled:    env.GetBool("TRACING_ENABLED", false),
			Endpoint:   env.Get("TRACING_ENDPOINT", "localhost:4317"),
			Insecure:   env.GetBool("TRACING_INSECURE", true),
			SampleRate: env.GetFloat("TRACING_SAMPLE_RATE", 1.0),
		},
	}

	// Validate required fields
	if err := validate(cfg); err != nil {
		return nil, err
	}

	GlobalConfig = cfg
	return cfg, nil
}

// LoadAIConfig loads only the typed AI capability settings. It is used by
// provider utilities that do not assemble the HTTP/database runtime.
func LoadAIConfig() (AIConfig, error) {
	if err := env.Load(); err != nil {
		return AIConfig{}, fmt.Errorf("load environment: %w", err)
	}
	cfg := loadAIConfig()
	if err := validateAIConfig(cfg, isProductionEnvironment(env.AppEnv())); err != nil {
		return AIConfig{}, err
	}
	return cfg, nil
}

func loadAIConfig() AIConfig {
	return AIConfig{
		Enabled:             env.GetBool("AI_ENABLED", false),
		DefaultProvider:     env.Get("AI_DEFAULT_PROVIDER", "openai"),
		DefaultModel:        env.Get("AI_DEFAULT_MODEL", ""),
		RequestTimeout:      env.GetDuration("AI_REQUEST_TIMEOUT", DefaultAIRequestTimeout),
		MaxInputBytes:       env.GetInt("AI_MAX_INPUT_BYTES", DefaultAIMaxInputBytes),
		MaxResponseBytes:    int64(env.GetInt("AI_MAX_RESPONSE_BYTES", int(DefaultAIMaxResponseBytes))),
		MaxStreamEventBytes: env.GetInt("AI_MAX_STREAM_EVENT_BYTES", DefaultAIMaxStreamEventBytes),
		OpenAI: AIProviderConfig{
			APIKey:  env.Get("OPENAI_API_KEY", ""),
			BaseURL: env.Get("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		},
	}
}

func defaultLogLevel(production bool) string {
	if production {
		return "info"
	}
	return "debug"
}

func defaultServerMode(production bool) string {
	if production {
		return "release"
	}
	return "debug"
}

// MustLoad loads configuration or panics
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// placeholderJWTSecrets are values shipped with .env.example or previously
// used by the scaffold; treating them as "real" would silently weaken auth.
var placeholderJWTSecrets = map[string]struct{}{
	"": {},
	"replace_me_with_a_long_random_secret_at_least_32_chars": {},
	"your_jwt_secret_key_here":                               {},
	"replace-me":                                             {},
	"change_me_in_production":                                {},
}

func validate(cfg *Config) error {
	if cfg.Database.Enabled && cfg.Database.Driver != "sqlite" && cfg.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required when database is enabled")
	}

	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	isProd := cfg.IsProduction()
	if err := validateAIConfig(cfg.AI, isProd); err != nil {
		return err
	}

	// JWT_SECRET strength: in production, reject known placeholders and
	// secrets shorter than 32 chars. In other envs, log a clear warning by
	// returning a non-fatal hint — but we don't have a logger here yet, so
	// we keep the hard rule production-only.
	if isProd {
		if _, isPlaceholder := placeholderJWTSecrets[cfg.JWT.Secret]; isPlaceholder {
			return fmt.Errorf("JWT_SECRET is set to a known placeholder value; generate one with `openssl rand -hex 32`")
		}
		if len(cfg.JWT.Secret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in production (current: %d)", len(cfg.JWT.Secret))
		}
	}

	if err := validateEmailConfig(cfg.Email); err != nil {
		return err
	}
	if cfg.Organization.InvitationTTL < 0 ||
		(slices.Contains(cfg.Starters.Optional, "organization") && cfg.Organization.InvitationTTL == 0) {
		return fmt.Errorf("ORGANIZATION_INVITATION_TTL must be greater than 0 when the organization starter is selected")
	}
	assetSelected := slices.Contains(cfg.Starters.Optional, "asset")
	if err := validateObjectStorageConfig(cfg, assetSelected); err != nil {
		return err
	}
	if err := validateAssetConfig(cfg.Asset, assetSelected); err != nil {
		return err
	}
	webhookSelected := slices.Contains(cfg.Starters.Optional, "webhook")
	if err := validateWebhookConfig(cfg, webhookSelected); err != nil {
		return err
	}

	// CORS: wildcard origin + credentials is rejected by browsers anyway.
	// Catch the misconfiguration early at startup.
	if cfg.CORS.AllowCredentials {
		for _, origin := range cfg.CORS.AllowOrigins {
			if strings.TrimSpace(origin) == "*" {
				return fmt.Errorf("CORS_ALLOW_ORIGINS cannot contain '*' when CORS_ALLOW_CREDENTIALS is true; list explicit origins instead")
			}
		}
	}

	// In production, refuse to start with localhost origins — almost
	// certainly a forgotten config.
	if isProd {
		for _, origin := range cfg.CORS.AllowOrigins {
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				return fmt.Errorf("CORS_ALLOW_ORIGINS contains a localhost origin in production: %q", origin)
			}
		}
	}

	if cfg.Middleware.RateLimit.Enabled {
		if cfg.Middleware.RateLimit.Max <= 0 {
			return fmt.Errorf("MIDDLEWARE_RATE_LIMIT_MAX must be greater than 0 when rate limit is enabled")
		}
		if cfg.Middleware.RateLimit.Window <= 0 {
			return fmt.Errorf("MIDDLEWARE_RATE_LIMIT_WINDOW must be greater than 0 when rate limit is enabled")
		}
	}

	if err := validateServerTransport(cfg.Server, cfg.Middleware); err != nil {
		return err
	}

	if err := validateTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return err
	}

	if cfg.Middleware.AuthenticationRateLimit.Enabled {
		endpoints := []struct {
			prefix string
			config AuthenticationEndpointRateLimitConfig
		}{
			{prefix: "AUTH_RATE_LIMIT_LOGIN", config: cfg.Middleware.AuthenticationRateLimit.Login},
			{prefix: "AUTH_RATE_LIMIT_REGISTER", config: cfg.Middleware.AuthenticationRateLimit.Register},
			{prefix: "AUTH_RATE_LIMIT_PASSWORD_RESET", config: cfg.Middleware.AuthenticationRateLimit.PasswordReset},
			{prefix: "AUTH_RATE_LIMIT_PASSWORD_RESET_CONFIRM", config: cfg.Middleware.AuthenticationRateLimit.PasswordResetConfirm},
		}
		for _, endpoint := range endpoints {
			if err := validateRateLimitRule(endpoint.prefix+"_IP", endpoint.config.PerIP, true); err != nil {
				return err
			}
			if err := validateRateLimitRule(endpoint.prefix+"_SUBJECT", endpoint.config.PerSubject, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateAIConfig(aiConfig AIConfig, production bool) error {
	if !aiConfig.Enabled {
		return nil
	}
	if aiConfig.RequestTimeout <= 0 || aiConfig.RequestTimeout > 15*time.Minute {
		return fmt.Errorf("AI_REQUEST_TIMEOUT must be greater than 0 and no more than 15m")
	}
	if aiConfig.MaxInputBytes < 1024 || aiConfig.MaxInputBytes > 16*1024*1024 {
		return fmt.Errorf("AI_MAX_INPUT_BYTES must be between 1024 and 16777216")
	}
	if aiConfig.MaxResponseBytes < 1024 || aiConfig.MaxResponseBytes > 32*1024*1024 {
		return fmt.Errorf("AI_MAX_RESPONSE_BYTES must be between 1024 and 33554432")
	}
	if aiConfig.MaxStreamEventBytes < 1024 || aiConfig.MaxStreamEventBytes > 4*1024*1024 {
		return fmt.Errorf("AI_MAX_STREAM_EVENT_BYTES must be between 1024 and 4194304")
	}
	if int64(aiConfig.MaxStreamEventBytes) > aiConfig.MaxResponseBytes {
		return fmt.Errorf("AI_MAX_STREAM_EVENT_BYTES must not exceed AI_MAX_RESPONSE_BYTES")
	}
	provider := strings.ToLower(strings.TrimSpace(aiConfig.DefaultProvider))
	if !validAIIdentifier(provider, 64) {
		return fmt.Errorf("AI_DEFAULT_PROVIDER must be a valid provider identifier")
	}
	if !validAIIdentifier(strings.TrimSpace(aiConfig.DefaultModel), 256) {
		return fmt.Errorf("AI_DEFAULT_MODEL must be an explicit valid provider model identifier when AI is enabled")
	}
	if provider != "openai" {
		return fmt.Errorf("AI_DEFAULT_PROVIDER %q is not registered by this scaffold", provider)
	}
	if strings.TrimSpace(aiConfig.OpenAI.APIKey) == "" {
		return fmt.Errorf("OPENAI_API_KEY is required when AI_ENABLED is true and AI_DEFAULT_PROVIDER is openai")
	}
	endpoint, err := url.Parse(strings.TrimSpace(aiConfig.OpenAI.BaseURL))
	if err != nil || !endpoint.IsAbs() || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("OPENAI_BASE_URL must be an absolute http or https URL without credentials, query, or fragment")
	}
	if endpoint.Scheme != "https" && endpoint.Scheme != "http" {
		return fmt.Errorf("OPENAI_BASE_URL must use http or https")
	}
	if production && endpoint.Scheme != "https" {
		return fmt.Errorf("OPENAI_BASE_URL must use https in production")
	}
	return nil
}

func validAIIdentifier(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if unicode.IsSpace(char) || unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func defaultObjectStorageDriver(optionalStarters []string, production bool) string {
	if slices.Contains(optionalStarters, "asset") && !production {
		return "local"
	}
	return "disabled"
}

func validateObjectStorageConfig(cfg *Config, assetSelected bool) error {
	driver := strings.TrimSpace(cfg.ObjectStorage.Driver)
	if driver == "" {
		driver = "disabled"
	}
	if driver != "disabled" && driver != "local" && driver != "r2" {
		return fmt.Errorf("OBJECT_STORAGE_DRIVER must be one of disabled, local, or r2")
	}
	if cfg.ObjectStorage.RequestTimeout < 0 ||
		(driver != "disabled" && cfg.ObjectStorage.RequestTimeout == 0) {
		return fmt.Errorf("OBJECT_STORAGE_REQUEST_TIMEOUT must be greater than 0 when object storage is enabled")
	}
	if driver == "local" && strings.TrimSpace(cfg.ObjectStorage.LocalRoot) == "" {
		return fmt.Errorf("OBJECT_STORAGE_LOCAL_ROOT is required for the local object storage driver")
	}
	if assetSelected && driver == "disabled" {
		return fmt.Errorf("OBJECT_STORAGE_DRIVER must be enabled when the asset starter is selected")
	}
	if assetSelected && cfg.IsProduction() && driver != "r2" {
		return fmt.Errorf("OBJECT_STORAGE_DRIVER must be r2 for the asset starter in production")
	}

	r2Values := []string{
		strings.TrimSpace(cfg.R2.AccessKeyID),
		strings.TrimSpace(cfg.R2.SecretAccessKey),
		strings.TrimSpace(cfg.R2.Bucket),
		strings.TrimSpace(cfg.R2.Endpoint),
	}
	configured := 0
	for _, value := range r2Values {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(r2Values) {
		return fmt.Errorf("R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, and R2_ENDPOINT must be configured together")
	}
	if driver == "r2" && configured != len(r2Values) {
		return fmt.Errorf("R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, and R2_ENDPOINT are required for the r2 object storage driver")
	}
	if configured == len(r2Values) {
		endpoint, err := url.Parse(r2Values[3])
		if err != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
			return fmt.Errorf("R2_ENDPOINT must be an absolute http or https origin without credentials, path, query, or fragment")
		}
		if endpoint.Scheme != "https" && endpoint.Scheme != "http" {
			return fmt.Errorf("R2_ENDPOINT must use http or https")
		}
		if cfg.IsProduction() && endpoint.Scheme != "https" {
			return fmt.Errorf("R2_ENDPOINT must use https in production")
		}
		if strings.TrimSpace(cfg.R2.Region) == "" {
			return fmt.Errorf("R2_REGION is required when R2 is configured")
		}
	}
	return nil
}

func validateAssetConfig(asset AssetConfig, selected bool) error {
	if !selected && asset == (AssetConfig{}) {
		return nil
	}
	if asset.MaxBytes <= 0 || asset.MaxBytes > 100*1024*1024 {
		return fmt.Errorf("ASSET_MAX_BYTES must be between 1 and 104857600")
	}
	if asset.UploadGrantTTL <= 0 || asset.UploadGrantTTL > time.Hour {
		return fmt.Errorf("ASSET_UPLOAD_GRANT_TTL must be greater than 0 and no more than 1h")
	}
	if asset.DownloadGrantTTL <= 0 || asset.DownloadGrantTTL > 15*time.Minute {
		return fmt.Errorf("ASSET_DOWNLOAD_GRANT_TTL must be greater than 0 and no more than 15m")
	}
	if asset.PendingTTL < asset.UploadGrantTTL || asset.PendingTTL > 24*time.Hour {
		return fmt.Errorf("ASSET_PENDING_TTL must be at least ASSET_UPLOAD_GRANT_TTL and no more than 24h")
	}
	return nil
}

func validateWebhookConfig(cfg *Config, selected bool) error {
	webhook := cfg.Webhook
	if !selected && webhook == (WebhookConfig{}) {
		return nil
	}
	if selected && len(strings.TrimSpace(webhook.EncryptionKey)) < 32 {
		return fmt.Errorf("WEBHOOK_ENCRYPTION_KEY must be at least 32 characters when the webhook starter is selected")
	}
	if webhook.RequestTimeout <= 0 || webhook.RequestTimeout > 30*time.Second {
		return fmt.Errorf("WEBHOOK_REQUEST_TIMEOUT must be greater than 0 and no more than 30s")
	}
	if webhook.MaxResponseBytes < 1024 || webhook.MaxResponseBytes > 1024*1024 {
		return fmt.Errorf("WEBHOOK_MAX_RESPONSE_BYTES must be between 1024 and 1048576")
	}
	if webhook.SecretOverlap < time.Minute || webhook.SecretOverlap > 7*24*time.Hour {
		return fmt.Errorf("WEBHOOK_SECRET_OVERLAP must be between 1m and 168h")
	}
	if webhook.EventRetention < 24*time.Hour || webhook.EventRetention > 90*24*time.Hour {
		return fmt.Errorf("WEBHOOK_EVENT_RETENTION must be between 24h and 2160h")
	}
	if cfg.IsProduction() && webhook.AllowInsecureHTTP {
		return fmt.Errorf("WEBHOOK_ALLOW_INSECURE_HTTP cannot be enabled in production")
	}
	if cfg.IsProduction() && webhook.AllowPrivateTargets {
		return fmt.Errorf("WEBHOOK_ALLOW_PRIVATE_TARGETS cannot be enabled in production")
	}
	return nil
}

func validateEmailConfig(emailConfig EmailConfig) error {
	fromConfigured := strings.TrimSpace(emailConfig.From) != ""
	apiKeyConfigured := strings.TrimSpace(emailConfig.ResendAPIKey) != ""
	if fromConfigured != apiKeyConfigured {
		return fmt.Errorf("MAIL_FROM and RESEND_API_KEY must be configured together")
	}
	if emailConfig.RequestTimeout < 0 {
		return fmt.Errorf("MAIL_REQUEST_TIMEOUT must not be negative")
	}
	if !fromConfigured {
		return nil
	}
	if emailConfig.RequestTimeout == 0 {
		return fmt.Errorf("MAIL_REQUEST_TIMEOUT must be greater than 0 when email is configured")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(emailConfig.From)); err != nil {
		return fmt.Errorf("MAIL_FROM must be a valid email address: %w", err)
	}
	return nil
}

func validateServerTransport(server ServerConfig, middleware MiddlewareConfig) error {
	values := []struct {
		name  string
		value int
	}{
		{name: "SERVER_READ_TIMEOUT", value: server.ReadTimeout},
		{name: "SERVER_READ_HEADER_TIMEOUT", value: server.ReadHeaderTimeout},
		{name: "SERVER_WRITE_TIMEOUT", value: server.WriteTimeout},
		{name: "SERVER_IDLE_TIMEOUT", value: server.IdleTimeout},
		{name: "SERVER_MAX_HEADER_BYTES", value: server.MaxHeaderBytes},
		{name: "MIDDLEWARE_REQUEST_TIMEOUT", value: middleware.RequestTimeout},
	}
	for _, item := range values {
		if item.value < 0 {
			return fmt.Errorf("%s must not be negative", item.name)
		}
	}

	if server.ReadTimeout > 0 && server.ReadHeaderTimeout > server.ReadTimeout {
		return fmt.Errorf("SERVER_READ_HEADER_TIMEOUT must not exceed SERVER_READ_TIMEOUT")
	}

	requestTimeout := middleware.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = DefaultMiddlewareRequestTimeoutSeconds
	}
	if server.WriteTimeout > 0 && server.WriteTimeout <= requestTimeout {
		return fmt.Errorf(
			"SERVER_WRITE_TIMEOUT must exceed MIDDLEWARE_REQUEST_TIMEOUT (%d <= %d); set SERVER_WRITE_TIMEOUT=0 only when intentionally disabling the transport write deadline",
			server.WriteTimeout,
			requestTimeout,
		)
	}

	return nil
}

func validateTrustedProxies(proxies []string) error {
	for _, value := range proxies {
		proxy := strings.TrimSpace(value)
		if proxy == "" {
			return fmt.Errorf("SERVER_TRUSTED_PROXIES contains an empty value")
		}

		if strings.Contains(proxy, "/") {
			_, network, err := net.ParseCIDR(proxy)
			if err != nil {
				return fmt.Errorf("SERVER_TRUSTED_PROXIES contains invalid CIDR %q", proxy)
			}
			ones, _ := network.Mask.Size()
			if ones == 0 {
				return fmt.Errorf("SERVER_TRUSTED_PROXIES must not trust every address: %q", proxy)
			}
			continue
		}

		if net.ParseIP(proxy) == nil {
			return fmt.Errorf("SERVER_TRUSTED_PROXIES contains invalid IP %q", proxy)
		}
	}
	return nil
}

func validateRateLimitRule(prefix string, rule RateLimitRuleConfig, required bool) error {
	if rule.Max <= 0 {
		if !required && rule.Max == 0 {
			return nil
		}
		return fmt.Errorf("%s_MAX must be greater than 0 when authentication rate limit is enabled", prefix)
	}
	if rule.Window <= 0 {
		return fmt.Errorf("%s_WINDOW must be greater than 0 when authentication rate limit is enabled", prefix)
	}
	return nil
}

// IsProduction returns true if running in production
func IsProduction() bool {
	return GlobalConfig != nil && GlobalConfig.IsProduction()
}

// IsDevelopment returns true if running in development
func IsDevelopment() bool {
	return GlobalConfig == nil || GlobalConfig.App.Env == "development"
}

func isProductionEnvironment(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "production", "prod", "release":
		return true
	default:
		return false
	}
}

// LoadFresh forces reload of configuration
func LoadFresh() (*Config, error) {
	if err := env.LoadFresh(); err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}
	return Load()
}

// Use registers an already-constructed config as the process-global config.
// This is primarily used by tests and alternate bootstraps that still want to
// reuse the standard DI graph and runtime assembly.
func Use(cfg *Config) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	GlobalConfig = cfg
	return cfg, nil
}

package config

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"slices"
	"strconv"
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
	// DefaultDatabaseMaxIdleConns bounds idle PostgreSQL connections per process.
	DefaultDatabaseMaxIdleConns = 10
	// DefaultDatabaseMaxOpenConns bounds total PostgreSQL connections per process.
	DefaultDatabaseMaxOpenConns = 100
	// DefaultDatabaseConnMaxIdleTime retires unused connections before infrastructure changes make them stale.
	DefaultDatabaseConnMaxIdleTime = 15 * time.Minute
	// DefaultDatabaseConnMaxLifetime rotates every connection even when continuously reused.
	DefaultDatabaseConnMaxLifetime = time.Hour
	// DefaultDatabaseConnectTimeout bounds startup connection establishment and ping.
	DefaultDatabaseConnectTimeout = 5 * time.Second
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
	// DefaultAuthenticationSessionTTL is the absolute lifetime of one user session.
	DefaultAuthenticationSessionTTL = 30 * 24 * time.Hour
	// DefaultAuthenticationSessionIdleTimeout expires an inactive user session.
	DefaultAuthenticationSessionIdleTimeout = 7 * 24 * time.Hour
	// DefaultAuthenticationSessionTouchInterval bounds persistence writes for active sessions.
	DefaultAuthenticationSessionTouchInterval = 5 * time.Minute
	// DefaultAuthenticationSessionRetention keeps terminal rows briefly for operations and audit correlation.
	DefaultAuthenticationSessionRetention = 30 * 24 * time.Hour
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
	App            AppConfig
	Starters       StarterConfig
	Server         ServerConfig
	Database       DatabaseConfig
	Queue          QueueConfig
	Scheduler      SchedulerConfig
	Authentication AuthenticationConfig
	BrowserSession BrowserSessionConfig
	Log            LogConfig
	Sentry         SentryConfig
	CORS           CORSConfig
	Email          EmailConfig
	Organization   OrganizationConfig
	AI             AIConfig
	ObjectStorage  ObjectStorageConfig
	Asset          AssetConfig
	Webhook        WebhookConfig
	R2             R2Config
	Middleware     MiddlewareConfig
	Metrics        MetricsConfig
	Tracing        TracingConfig
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

// ValidateDatabase applies the database policy shared by configuration loading
// and alternate bootstraps that construct the database directly.
func (c *Config) ValidateDatabase() error {
	if c == nil {
		return fmt.Errorf("database configuration is required")
	}
	if err := c.Database.Validate(); err != nil {
		return err
	}
	if c.IsProduction() && c.Database.Enabled && c.Database.Driver == "postgres" {
		switch c.Database.SSLMode {
		case "require", "verify-ca", "verify-full":
		default:
			return fmt.Errorf("DB_SSLMODE must require TLS in production; use require, verify-ca, or verify-full")
		}
	}
	return nil
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

// DefaultRateLimitMaxBuckets bounds one process-local limiter store by default.
const DefaultRateLimitMaxBuckets = 10_000

// RateLimitConfig holds default HTTP rate limit configuration.
type RateLimitConfig struct {
	Enabled    bool
	Max        int
	Window     time.Duration
	MaxBuckets int
	SkipPaths  []string
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
	MaxBucketsPerRule    int
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
	ConnMaxIdleTime      time.Duration
	ConnMaxLifetime      time.Duration
	ConnectTimeout       time.Duration
	LogLevel             string
	SlowThreshold        time.Duration
	IgnoreRecordNotFound bool
}

// DBName returns the database name (alias for Name)
func (d DatabaseConfig) DBName() string {
	return d.Name
}

// Validate rejects ambiguous drivers, unbounded pools, inert lifetime policy,
// and incomplete connection settings before any database resource is created.
func (d DatabaseConfig) Validate() error {
	if !d.Enabled {
		return nil
	}
	if d.Driver != "postgres" {
		return fmt.Errorf("DB_DRIVER must be postgres")
	}
	if d.MaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be greater than 0 to keep the pool bounded")
	}
	if d.MaxIdleConns < 0 || d.MaxIdleConns > d.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be between 0 and DB_MAX_OPEN_CONNS")
	}
	if d.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_IDLE_TIME must be greater than 0")
	}
	if d.ConnMaxLifetime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME must be greater than 0")
	}
	if d.ConnMaxIdleTime > d.ConnMaxLifetime {
		return fmt.Errorf("DB_CONN_MAX_IDLE_TIME must not exceed DB_CONN_MAX_LIFETIME")
	}
	if d.ConnectTimeout <= 0 {
		return fmt.Errorf("DB_CONNECT_TIMEOUT must be greater than 0")
	}
	if d.SlowThreshold <= 0 {
		return fmt.Errorf("DB_SLOW_THRESHOLD must be greater than 0")
	}
	switch strings.ToLower(strings.TrimSpace(d.LogLevel)) {
	case "", "silent", "error", "warn", "warning", "info":
	default:
		return fmt.Errorf("DB_LOG_LEVEL must be one of silent, error, warn, or info")
	}

	if strings.TrimSpace(d.Host) == "" {
		return fmt.Errorf("DB_HOST is required for postgres")
	}
	if d.Port < 1 || d.Port > 65_535 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("DB_NAME is required for postgres")
	}
	if strings.TrimSpace(d.Username) == "" {
		return fmt.Errorf("DB_USERNAME is required for postgres")
	}
	if d.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required for postgres")
	}
	if strings.TrimSpace(d.Timezone) == "" {
		return fmt.Errorf("DB_TIMEZONE is required for postgres")
	}
	switch d.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("DB_SSLMODE must be a supported PostgreSQL SSL mode")
	}
	return nil
}

type QueueConfig struct {
	Driver            string
	DefaultQueue      string
	BufferSize        int
	WorkerConcurrency int
	WorkerSleep       time.Duration
	WorkerTimeout     time.Duration
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
}

type SchedulerConfig struct {
	Enabled bool
}

// AuthenticationConfig owns the server-side user session lifecycle.
type AuthenticationConfig struct {
	SessionTTL           time.Duration
	SessionIdleTimeout   time.Duration
	SessionTouchInterval time.Duration
	SessionRetention     time.Duration
}

// BrowserSessionConfig controls the optional same-origin HttpOnly session adapter.
type BrowserSessionConfig struct {
	Enabled bool
	Origin  string
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
	TransferSigningKey string
	MaxBytes           int64
	UploadGrantTTL     time.Duration
	DownloadGrantTTL   time.Duration
	PendingTTL         time.Duration
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
	if legacy := configuredLegacyJWTKey(); legacy != "" {
		return nil, fmt.Errorf("%s is no longer supported; Luas user authentication now uses opaque server-side sessions", legacy)
	}
	databaseConfig, err := loadDatabaseConfig()
	if err != nil {
		return nil, err
	}

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
		Database: databaseConfig,
		Queue: QueueConfig{
			Driver:            env.Get("QUEUE_DRIVER", "sync"),
			DefaultQueue:      env.Get("QUEUE_DEFAULT", "default"),
			BufferSize:        env.GetInt("QUEUE_BUFFER_SIZE", 256),
			WorkerConcurrency: env.GetInt("QUEUE_WORKER_CONCURRENCY", 1),
			WorkerSleep:       env.GetDuration("QUEUE_WORKER_SLEEP", time.Second),
			WorkerTimeout:     env.GetDuration("QUEUE_WORKER_TIMEOUT", 60*time.Second),
			LeaseDuration:     env.GetDuration("QUEUE_LEASE_DURATION", 90*time.Second),
			HeartbeatInterval: env.GetDuration("QUEUE_HEARTBEAT_INTERVAL", 20*time.Second),
		},
		Scheduler: SchedulerConfig{
			Enabled: env.GetBool("SCHEDULER_ENABLED", false),
		},
		Authentication: AuthenticationConfig{
			SessionTTL:           env.GetDuration("AUTH_SESSION_TTL", DefaultAuthenticationSessionTTL),
			SessionIdleTimeout:   env.GetDuration("AUTH_SESSION_IDLE_TIMEOUT", DefaultAuthenticationSessionIdleTimeout),
			SessionTouchInterval: env.GetDuration("AUTH_SESSION_TOUCH_INTERVAL", DefaultAuthenticationSessionTouchInterval),
			SessionRetention:     env.GetDuration("AUTH_SESSION_RETENTION", DefaultAuthenticationSessionRetention),
		},
		BrowserSession: BrowserSessionConfig{
			Enabled: env.GetBool("BROWSER_SESSION_ENABLED", false),
			Origin:  strings.TrimSpace(env.Get("BROWSER_SESSION_ORIGIN", "")),
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
			TransferSigningKey: env.Get("ASSET_TRANSFER_SIGNING_KEY", ""),
			MaxBytes:           int64(env.GetInt("ASSET_MAX_BYTES", int(DefaultAssetMaxBytes))),
			UploadGrantTTL:     env.GetDuration("ASSET_UPLOAD_GRANT_TTL", DefaultAssetUploadGrantTTL),
			DownloadGrantTTL:   env.GetDuration("ASSET_DOWNLOAD_GRANT_TTL", DefaultAssetDownloadGrantTTL),
			PendingTTL:         env.GetDuration("ASSET_PENDING_TTL", DefaultAssetPendingTTL),
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
				Enabled:    env.GetBool("MIDDLEWARE_RATE_LIMIT_ENABLED", isProd),
				Max:        env.GetInt("MIDDLEWARE_RATE_LIMIT_MAX", 600),
				Window:     env.GetDuration("MIDDLEWARE_RATE_LIMIT_WINDOW", time.Minute),
				MaxBuckets: env.GetInt("MIDDLEWARE_RATE_LIMIT_MAX_BUCKETS", DefaultRateLimitMaxBuckets),
				SkipPaths: env.GetSlice("MIDDLEWARE_RATE_LIMIT_SKIP_PATHS", []string{
					"/health",
					"/health/live",
					"/health/ready",
					"/metrics",
					"/v1/health",
				}),
			},
			AuthenticationRateLimit: AuthenticationRateLimitConfig{
				Enabled:           env.GetBool("AUTH_RATE_LIMIT_ENABLED", isProd),
				MaxBucketsPerRule: env.GetInt("AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE", DefaultRateLimitMaxBuckets),
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

func loadDatabaseConfig() (DatabaseConfig, error) {
	enabled, err := strictDatabaseBool("DB_ENABLED", true)
	if err != nil {
		return DatabaseConfig{}, err
	}
	port, err := strictDatabaseInt("DB_PORT", 5432)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxIdle, err := strictDatabaseInt("DB_MAX_IDLE_CONNS", DefaultDatabaseMaxIdleConns)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxOpen, err := strictDatabaseInt("DB_MAX_OPEN_CONNS", DefaultDatabaseMaxOpenConns)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxIdleTime, err := strictDatabaseDuration(
		"DB_CONN_MAX_IDLE_TIME",
		DefaultDatabaseConnMaxIdleTime,
	)
	if err != nil {
		return DatabaseConfig{}, err
	}
	maxLifetime, err := strictDatabaseDuration(
		"DB_CONN_MAX_LIFETIME",
		DefaultDatabaseConnMaxLifetime,
	)
	if err != nil {
		return DatabaseConfig{}, err
	}
	connectTimeout, err := strictDatabaseDuration(
		"DB_CONNECT_TIMEOUT",
		DefaultDatabaseConnectTimeout,
	)
	if err != nil {
		return DatabaseConfig{}, err
	}
	slowThreshold, err := strictDatabaseDuration("DB_SLOW_THRESHOLD", time.Second)
	if err != nil {
		return DatabaseConfig{}, err
	}
	ignoreNotFound, err := strictDatabaseBool("DB_LOG_IGNORE_NOT_FOUND", true)
	if err != nil {
		return DatabaseConfig{}, err
	}

	return DatabaseConfig{
		Enabled:              enabled,
		Driver:               strings.TrimSpace(env.Get("DB_DRIVER", "postgres")),
		Host:                 strings.TrimSpace(env.Get("DB_HOST", "localhost")),
		Port:                 port,
		Name:                 strings.TrimSpace(env.Get("DB_NAME", "")),
		Username:             strings.TrimSpace(env.Get("DB_USERNAME", "")),
		Password:             env.Get("DB_PASSWORD", ""),
		SSLMode:              strings.TrimSpace(env.Get("DB_SSLMODE", "disable")),
		Timezone:             strings.TrimSpace(env.Get("DB_TIMEZONE", "UTC")),
		MaxIdleConns:         maxIdle,
		MaxOpenConns:         maxOpen,
		ConnMaxIdleTime:      maxIdleTime,
		ConnMaxLifetime:      maxLifetime,
		ConnectTimeout:       connectTimeout,
		LogLevel:             strings.TrimSpace(env.Get("DB_LOG_LEVEL", "")),
		SlowThreshold:        slowThreshold,
		IgnoreRecordNotFound: ignoreNotFound,
	}, nil
}

func strictDatabaseInt(key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(env.Get(key, strconv.Itoa(defaultValue)))
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return value, nil
}

func strictDatabaseBool(key string, defaultValue bool) (bool, error) {
	raw := strings.ToLower(strings.TrimSpace(env.Get(key, strconv.FormatBool(defaultValue))))
	switch raw {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}

func strictDatabaseDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(env.Get(key, defaultValue.String()))
	if value, err := time.ParseDuration(raw); err == nil {
		return value, nil
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 15m or legacy integer seconds", key)
	}
	const (
		maxDurationSeconds = int64(1<<63-1) / int64(time.Second)
		minDurationSeconds = int64(-1<<63) / int64(time.Second)
	)
	if seconds > maxDurationSeconds || seconds < minDurationSeconds {
		return 0, fmt.Errorf("%s duration is out of range", key)
	}
	return time.Duration(seconds) * time.Second, nil
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

func configuredLegacyJWTKey() string {
	for _, key := range []string{"JWT_SECRET", "JWT_EXPIRE_DAYS"} {
		if strings.TrimSpace(env.Get(key, "")) != "" {
			return key
		}
	}
	return ""
}

// MustLoad loads configuration or panics
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func validate(cfg *Config) error {
	if err := cfg.ValidateDatabase(); err != nil {
		return err
	}

	isProd := cfg.IsProduction()
	if err := validateAIConfig(cfg.AI, isProd); err != nil {
		return err
	}
	if err := validateAuthenticationConfig(cfg.Authentication); err != nil {
		return err
	}
	if err := validateBrowserSessionConfig(cfg.BrowserSession, cfg.Database.Enabled, isProd); err != nil {
		return err
	}
	if err := validateQueueConfig(cfg.Queue, cfg.Database.Enabled); err != nil {
		return err
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
	if err := validateAssetConfig(
		cfg.Asset,
		assetSelected,
		assetSelected && strings.TrimSpace(cfg.ObjectStorage.Driver) == "local",
	); err != nil {
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
		if cfg.Middleware.RateLimit.MaxBuckets <= 0 {
			return fmt.Errorf("MIDDLEWARE_RATE_LIMIT_MAX_BUCKETS must be greater than 0 when rate limit is enabled")
		}
	}

	if err := validateServerTransport(cfg.Server, cfg.Middleware); err != nil {
		return err
	}

	if err := validateTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return err
	}

	if cfg.Middleware.AuthenticationRateLimit.Enabled {
		if cfg.Middleware.AuthenticationRateLimit.MaxBucketsPerRule <= 0 {
			return fmt.Errorf("AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE must be greater than 0 when authentication rate limit is enabled")
		}
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

func validateQueueConfig(queue QueueConfig, databaseEnabled bool) error {
	switch strings.ToLower(strings.TrimSpace(queue.Driver)) {
	case "sync", "memory":
	case "postgres":
		if !databaseEnabled {
			return fmt.Errorf("QUEUE_DRIVER=postgres requires DB_ENABLED=true")
		}
	default:
		return fmt.Errorf("QUEUE_DRIVER must be sync, memory, or postgres")
	}
	if strings.TrimSpace(queue.DefaultQueue) == "" || len(queue.DefaultQueue) > 128 {
		return fmt.Errorf("QUEUE_DEFAULT must contain 1 to 128 characters")
	}
	if queue.BufferSize < 1 || queue.WorkerConcurrency < 1 || queue.WorkerSleep <= 0 || queue.WorkerTimeout <= 0 {
		return fmt.Errorf("queue buffer, concurrency, sleep, and timeout values must be greater than 0")
	}
	if queue.LeaseDuration <= queue.WorkerTimeout {
		return fmt.Errorf("QUEUE_LEASE_DURATION must be greater than QUEUE_WORKER_TIMEOUT")
	}
	if queue.HeartbeatInterval <= 0 || queue.HeartbeatInterval >= queue.LeaseDuration {
		return fmt.Errorf("QUEUE_HEARTBEAT_INTERVAL must be greater than 0 and less than QUEUE_LEASE_DURATION")
	}
	return nil
}

func validateAuthenticationConfig(cfg AuthenticationConfig) error {
	const (
		minSessionTTL  = 15 * time.Minute
		maxSessionTTL  = 180 * 24 * time.Hour
		minIdleTimeout = 5 * time.Minute
		maxTouch       = time.Hour
		maxRetention   = 365 * 24 * time.Hour
	)

	if cfg.SessionTTL < minSessionTTL || cfg.SessionTTL > maxSessionTTL {
		return fmt.Errorf("AUTH_SESSION_TTL must be between %s and %s", minSessionTTL, maxSessionTTL)
	}
	if cfg.SessionIdleTimeout < minIdleTimeout || cfg.SessionIdleTimeout > cfg.SessionTTL {
		return fmt.Errorf("AUTH_SESSION_IDLE_TIMEOUT must be between %s and AUTH_SESSION_TTL", minIdleTimeout)
	}
	if cfg.SessionTouchInterval <= 0 || cfg.SessionTouchInterval > maxTouch || cfg.SessionTouchInterval > cfg.SessionIdleTimeout {
		return fmt.Errorf("AUTH_SESSION_TOUCH_INTERVAL must be greater than 0, no more than %s, and no longer than AUTH_SESSION_IDLE_TIMEOUT", maxTouch)
	}
	if cfg.SessionRetention < 0 || cfg.SessionRetention > maxRetention {
		return fmt.Errorf("AUTH_SESSION_RETENTION must be between 0 and %s", maxRetention)
	}
	return nil
}

func validateBrowserSessionConfig(cfg BrowserSessionConfig, databaseEnabled, production bool) error {
	if !cfg.Enabled {
		return nil
	}
	if !databaseEnabled {
		return fmt.Errorf("DB_ENABLED must be true when BROWSER_SESSION_ENABLED is true")
	}

	origin := strings.TrimSpace(cfg.Origin)
	parsed, err := url.Parse(origin)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("BROWSER_SESSION_ORIGIN must be an absolute origin without credentials, path, query, or fragment")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("BROWSER_SESSION_ORIGIN must use http or https")
	}
	if production && parsed.Scheme != "https" {
		return fmt.Errorf("BROWSER_SESSION_ORIGIN must use https in production")
	}
	if parsed.Scheme == "http" && !isLoopbackHostname(parsed.Hostname()) {
		return fmt.Errorf("BROWSER_SESSION_ORIGIN may use http only for a loopback host")
	}
	return nil
}

func isLoopbackHostname(hostname string) bool {
	if strings.EqualFold(strings.TrimSpace(hostname), "localhost") {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(hostname))
	return ip != nil && ip.IsLoopback()
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

func validateAssetConfig(asset AssetConfig, selected, localTransfers bool) error {
	if !selected && asset == (AssetConfig{}) {
		return nil
	}
	if localTransfers && len(strings.TrimSpace(asset.TransferSigningKey)) < 32 {
		return fmt.Errorf("ASSET_TRANSFER_SIGNING_KEY must be at least 32 characters for local asset transfers")
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

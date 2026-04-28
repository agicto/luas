package config

import (
	"fmt"
	"time"

	"github.com/zgiai/zgo/pkg/env"
)

// GlobalConfig stores the global configuration
var GlobalConfig *Config

// Config holds all application configuration
type Config struct {
	App        AppConfig
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Queue      QueueConfig
	Scheduler  SchedulerConfig
	JWT        JWTConfig
	Log        LogConfig
	CORS       CORSConfig
	Email      EmailConfig
	AI         AIConfig
	R2         R2Config
	Middleware MiddlewareConfig
	Tracing    TracingConfig
	ClickHouse ClickHouseConfig
}

type AppConfig struct {
	Name      string
	Env       string
	Debug     bool
	URL       string
	Key       string
	JWTSecret string
	JWTExpire time.Duration
}

type ServerConfig struct {
	Host           string
	Port           int
	Mode           string
	ReadTimeout    int
	WriteTimeout   int
	RequestTimeout int // Request timeout in seconds (for middleware)
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	RequestTimeout int   // Request timeout in seconds, default 180 (3 min)
	BodyLimit      int64 // Max body size in bytes, default 10MB
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
	Level string
	File  string
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
}

type EmailConfig struct {
	From         string
	ResendAPIKey string
}

type AIProviderConfig struct {
	APIKey  string
	BaseURL string
}

type AIConfig struct {
	Enabled         bool
	DefaultProvider string
	DefaultModel    string
	RequestTimeout  time.Duration
	OpenAI          AIProviderConfig
	Anthropic       AIProviderConfig
	Gemini          AIProviderConfig
}

type R2Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	Endpoint        string
	PublicURL       string
	PublicDomain    string
}

// TracingConfig holds OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled    bool
	Endpoint   string  // OTLP endpoint (e.g., "localhost:4317")
	Insecure   bool    // Use insecure connection
	SampleRate float64 // Sampling rate (0.0 to 1.0)
}

// ClickHouseConfig holds ClickHouse configuration
type ClickHouseConfig struct {
	Enabled   bool
	Endpoint  string
	Database  string
	Username  string
	Password  string
	BatchSize int
	Interval  time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	env.Load()

	expireDays := env.GetInt("JWT_EXPIRE_DAYS", 7)

	cfg := &Config{
		App: AppConfig{
			Name:      env.Get("APP_NAME", "ZGO"),
			Env:       env.Get("APP_ENV", "development"),
			Debug:     env.GetBool("APP_DEBUG", true),
			URL:       env.Get("APP_URL", "http://localhost:8025"),
			Key:       env.Get("APP_KEY", ""),
			JWTSecret: env.Get("JWT_SECRET", ""),
			JWTExpire: time.Duration(expireDays) * 24 * time.Hour,
		},
		Server: ServerConfig{
			Host:         env.Get("SERVER_HOST", ""),
			Port:         env.GetInt("SERVER_PORT", 8025),
			Mode:         env.Get("SERVER_MODE", env.Get("GIN_MODE", "debug")),
			ReadTimeout:  env.GetInt("SERVER_READ_TIMEOUT", 60),
			WriteTimeout: env.GetInt("SERVER_WRITE_TIMEOUT", 60),
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
			Timezone:             env.Get("DB_TIMEZONE", "Asia/Shanghai"),
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
			Level: env.Get("LOG_LEVEL", "debug"),
			File:  env.Get("LOG_FILE", env.Get("LOG_FILENAME", "storage/logs/app.log")),
		},
		CORS: CORSConfig{
			AllowOrigins:     env.GetSlice("CORS_ALLOW_ORIGINS", []string{"*"}),
			AllowMethods:     env.GetSlice("CORS_ALLOW_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowHeaders:     env.GetSlice("CORS_ALLOW_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}),
			ExposeHeaders:    env.GetSlice("CORS_EXPOSE_HEADERS", []string{"Content-Length", "X-Request-ID"}),
			AllowCredentials: env.GetBool("CORS_ALLOW_CREDENTIALS", true),
		},
		Email: EmailConfig{
			From:         env.Get("MAIL_FROM", ""),
			ResendAPIKey: env.Get("RESEND_API_KEY", ""),
		},
		AI: AIConfig{
			Enabled:         env.GetBool("AI_ENABLED", true),
			DefaultProvider: env.Get("AI_DEFAULT_PROVIDER", "openai"),
			DefaultModel:    env.Get("AI_DEFAULT_MODEL", "gpt-5.4"),
			RequestTimeout:  env.GetDuration("AI_REQUEST_TIMEOUT", 120*time.Second),
			OpenAI: AIProviderConfig{
				APIKey:  env.Get("OPENAI_API_KEY", ""),
				BaseURL: env.Get("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			},
			Anthropic: AIProviderConfig{
				APIKey:  env.Get("ANTHROPIC_API_KEY", ""),
				BaseURL: env.Get("ANTHROPIC_BASE_URL", ""),
			},
			Gemini: AIProviderConfig{
				APIKey:  env.Get("GEMINI_API_KEY", ""),
				BaseURL: env.Get("GEMINI_BASE_URL", ""),
			},
		},
		R2: R2Config{
			AccessKeyID:     env.Get("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: env.Get("R2_SECRET_ACCESS_KEY", ""),
			Bucket:          env.Get("R2_BUCKET", ""),
			Region:          env.Get("R2_REGION", "auto"),
			Endpoint:        env.Get("R2_ENDPOINT", ""),
			PublicURL:       env.Get("R2_PUBLIC_URL", ""),
			PublicDomain:    env.Get("R2_PUBLIC_DOMAIN", ""),
		},
		Middleware: MiddlewareConfig{
			RequestTimeout: env.GetInt("MIDDLEWARE_REQUEST_TIMEOUT", 180),                   // 3 minutes default
			BodyLimit:      int64(env.GetInt("MIDDLEWARE_BODY_LIMIT_MB", 10)) * 1024 * 1024, // 10MB default
		},
		Tracing: TracingConfig{
			Enabled:    env.GetBool("TRACING_ENABLED", false),
			Endpoint:   env.Get("TRACING_ENDPOINT", "localhost:4317"),
			Insecure:   env.GetBool("TRACING_INSECURE", true),
			SampleRate: env.GetFloat("TRACING_SAMPLE_RATE", 1.0),
		},
		ClickHouse: ClickHouseConfig{
			Enabled:   env.GetBool("LOG_CH_ENABLED", false),
			Endpoint:  env.Get("LOG_CH_ENDPOINT", "localhost:9000"),
			Database:  env.Get("LOG_CH_DATABASE", "zgo_logs"),
			Username:  env.Get("LOG_CH_USERNAME", "zgo_user"),
			Password:  env.Get("LOG_CH_PASSWORD", "zgo_pass"),
			BatchSize: env.GetInt("LOG_CH_BATCH_SIZE", 100),
			Interval:  env.GetDuration("LOG_CH_INTERVAL", 5*time.Second),
		},
	}

	// Validate required fields
	if err := validate(cfg); err != nil {
		return nil, err
	}

	GlobalConfig = cfg
	return cfg, nil
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
	if cfg.Database.Enabled && cfg.Database.Driver != "sqlite" && cfg.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required when database is enabled")
	}
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

// IsProduction returns true if running in production
func IsProduction() bool {
	return GlobalConfig != nil && GlobalConfig.App.Env == "production"
}

// IsDevelopment returns true if running in development
func IsDevelopment() bool {
	return GlobalConfig == nil || GlobalConfig.App.Env == "development"
}

// LoadFresh forces reload of configuration
func LoadFresh() (*Config, error) {
	env.LoadFresh()
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

// CacheConfig caches config (no-op for simplified version)
func CacheConfig(cfg *Config) error {
	return nil
}

// ClearCache clears config cache (no-op for simplified version)
func ClearCache() error {
	return nil
}

// CacheFilePath returns cache file path
func CacheFilePath() string {
	return "storage/framework/cache/config.json"
}

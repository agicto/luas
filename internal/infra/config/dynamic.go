package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zgiai/zgo/pkg/env"
)

// Repository is the main configuration storage and management tool
type Repository struct {
	items map[string]any
	mu    sync.RWMutex
}

// Global is the global configuration instance
var Global *Repository

// instance for singleton pattern
var (
	instance *Repository
	once     sync.Once
)

// New creates a new configuration repository
func New() *Repository {
	return &Repository{
		items: make(map[string]any),
	}
}

// Instance returns the singleton configuration instance
func Instance() *Repository {
	once.Do(func() {
		instance = New()
		Global = instance
	})
	return instance
}

// LoadDynamic loads configuration from environment variables
func LoadDynamic() *Repository {
	r := Instance()
	r.loadFromEnv()
	return r
}

// loadFromEnv loads all configuration from environment
func (r *Repository) loadFromEnv() {
	// Load .env files with priority
	env.Load()

	// App
	r.Set("app.name", env.Get("APP_NAME", "ZGO"))
	r.Set("app.env", env.Get("APP_ENV", "development"))
	r.Set("app.debug", env.GetBool("APP_DEBUG", true))
	r.Set("app.url", env.Get("APP_URL", "http://localhost:8025"))
	r.Set("app.key", env.Get("APP_KEY", ""))

	// Server
	r.Set("server.port", env.GetInt("SERVER_PORT", 8025))
	r.Set("server.host", env.Get("SERVER_HOST", ""))
	r.Set("server.mode", env.Get("SERVER_MODE", env.Get("GIN_MODE", "debug")))
	r.Set("server.read_timeout", env.GetInt("SERVER_READ_TIMEOUT", 60))
	r.Set("server.write_timeout", env.GetInt("SERVER_WRITE_TIMEOUT", 60))

	// Database
	r.Set("database.enabled", env.GetBool("DB_ENABLED", true))
	r.Set("database.driver", env.Get("DB_DRIVER", "postgres"))
	r.Set("database.host", env.Get("DB_HOST", "localhost"))
	r.Set("database.port", env.GetInt("DB_PORT", 5432))
	r.Set("database.name", env.Get("DB_NAME", ""))
	r.Set("database.username", env.Get("DB_USERNAME", ""))
	r.Set("database.password", env.Get("DB_PASSWORD", ""))
	r.Set("database.sslmode", env.Get("DB_SSLMODE", "disable"))
	r.Set("database.timezone", env.Get("DB_TIMEZONE", "Asia/Shanghai"))
	r.Set("database.max_idle_conns", env.GetInt("DB_MAX_IDLE_CONNS", 10))
	r.Set("database.max_open_conns", env.GetInt("DB_MAX_OPEN_CONNS", 100))

	// Redis
	r.Set("redis.host", env.Get("REDIS_HOST", "localhost"))
	r.Set("redis.port", env.GetInt("REDIS_PORT", 6379))
	r.Set("redis.password", env.Get("REDIS_PASSWORD", ""))
	r.Set("redis.db", env.GetInt("REDIS_DB", 0))

	// JWT
	r.Set("jwt.secret", env.Get("JWT_SECRET", ""))
	r.Set("jwt.expire_days", env.GetInt("JWT_EXPIRE_DAYS", 7))

	// Log
	r.Set("log.level", env.Get("LOG_LEVEL", "debug"))
	r.Set("log.file", env.Get("LOG_FILE", env.Get("LOG_FILENAME", "storage/logs/app.log")))

	// CORS
	r.Set("cors.allowed_origins", env.GetSlice("CORS_ALLOW_ORIGINS", env.GetSlice("CORS_ALLOWED_ORIGINS", []string{"*"})))
	r.Set("cors.allowed_methods", env.GetSlice("CORS_ALLOW_METHODS", env.GetSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})))
	r.Set("cors.allowed_headers", env.GetSlice("CORS_ALLOW_HEADERS", env.GetSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"})))
	r.Set("cors.expose_headers", env.GetSlice("CORS_EXPOSE_HEADERS", []string{"Content-Length"}))
	r.Set("cors.allow_credentials", env.GetBool("CORS_ALLOW_CREDENTIALS", true))

	// Email
	r.Set("mail.from", env.Get("MAIL_FROM", ""))
	r.Set("mail.resend_api_key", env.Get("RESEND_API_KEY", ""))

	// AI
	r.Set("ai.enabled", env.GetBool("AI_ENABLED", true))
	r.Set("ai.default_provider", env.Get("AI_DEFAULT_PROVIDER", "openai"))
	r.Set("ai.default_model", env.Get("AI_DEFAULT_MODEL", "gpt-5.4"))
	r.Set("ai.request_timeout", env.GetDuration("AI_REQUEST_TIMEOUT", 120*time.Second))
	r.Set("ai.openai.api_key", env.Get("OPENAI_API_KEY", ""))
	r.Set("ai.openai.base_url", env.Get("OPENAI_BASE_URL", "https://api.openai.com/v1"))
	r.Set("ai.anthropic.api_key", env.Get("ANTHROPIC_API_KEY", ""))
	r.Set("ai.anthropic.base_url", env.Get("ANTHROPIC_BASE_URL", ""))
	r.Set("ai.gemini.api_key", env.Get("GEMINI_API_KEY", ""))
	r.Set("ai.gemini.base_url", env.Get("GEMINI_BASE_URL", ""))

	// Backward-compatible alias
	r.Set("openai.api_key", env.Get("OPENAI_API_KEY", ""))
}

// Has determines if the given configuration value exists
func (r *Repository) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.get(key) != nil
}

// Get retrieves a configuration value using dot notation
func (r *Repository) Get(key string, defaultVal ...any) any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val := r.get(key)
	if val == nil && len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return val
}

// String returns a string configuration value
func (r *Repository) String(key string, defaultVal ...string) string {
	val := r.Get(key)
	if val == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// Int returns an integer configuration value
func (r *Repository) Int(key string, defaultVal ...int) int {
	val := r.Get(key)
	if val == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// Bool returns a boolean configuration value
func (r *Repository) Bool(key string, defaultVal ...bool) bool {
	val := r.Get(key)
	if val == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// Duration returns a duration configuration value
func (r *Repository) Duration(key string, defaultVal ...time.Duration) time.Duration {
	val := r.Get(key)
	if val == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	switch v := val.(type) {
	case time.Duration:
		return v
	case int:
		return time.Duration(v) * time.Second
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 0
}

// Slice returns a slice configuration value
func (r *Repository) Slice(key string, defaultVal ...[]string) []string {
	val := r.Get(key)
	if val == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return nil
	}
	if s, ok := val.([]string); ok {
		return s
	}
	return nil
}

// Set sets a configuration value using dot notation
func (r *Repository) Set(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.set(key, value)
}

// All returns all configuration items
func (r *Repository) All() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.items
}

// get retrieves a value using dot notation (internal, no lock)
func (r *Repository) get(key string) any {
	parts := strings.Split(key, ".")
	current := any(r.items)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// set sets a value using dot notation (internal, no lock)
func (r *Repository) set(key string, value any) {
	parts := strings.Split(key, ".")
	current := r.items

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]any)
		}
		current = current[part].(map[string]any)
	}

	current[parts[len(parts)-1]] = value
}

// Cache caches the configuration to a JSON file
func (r *Repository) Cache(path ...string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cachePath := "storage/framework/cache/config.json"
	if len(path) > 0 {
		cachePath = path[0]
	}

	// Ensure directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// ClearDynamicCache removes the cached configuration
func ClearDynamicCache(path ...string) error {
	cachePath := "storage/framework/cache/config.json"
	if len(path) > 0 {
		cachePath = path[0]
	}
	return os.Remove(cachePath)
}

// ============================================
// Global helper functions for configuration access
// ============================================

// ConfigDynamic returns a configuration value (global helper)
func ConfigDynamic(key string, defaultVal ...any) any {
	if Global == nil {
		LoadDynamic()
	}
	return Global.Get(key, defaultVal...)
}

// ConfigString returns a string configuration value
func ConfigString(key string, defaultVal ...string) string {
	if Global == nil {
		Load()
	}
	return Global.String(key, defaultVal...)
}

// ConfigInt returns an integer configuration value
func ConfigInt(key string, defaultVal ...int) int {
	if Global == nil {
		Load()
	}
	return Global.Int(key, defaultVal...)
}

// ConfigBool returns a boolean configuration value
func ConfigBool(key string, defaultVal ...bool) bool {
	if Global == nil {
		Load()
	}
	return Global.Bool(key, defaultVal...)
}

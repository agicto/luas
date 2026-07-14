package env

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Priority order (highest to lowest):
// 1. System environment variables (Docker, K8s, shell export)
// 2. LUAS_ENV_FILE (explicit file selected by a system environment variable)
// 3. .env.{APP_ENV}.local (environment-specific local overrides, not committed)
// 4. .env.local (local overrides, not committed)
// 5. .env.{APP_ENV} (environment-specific, e.g., .env.production)
// 6. .env (base configuration)
// 7. Default values in code

var (
	loadMu           sync.Mutex
	loaded           bool
	loadErr          error
	appEnv           string
	loadedFileValues = make(map[string]string)
)

// Load loads environment files respecting priority.
// System environment variables always have highest priority.
// This function is idempotent - calling it multiple times has no effect.
func Load() error {
	loadMu.Lock()
	defer loadMu.Unlock()
	if loaded {
		return loadErr
	}

	loadErr = loadEnvFiles()
	loaded = true
	return loadErr
}

// LoadFresh forces reload of environment files.
// It is intended for tests and diagnostics, not runtime hot reload: services
// already constructed from configuration are not rebuilt by this function.
func LoadFresh() error {
	loadMu.Lock()
	defer loadMu.Unlock()

	removeLoadedFileValues()
	loadErr = loadEnvFiles()
	loaded = true
	return loadErr
}

func loadEnvFiles() error {
	// Capture process values before applying files. Presence, including an empty
	// value, is authoritative over every file layer.
	systemEnv := captureSystemEnv()
	appEnv = firstNonEmpty(systemEnv["APP_ENV"], systemEnv["GO_ENV"], systemEnv["GIN_MODE"])

	baseValues, err := readEnvFile(".env", false)
	if err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	explicitValues := map[string]string(nil)
	if envFile := systemEnv["LUAS_ENV_FILE"]; envFile != "" {
		explicitValues, err = readEnvFile(envFile, true)
		if err != nil {
			return fmt.Errorf("load LUAS_ENV_FILE %q: %w", envFile, err)
		}
	}

	// Environment selection cannot depend on an environment-specific file.
	// This avoids circular file selection while still allowing the base or an
	// explicitly selected file to choose the environment.
	selectedEnv := firstNonEmpty(
		systemEnv["APP_ENV"],
		systemEnv["GO_ENV"],
		systemEnv["GIN_MODE"],
		explicitValues["APP_ENV"],
		explicitValues["GO_ENV"],
		explicitValues["GIN_MODE"],
		baseValues["APP_ENV"],
		baseValues["GO_ENV"],
		baseValues["GIN_MODE"],
	)

	// Merge file layers from lowest to highest priority. Map assignment is
	// explicit because godotenv.Load intentionally never overwrites a value,
	// which would reverse the documented overlay order.
	merged := make(map[string]string)
	mergeEnv(merged, baseValues)
	if selectedEnv != "" {
		environmentValues, readErr := readEnvFile(".env."+selectedEnv, false)
		if readErr != nil {
			return fmt.Errorf("load .env.%s: %w", selectedEnv, readErr)
		}
		mergeEnv(merged, environmentValues)
	}
	localValues, err := readEnvFile(".env.local", false)
	if err != nil {
		return fmt.Errorf("load .env.local: %w", err)
	}
	mergeEnv(merged, localValues)
	if selectedEnv != "" {
		localEnvironmentValues, readErr := readEnvFile(".env."+selectedEnv+".local", false)
		if readErr != nil {
			return fmt.Errorf("load .env.%s.local: %w", selectedEnv, readErr)
		}
		mergeEnv(merged, localEnvironmentValues)
	}
	mergeEnv(merged, explicitValues)

	// APP_ENV is the canonical output. GO_ENV and GIN_MODE are accepted as
	// process/base inputs, but local overlays cannot change the environment
	// identity after file selection has completed.
	delete(merged, "GO_ENV")
	delete(merged, "GIN_MODE")
	if selectedEnv == "" {
		delete(merged, "APP_ENV")
	} else {
		merged["APP_ENV"] = selectedEnv
	}

	appliedValues := make(map[string]string, len(merged))
	for key, value := range merged {
		if _, exists := systemEnv[key]; exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			for appliedKey := range appliedValues {
				_ = os.Unsetenv(appliedKey)
			}
			return fmt.Errorf("apply environment key %q: %w", key, err)
		}
		appliedValues[key] = value
	}
	loadedFileValues = appliedValues

	appEnv = selectedEnv
	return nil
}

func readEnvFile(path string, required bool) (map[string]string, error) {
	values, err := godotenv.Read(path)
	if err != nil {
		if !required && errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return values, nil
}

func mergeEnv(target map[string]string, source map[string]string) {
	for key, value := range source {
		target[key] = value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if normalized := strings.TrimSpace(value); normalized != "" {
			return normalized
		}
	}
	return ""
}

func removeLoadedFileValues() {
	for key, loadedValue := range loadedFileValues {
		currentValue, exists := os.LookupEnv(key)
		if exists && currentValue == loadedValue {
			_ = os.Unsetenv(key)
		}
	}
	loadedFileValues = make(map[string]string)
}

func loadForGetter() {
	// Typed configuration loading returns file errors to the caller. Legacy
	// scalar getters cannot change signature, so they retain process/default
	// lookup behavior while config.Load remains the fail-fast runtime path.
	if err := Load(); err != nil {
		return
	}
}

// captureSystemEnv captures current environment variables before .env loading
func captureSystemEnv() map[string]string {
	result := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// Get returns the value of an environment variable.
// If the variable is not set, returns the default value.
func Get(key string, defaultValue ...string) string {
	loadForGetter()
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetOrFail returns the value of an environment variable.
// Panics if the variable is not set.
func GetOrFail(key string) string {
	loadForGetter()
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	panic("Environment variable " + key + " is required but not set")
}

// GetBool returns the boolean value of an environment variable.
// Recognizes: true, false, 1, 0, yes, no, on, off (case-insensitive)
func GetBool(key string, defaultValue ...bool) bool {
	loadForGetter()
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off", "":
		return false
	}

	// Try parsing as bool
	b, err := strconv.ParseBool(value)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return b
}

// GetInt returns the integer value of an environment variable.
func GetInt(key string, defaultValue ...int) int {
	loadForGetter()
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	i, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return i
}

// GetInt64 returns the int64 value of an environment variable.
func GetInt64(key string, defaultValue ...int64) int64 {
	loadForGetter()
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	i, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return i
}

// GetFloat returns the float64 value of an environment variable.
func GetFloat(key string, defaultValue ...float64) float64 {
	loadForGetter()
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return f
}

// GetDuration returns the time.Duration value of an environment variable.
func GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	loadForGetter()
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	d, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return d
}

// GetSlice returns a slice from a comma-separated environment variable.
func GetSlice(key string, defaultValue ...[]string) []string {
	loadForGetter()
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Set sets an environment variable.
// This is useful for testing.
func Set(key, value string) {
	os.Setenv(key, value)
}

// Unset removes an environment variable.
func Unset(key string) {
	os.Unsetenv(key)
}

// AppEnv returns the current application environment.
// Returns "development" if not set.
func AppEnv() string {
	loadForGetter()
	loadMu.Lock()
	current := appEnv
	loadMu.Unlock()
	if current != "" {
		return current
	}
	return "development"
}

// IsProduction returns true if running in production mode.
func IsProduction() bool {
	env := strings.ToLower(AppEnv())
	return env == "production" || env == "prod" || env == "release"
}

// IsDevelopment returns true if running in development mode.
func IsDevelopment() bool {
	env := strings.ToLower(AppEnv())
	return env == "development" || env == "dev" || env == "local" || env == "debug"
}

// IsTesting returns true if running in test mode.
func IsTesting() bool {
	env := strings.ToLower(AppEnv())
	return env == "testing" || env == "test"
}

// IsLocal returns true if running locally (development or testing).
func IsLocal() bool {
	return IsDevelopment() || IsTesting()
}

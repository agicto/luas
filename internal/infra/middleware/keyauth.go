package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/pkg/response"
)

// KeyAuthResult carries validation output back into the request context.
type KeyAuthResult struct {
	Key    string
	Values map[string]any
}

// KeyAuthConfig holds API key authentication configuration
type KeyAuthConfig struct {
	// KeyLookup is where to look for the key: "header:X-API-Key", "query:api_key", "cookie:api_key"
	// Default: "header:X-API-Key"
	KeyLookup string

	// Validator is a function to validate the API key
	// Returns true if key is valid
	Validator func(key string) bool

	// ValidatorWithContext validates a key with access to the request context
	// and can populate extra values into gin.Context on success.
	ValidatorWithContext func(c *gin.Context, key string) (*KeyAuthResult, error)

	// ContextKey is the key used to store the API key in context
	// Default: "api_key"
	ContextKey string

	// ErrorMessage is the message returned when authentication fails
	// Default: "Invalid or missing API key"
	ErrorMessage string

	// AuthScheme is the scheme expected before the key (e.g., "Bearer")
	// Default: "" (no scheme)
	AuthScheme string
}

// DefaultKeyAuthConfig returns default key auth configuration
func DefaultKeyAuthConfig() KeyAuthConfig {
	return KeyAuthConfig{
		KeyLookup:    "header:X-API-Key",
		ContextKey:   "api_key",
		ErrorMessage: "Invalid or missing API key",
		AuthScheme:   "",
	}
}

// KeyAuth returns API key authentication middleware
func KeyAuth(validator func(key string) bool) gin.HandlerFunc {
	cfg := DefaultKeyAuthConfig()
	cfg.Validator = validator
	return KeyAuthWithConfig(cfg)
}

// KeyAuthWithConfig returns API key authentication middleware with custom config
func KeyAuthWithConfig(cfg KeyAuthConfig) gin.HandlerFunc {
	if cfg.KeyLookup == "" {
		cfg.KeyLookup = "header:X-API-Key"
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = "api_key"
	}
	if cfg.ErrorMessage == "" {
		cfg.ErrorMessage = "Invalid or missing API key"
	}
	if cfg.Validator == nil && cfg.ValidatorWithContext == nil {
		panic("KeyAuth middleware requires a validator function")
	}

	// Parse key lookup
	parts := strings.SplitN(cfg.KeyLookup, ":", 2)
	if len(parts) != 2 {
		panic("KeyAuth KeyLookup must be in format 'source:name'")
	}
	source := strings.ToLower(parts[0])
	name := parts[1]

	return func(c *gin.Context) {
		var key string

		// Extract key based on source
		switch source {
		case "header":
			key = c.GetHeader(name)
			// Handle auth scheme if present
			if cfg.AuthScheme != "" && strings.HasPrefix(key, cfg.AuthScheme+" ") {
				key = strings.TrimPrefix(key, cfg.AuthScheme+" ")
			}
		case "query":
			key = c.Query(name)
		case "cookie":
			key, _ = c.Cookie(name)
		case "form":
			key = c.PostForm(name)
		default:
			response.Abort(c, http.StatusInternalServerError, "Invalid key auth configuration")
			return
		}

		if key == "" {
			response.Abort(c, http.StatusUnauthorized, cfg.ErrorMessage)
			return
		}

		var (
			result *KeyAuthResult
			err    error
			valid  bool
		)

		if cfg.ValidatorWithContext != nil {
			result, err = cfg.ValidatorWithContext(c, key)
			valid = err == nil && result != nil
		} else {
			valid = cfg.Validator(key)
		}

		if !valid {
			response.Abort(c, http.StatusUnauthorized, cfg.ErrorMessage)
			return
		}

		if result != nil {
			if result.Key != "" {
				key = result.Key
			}
			for contextKey, value := range result.Values {
				c.Set(contextKey, value)
			}
		}

		// Store key in context
		c.Set(cfg.ContextKey, key)

		c.Next()
	}
}

// GetAPIKey retrieves the API key from context
func GetAPIKey(c *gin.Context) string {
	if key, exists := c.Get("api_key"); exists {
		return key.(string)
	}
	return ""
}

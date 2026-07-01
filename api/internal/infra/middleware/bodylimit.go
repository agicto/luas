package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/response"
)

// BodyLimitConfig holds body limit middleware configuration
type BodyLimitConfig struct {
	// MaxSize is the maximum allowed request body size in bytes
	// Default: 10MB
	MaxSize int64

	// ErrorMessage is the message returned when limit is exceeded
	// Default: "Request body too large"
	ErrorMessage string
}

// DefaultBodyLimitConfig returns default body limit configuration
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		MaxSize:      10 * 1024 * 1024, // 10MB
		ErrorMessage: "Request body too large",
	}
}

// BodyLimitFromConfig returns body limit middleware using global config
// Uses MIDDLEWARE_BODY_LIMIT_MB env var (in megabytes)
func BodyLimitFromConfig() gin.HandlerFunc {
	maxSize := int64(10 * 1024 * 1024) // default 10MB
	if config.GlobalConfig != nil && config.GlobalConfig.Middleware.BodyLimit > 0 {
		maxSize = config.GlobalConfig.Middleware.BodyLimit
	}
	return BodyLimit(maxSize)
}

// BodyLimit returns body limit middleware with size in bytes
func BodyLimit(maxSize int64) gin.HandlerFunc {
	return BodyLimitWithConfig(BodyLimitConfig{
		MaxSize:      maxSize,
		ErrorMessage: "Request body too large",
	})
}

// BodyLimitWithConfig returns body limit middleware with custom config
func BodyLimitWithConfig(cfg BodyLimitConfig) gin.HandlerFunc {
	defaults := DefaultBodyLimitConfig()
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = defaults.MaxSize
	}
	if cfg.ErrorMessage == "" {
		cfg.ErrorMessage = defaults.ErrorMessage
	}

	return func(c *gin.Context) {
		// Check Content-Length header first (fast path)
		if c.Request.ContentLength > cfg.MaxSize {
			response.AbortWithCode(c, http.StatusRequestEntityTooLarge, response.ErrorCodeRequestTooLarge, cfg.ErrorMessage)
			return
		}

		// Wrap the body with a size limiter
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxSize)

		c.Next()

		// Check if we hit the limit during body read
		var maxBytesErr *http.MaxBytesError
		if last := c.Errors.Last(); last != nil && errors.As(last.Err, &maxBytesErr) {
			response.AbortWithCode(c, http.StatusRequestEntityTooLarge, response.ErrorCodeRequestTooLarge, cfg.ErrorMessage)
		}
	}
}

package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/response"
)

// TimeoutConfig holds timeout middleware configuration
type TimeoutConfig struct {
	// Timeout is the maximum duration for request processing
	// Default: 3 minutes (for AI/LLM calls)
	Timeout time.Duration

	// ErrorMessage is the message returned when timeout occurs
	// Default: "Request timeout"
	ErrorMessage string

	// ErrorHandler is a custom handler for timeout errors
	// If nil, returns 503 Service Unavailable with ErrorMessage
	ErrorHandler func(c *gin.Context)
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:      3 * time.Minute, // 3 minutes for AI/LLM calls
		ErrorMessage: "Request timeout",
	}
}

// TimeoutFromConfig returns timeout middleware using global config
// Uses MIDDLEWARE_REQUEST_TIMEOUT env var (in seconds)
func TimeoutFromConfig() gin.HandlerFunc {
	timeout := 3 * time.Minute // default 3 minutes
	if config.GlobalConfig != nil && config.GlobalConfig.Middleware.RequestTimeout > 0 {
		timeout = time.Duration(config.GlobalConfig.Middleware.RequestTimeout) * time.Second
	}
	return Timeout(timeout)
}

// Timeout returns timeout middleware with specified duration
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return TimeoutWithConfig(TimeoutConfig{
		Timeout:      timeout,
		ErrorMessage: "Request timeout",
	})
}

// TimeoutWithConfig returns cooperative timeout middleware with custom config.
//
// The middleware sets a deadline on the request context and runs the remaining
// Gin chain synchronously. Handlers and downstream calls should observe
// c.Request.Context().Done(). If the chain returns after the deadline without
// writing a response, the middleware emits the standard timeout error.
func TimeoutWithConfig(cfg TimeoutConfig) gin.HandlerFunc {
	defaults := DefaultTimeoutConfig()
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaults.Timeout
	}
	if cfg.ErrorMessage == "" {
		cfg.ErrorMessage = defaults.ErrorMessage
	}

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		if ctx.Err() != context.DeadlineExceeded || c.Writer.Written() {
			return
		}

		if cfg.ErrorHandler != nil {
			cfg.ErrorHandler(c)
			return
		}

		response.AbortWithCode(c, http.StatusServiceUnavailable, response.ErrorCodeTimeout, cfg.ErrorMessage)
	}
}

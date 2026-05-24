package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/gin-gonic/gin"
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

// TimeoutWithConfig returns timeout middleware with custom config
func TimeoutWithConfig(cfg TimeoutConfig) gin.HandlerFunc {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.ErrorMessage == "" {
		cfg.ErrorMessage = "Request timeout"
	}

	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to signal completion
		done := make(chan struct{})

		// Run handler in goroutine. If the deadline fires before the
		// handler returns, the timeout response is sent below but this
		// goroutine keeps running until the handler finishes — recover
		// any write-after-abort panics from gin so the server stays up.
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("timeout middleware: handler panic after timeout (request likely already aborted): %v", r)
				}
				close(done)
			}()
			c.Next()
		}()

		// Wait for completion or timeout
		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				c.Abort()
				if cfg.ErrorHandler != nil {
					cfg.ErrorHandler(c)
				} else {
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"success": false,
						"error": gin.H{
							"code":    "TIMEOUT",
							"message": cfg.ErrorMessage,
						},
					})
				}
			}
		}
	}
}

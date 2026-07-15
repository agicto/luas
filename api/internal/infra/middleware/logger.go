package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/pkg/logger"
)

// Logger is the request logging middleware
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Request processed
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		logger.Info("HTTP Request", map[string]any{
			"status":    statusCode,
			"latency":   latency.String(),
			"client_ip": clientIP,
			"method":    method,
			"path":      path,
		})
	}
}

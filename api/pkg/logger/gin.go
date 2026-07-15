package logger

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLogger returns a gin.HandlerFunc that logs requests using the platform logger
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Fill the log record
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		// Use the platform logger
		fields := map[string]any{
			"status":     statusCode,
			"latency":    latency.String(),
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"request_id": c.GetString("request_id"),
		}

		if len(c.Errors) > 0 {
			errorTypes := make([]string, 0, len(c.Errors))
			for _, requestError := range c.Errors {
				errorTypes = append(errorTypes, fmt.Sprintf("%T", requestError.Err))
			}
			fields["error_count"] = len(c.Errors)
			fields["error_types"] = errorTypes
			Error("HTTP Request Error", fields)
		} else {
			Info("HTTP Request", fields)
		}
	}
}

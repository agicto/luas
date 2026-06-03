package middleware

import (
	"github.com/gin-gonic/gin"
)

// VersionMiddleware handles API versioning
func VersionMiddleware(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_version", version)
		c.Next()
	}
}

// GetVersion gets the current API version
func GetVersion(c *gin.Context) string {
	if v, exists := c.Get("api_version"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "v1" // Default version
}

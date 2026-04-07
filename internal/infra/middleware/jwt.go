package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/infra/jwt"
	"github.com/zgiai/zgo/pkg/response"
)

// JWTAuth creates a JWT authentication middleware with an explicit service dependency.
func JWTAuth(svc *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.Error(c, http.StatusInternalServerError, "JWT service not initialized")
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Error(c, http.StatusUnauthorized, "Invalid authorization format")
			c.Abort()
			return
		}

		claims, err := svc.ParseToken(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// JWTAuthWithService preserves the old explicit-injection entrypoint.
func JWTAuthWithService(svc *jwt.Service) gin.HandlerFunc {
	return JWTAuth(svc)
}

package apikey

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

const apiKeyScopesContextKey = "apiKeyScopes"

// RequireScopes rejects authenticated API keys that do not grant every scope.
// Apply it after the starter-owned "api_key" middleware.
func RequireScopes(requiredScopes ...string) gin.HandlerFunc {
	normalized, err := normalizeAPIKeyScopes(requiredScopes)
	if err != nil || len(normalized) == 0 {
		panic("apikey: RequireScopes needs at least one valid scope")
	}

	return func(c *gin.Context) {
		value, exists := c.Get(apiKeyScopesContextKey)
		scopes, ok := value.([]string)
		if !exists || !ok {
			response.AbortUnauthorized(c, "API key authentication required")
			return
		}

		key := &domain.APIKey{Scopes: scopes}
		for _, required := range normalized {
			if !key.HasScope(required) {
				response.AbortWithCode(
					c,
					http.StatusForbidden,
					domain.CodePermissionDenied,
					"API key scope denied",
				)
				return
			}
		}

		c.Next()
	}
}

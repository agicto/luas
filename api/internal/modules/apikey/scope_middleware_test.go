package apikey

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestRequireScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		scopes    []string
		setScopes bool
		wantCode  int
		wantError string
	}{
		{
			name:      "all required scopes",
			scopes:    []string{"models:read", "models:invoke"},
			setScopes: true,
			wantCode:  http.StatusNoContent,
		},
		{
			name:      "wildcard",
			scopes:    []string{"*"},
			setScopes: true,
			wantCode:  http.StatusNoContent,
		},
		{
			name:      "scope denied",
			scopes:    []string{"models:read"},
			setScopes: true,
			wantCode:  http.StatusForbidden,
			wantError: domain.CodePermissionDenied,
		},
		{
			name:      "authentication missing",
			wantCode:  http.StatusUnauthorized,
			wantError: "AUTH.UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/resource", func(c *gin.Context) {
				if tt.setScopes {
					c.Set(apiKeyScopesContextKey, tt.scopes)
				}
				c.Next()
			}, RequireScopes("models:read", "models:invoke"), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			responseRecorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/resource", nil)
			engine.ServeHTTP(responseRecorder, request)

			assert.Equal(t, tt.wantCode, responseRecorder.Code)
			if tt.wantError != "" {
				var body struct {
					ErrorCode string `json:"error_code"`
				}
				require.NoError(t, json.Unmarshal(responseRecorder.Body.Bytes(), &body))
				assert.Equal(t, tt.wantError, body.ErrorCode)
			}
		})
	}
}

func TestRequireScopesRejectsInvalidConfiguration(t *testing.T) {
	assert.Panics(t, func() { RequireScopes() })
	assert.Panics(t, func() { RequireScopes("models") })
}

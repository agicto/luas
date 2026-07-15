package apikey

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/pkg/response"
)

type apiKeyValidationErrorService struct {
	Service
	err error
}

func (s *apiKeyValidationErrorService) Validate(
	context.Context,
	string,
	...string,
) (*domain.APIKey, error) {
	return nil, s.err
}

func TestAPIKeyMiddlewarePreservesDomainErrorCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		err       error
		errorCode string
	}{
		{name: "invalid", err: domain.ErrAPIKeyInvalid, errorCode: domain.CodeAPIKeyInvalid},
		{name: "expired", err: domain.ErrAPIKeyExpired, errorCode: domain.CodeAPIKeyExpired},
		{name: "revoked", err: domain.ErrAPIKeyRevoked, errorCode: domain.CodeAPIKeyRevoked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response.DefaultErrorMapper.Register(tt.err, http.StatusUnauthorized, tt.errorCode)
			engine := apiKeyMiddlewareTestEngine(&apiKeyValidationErrorService{err: tt.err})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("X-API-Key", "luas_test.secret")
			result := httptest.NewRecorder()

			engine.ServeHTTP(result, request)

			assert.Equal(t, http.StatusUnauthorized, result.Code)
			var body struct {
				ErrorCode string `json:"error_code"`
			}
			require.NoError(t, json.Unmarshal(result.Body.Bytes(), &body))
			assert.Equal(t, tt.errorCode, body.ErrorCode)
		})
	}
}

func TestAPIKeyMiddlewareUsesAuthUnauthorizedWhenHeaderIsMissing(t *testing.T) {
	engine := apiKeyMiddlewareTestEngine(&apiKeyValidationErrorService{})
	result := httptest.NewRecorder()

	engine.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/protected", nil))

	assert.Equal(t, http.StatusUnauthorized, result.Code)
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &body))
	assert.Equal(t, response.ErrorCodeUnauthorized, body.ErrorCode)
}

func apiKeyMiddlewareTestEngine(service Service) *gin.Engine {
	engine := gin.New()
	routes := router.New(engine)
	handler := NewHandler(service)
	handler.RegisterMiddleware(routes)
	routes.WithMiddleware("api_key").GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return engine
}

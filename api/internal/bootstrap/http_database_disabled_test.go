package bootstrap_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/jwt"
	"github.com/zgiai/luas/api/internal/wiring"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestHTTPKernelDatabaseDisabledDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		App: config.AppConfig{
			Name:  "Luas",
			Env:   "test",
			Debug: true,
		},
		Server: config.ServerConfig{
			Mode: gin.TestMode,
		},
		Database: config.DatabaseConfig{
			Enabled: false,
		},
		JWT: config.JWTConfig{
			Secret: "database-disabled-test-secret",
			Expire: time.Hour,
		},
		CORS: config.CORSConfig{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
		},
		Middleware: config.MiddlewareConfig{
			RequestTimeout: 1,
			BodyLimit:      1024,
		},
	}

	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		t.Fatalf("initialize application: %v", err)
	}
	kernel := bootstrap.NewHttpKernel(application)
	token, err := jwt.NewService(cfg).GenerateToken(7, "disabled-db-user")
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		authorized bool
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unauthorized mutation keeps its primary response",
			method:     http.MethodDelete,
			path:       "/v1/api-keys/123",
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.ErrorCodeUnauthorized,
		},
		{
			name:       "registration reports unavailable persistence",
			method:     http.MethodPost,
			path:       "/v1/register",
			body:       `{"username":"disabled-db","password":"secret12","email":"disabled-db@example.com"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.ErrorCodeServiceUnavailable,
		},
		{
			name:       "profile reports unavailable persistence",
			method:     http.MethodGet,
			path:       "/v1/users/profile",
			authorized: true,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.ErrorCodeServiceUnavailable,
		},
		{
			name:       "API keys report unavailable persistence",
			method:     http.MethodGet,
			path:       "/v1/api-keys",
			authorized: true,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.ErrorCodeServiceUnavailable,
		},
		{
			name:       "audit history reports unavailable persistence",
			method:     http.MethodGet,
			path:       "/v1/audit-logs",
			authorized: true,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   response.ErrorCodeServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			if test.authorized {
				request.Header.Set("Authorization", "Bearer "+token)
			}
			recorder := httptest.NewRecorder()

			kernel.Engine.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var payload response.ErrorResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("response is not one valid JSON envelope: %v; body = %s", err, recorder.Body.String())
			}
			if payload.ErrorCode != test.wantCode {
				t.Fatalf("error_code = %q, want %q", payload.ErrorCode, test.wantCode)
			}
			if payload.RequestID == "" {
				t.Fatal("request_id is empty")
			}
		})
	}
}

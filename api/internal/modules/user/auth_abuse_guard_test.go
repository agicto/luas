package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	infraMiddleware "github.com/zgiai/luas/api/internal/infra/middleware"
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/pkg/response"
)

type authGuardFakeService struct {
	loginCalls         int
	passwordResetCalls int
}

func (s *authGuardFakeService) Register(context.Context, *UserRegisterRequest) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "new-user", Status: 1}, nil
}

func (s *authGuardFakeService) Login(context.Context, *UserLoginRequest) (*UserLoginResponse, error) {
	s.loginCalls++
	return &UserLoginResponse{
		AccessToken: "test-token",
		User:        &domain.User{ID: 1, Username: "test-user", Status: 1},
	}, nil
}

func (s *authGuardFakeService) RequestPasswordReset(context.Context, *UserPasswordResetRequest) error {
	s.passwordResetCalls++
	return nil
}

func (s *authGuardFakeService) ConfirmPasswordReset(context.Context, *UserPasswordResetConfirmRequest) error {
	return nil
}

func authProtectionTestEngine(t *testing.T, cfg config.AuthenticationRateLimitConfig, service *authGuardFakeService) *gin.Engine {
	t.Helper()

	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies(nil) error = %v", err)
	}
	engine.Use(infraMiddleware.RequestIDWithConfig(infraMiddleware.RequestIDConfig{
		Generator: func() string { return "req_auth_limit" },
	}))

	handler := NewHandler(service, nil, nil, nil, nil, newAuthAbuseGuard(cfg), nil)
	routes := router.New(engine).Prefix("/v1")
	handler.RegisterMiddleware(routes)
	handler.RegisterRoutes(routes)
	return engine
}

func performAuthRequest(engine *gin.Engine, path, body, remoteAddr string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = remoteAddr
	result := httptest.NewRecorder()
	engine.ServeHTTP(result, request)
	return result
}

func TestAuthenticationRateLimitBlocksOneSourceAcrossLoginSubjects(t *testing.T) {
	service := &authGuardFakeService{}
	engine := authProtectionTestEngine(t, config.AuthenticationRateLimitConfig{
		Enabled: true,
		Login: config.AuthenticationEndpointRateLimitConfig{
			PerIP:      config.RateLimitRuleConfig{Max: 1, Window: time.Minute},
			PerSubject: config.RateLimitRuleConfig{Max: 10, Window: time.Minute},
		},
	}, service)

	first := performAuthRequest(engine, "/v1/login", `{"username":"alice","password":"wrong"}`, "198.51.100.10:1001")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body = %s", first.Code, http.StatusOK, first.Body.String())
	}

	second := performAuthRequest(engine, "/v1/login", `{"username":"bob","password":"wrong"}`, "198.51.100.10:1002")
	assertAuthenticationRateLimited(t, second)
	if service.loginCalls != 1 {
		t.Fatalf("login service calls = %d, want 1", service.loginCalls)
	}

	reset := performAuthRequest(engine, "/v1/password/reset", `{"email":"alice@example.com"}`, "198.51.100.10:1003")
	if reset.Code != http.StatusOK {
		t.Fatalf("password reset status = %d, want %d; body = %s", reset.Code, http.StatusOK, reset.Body.String())
	}
	if service.passwordResetCalls != 1 {
		t.Fatalf("password reset service calls = %d, want 1", service.passwordResetCalls)
	}
}

func TestAuthenticationRateLimitBlocksOneSubjectAcrossSources(t *testing.T) {
	service := &authGuardFakeService{}
	engine := authProtectionTestEngine(t, config.AuthenticationRateLimitConfig{
		Enabled: true,
		Login: config.AuthenticationEndpointRateLimitConfig{
			PerIP:      config.RateLimitRuleConfig{Max: 10, Window: time.Minute},
			PerSubject: config.RateLimitRuleConfig{Max: 1, Window: time.Minute},
		},
	}, service)

	first := performAuthRequest(engine, "/v1/login", `{"username":"Alice@example.com","password":"wrong"}`, "198.51.100.10:1001")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body = %s", first.Code, http.StatusOK, first.Body.String())
	}

	second := performAuthRequest(engine, "/v1/login", `{"username":" alice@EXAMPLE.com ","password":"wrong"}`, "203.0.113.20:1002")
	assertAuthenticationRateLimited(t, second)
	if service.loginCalls != 1 {
		t.Fatalf("login service calls = %d, want 1", service.loginCalls)
	}
}

func TestAuthenticationRateLimitResetsAfterWindow(t *testing.T) {
	service := &authGuardFakeService{}
	engine := authProtectionTestEngine(t, config.AuthenticationRateLimitConfig{
		Enabled: true,
		Login: config.AuthenticationEndpointRateLimitConfig{
			PerIP: config.RateLimitRuleConfig{Max: 1, Window: 20 * time.Millisecond},
		},
	}, service)

	body := `{"username":"alice","password":"wrong"}`
	first := performAuthRequest(engine, "/v1/login", body, "198.51.100.10:1001")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
	}
	assertAuthenticationRateLimited(t, performAuthRequest(engine, "/v1/login", body, "198.51.100.10:1002"))

	time.Sleep(30 * time.Millisecond)
	afterReset := performAuthRequest(engine, "/v1/login", body, "198.51.100.10:1003")
	if afterReset.Code != http.StatusOK {
		t.Fatalf("status after reset = %d, want %d; body = %s", afterReset.Code, http.StatusOK, afterReset.Body.String())
	}
	if service.loginCalls != 2 {
		t.Fatalf("login service calls = %d, want 2", service.loginCalls)
	}
}

func assertAuthenticationRateLimited(t *testing.T, result *httptest.ResponseRecorder) {
	t.Helper()

	if result.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body = %s", result.Code, http.StatusTooManyRequests, result.Body.String())
	}
	for _, name := range []string{"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if got := result.Header().Get(name); got != "" {
			t.Fatalf("sensitive auth response exposed %s=%q", name, got)
		}
	}

	var payload response.ErrorResponse
	if err := json.Unmarshal(result.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload.ErrorCode != response.ErrorCodeRateLimited {
		t.Fatalf("error_code = %q, want %q", payload.ErrorCode, response.ErrorCodeRateLimited)
	}
	if payload.RequestID != "req_auth_limit" {
		t.Fatalf("request_id = %q, want req_auth_limit", payload.RequestID)
	}
}

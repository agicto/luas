package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/app"
	"github.com/zgiai/luas/api/internal/infra/config"
	infraMiddleware "github.com/zgiai/luas/api/internal/infra/middleware"
	"github.com/zgiai/luas/api/internal/starter"
	"github.com/zgiai/luas/api/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testHTTPConfig() *config.Config {
	return &config.Config{
		CORS: config.CORSConfig{
			AllowOrigins:     []string{"http://localhost:3000"},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
			ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
			AllowCredentials: true,
		},
		Middleware: config.MiddlewareConfig{
			RequestTimeout: 1,
			BodyLimit:      4,
		},
	}
}

func TestNewHTTPServerHonorsTransportConfig(t *testing.T) {
	cfg := testHTTPConfig()
	cfg.Server = config.ServerConfig{
		Host:              "127.0.0.1",
		Port:              18025,
		ReadTimeout:       61,
		ReadHeaderTimeout: 11,
		WriteTimeout:      191,
		IdleTimeout:       121,
		MaxHeaderBytes:    64 * 1024,
	}
	handler := http.NewServeMux()

	server := newHTTPServer(cfg, handler)

	if server.Addr != "127.0.0.1:18025" {
		t.Fatalf("server address = %q, want %q", server.Addr, "127.0.0.1:18025")
	}
	if server.Handler != handler {
		t.Fatal("server handler does not match")
	}
	if server.ReadTimeout != 61*time.Second {
		t.Fatalf("read timeout = %s, want 61s", server.ReadTimeout)
	}
	if server.ReadHeaderTimeout != 11*time.Second {
		t.Fatalf("read header timeout = %s, want 11s", server.ReadHeaderTimeout)
	}
	if server.WriteTimeout != 191*time.Second {
		t.Fatalf("write timeout = %s, want 191s", server.WriteTimeout)
	}
	if server.IdleTimeout != 121*time.Second {
		t.Fatalf("idle timeout = %s, want 121s", server.IdleTimeout)
	}
	if server.MaxHeaderBytes != 64*1024 {
		t.Fatalf("max header bytes = %d, want %d", server.MaxHeaderBytes, 64*1024)
	}
}

func TestNewHTTPServerDefaultsEmptyHostToLoopback(t *testing.T) {
	cfg := testHTTPConfig()
	cfg.Server.Port = 18025

	server := newHTTPServer(cfg, http.NotFoundHandler())

	if server.Addr != "127.0.0.1:18025" {
		t.Fatalf("server address = %q, want loopback", server.Addr)
	}
}

func TestNewHTTPServerFormatsIPv6ListenAddress(t *testing.T) {
	cfg := testHTTPConfig()
	cfg.Server.Host = "::1"
	cfg.Server.Port = 18025

	server := newHTTPServer(cfg, http.NotFoundHandler())

	if server.Addr != "[::1]:18025" {
		t.Fatalf("server address = %q, want bracketed IPv6", server.Addr)
	}
}

func testHTTPConfigWithRateLimit() *config.Config {
	cfg := testHTTPConfig()
	cfg.Middleware.RateLimit = config.RateLimitConfig{
		Enabled:   true,
		Max:       1,
		Window:    time.Minute,
		SkipPaths: []string{"/health"},
	}
	return cfg
}

func TestHTTPKernelDoesNotExposeMetricsByDefaultInProduction(t *testing.T) {
	cfg := testHTTPConfig()
	cfg.App = config.AppConfig{
		Name: "Luas",
		Env:  "production",
	}
	cfg.Server.Mode = gin.TestMode

	kernel := NewHttpKernel(&app.Application{
		Config:   cfg,
		Starters: starter.NewRegistry(),
	})

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()
	kernel.Engine.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("GET /metrics status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestHTTPKernelExposesMetricsWhenEnabled(t *testing.T) {
	cfg := testHTTPConfig()
	cfg.App = config.AppConfig{
		Name: "Luas",
		Env:  "development",
	}
	cfg.Server.Mode = gin.TestMode
	cfg.Metrics.Enabled = true

	kernel := NewHttpKernel(&app.Application{
		Config:   cfg,
		Starters: starter.NewRegistry(),
	})

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()
	kernel.Engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("GET /metrics Content-Type = %q, want Prometheus text format", got)
	}
}

func TestApplyGlobalMiddleware_AddsSecurityAndCORSHeaders(t *testing.T) {
	router := gin.New()
	router.Use(infraMiddleware.RequestIDWithConfig(infraMiddleware.RequestIDConfig{
		Generator: func() string { return "req_test" },
	}))
	applyGlobalMiddleware(router, testHTTPConfig())
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("X-Request-ID"); got != "req_test" {
		t.Fatalf("X-Request-ID = %q, want req_test", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %q, want SAMEORIGIN", got)
	}
	if got := w.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want localhost origin", got)
	}
}

func TestApplyGlobalMiddleware_BodyLimitUsesErrorContract(t *testing.T) {
	router := gin.New()
	router.Use(infraMiddleware.RequestIDWithConfig(infraMiddleware.RequestIDConfig{
		Generator: func() string { return "req_body_limit" },
	}))
	applyGlobalMiddleware(router, testHTTPConfig())
	router.POST("/echo", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("12345"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	var payload response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload.ErrorCode != response.ErrorCodeRequestTooLarge {
		t.Fatalf("error_code = %q, want %q", payload.ErrorCode, response.ErrorCodeRequestTooLarge)
	}
	if payload.RequestID != "req_body_limit" {
		t.Fatalf("request_id = %q, want req_body_limit", payload.RequestID)
	}
}

func TestApplyGlobalMiddleware_AddsRequestDeadline(t *testing.T) {
	router := gin.New()
	applyGlobalMiddleware(router, testHTTPConfig())
	router.GET("/deadline", func(c *gin.Context) {
		deadline, ok := c.Request.Context().Deadline()
		if !ok {
			c.String(http.StatusInternalServerError, "missing deadline")
			return
		}
		if remaining := time.Until(deadline); remaining <= 0 || remaining > 2*time.Second {
			c.String(http.StatusInternalServerError, "unexpected deadline")
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/deadline", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestApplyGlobalMiddleware_RateLimitUsesErrorContract(t *testing.T) {
	router := gin.New()
	router.Use(infraMiddleware.RequestIDWithConfig(infraMiddleware.RequestIDConfig{
		Generator: func() string { return "req_rate_limit" },
	}))
	applyGlobalMiddleware(router, testHTTPConfigWithRateLimit())
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/limited", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d; body = %s", w.Code, http.StatusTooManyRequests, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header is empty")
	}

	var payload response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload.ErrorCode != response.ErrorCodeRateLimited {
		t.Fatalf("error_code = %q, want %q", payload.ErrorCode, response.ErrorCodeRateLimited)
	}
	if payload.RequestID != "req_rate_limit" {
		t.Fatalf("request_id = %q, want req_rate_limit", payload.RequestID)
	}
}

func TestApplyGlobalMiddleware_RateLimitSkipsConfiguredPaths(t *testing.T) {
	router := gin.New()
	applyGlobalMiddleware(router, testHTTPConfigWithRateLimit())
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d; body = %s", i+1, w.Code, http.StatusNoContent, w.Body.String())
		}
	}
}

func TestApplyGlobalMiddleware_CORSPreflightDoesNotConsumeRateLimit(t *testing.T) {
	router := gin.New()
	applyGlobalMiddleware(router, testHTTPConfigWithRateLimit())
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.OPTIONS("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	preflight := httptest.NewRequest(http.MethodOptions, "/limited", nil)
	preflight.Header.Set("Origin", "http://localhost:3000")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, preflight)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("first GET status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/limited", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second GET status = %d, want %d; body = %s", w.Code, http.StatusTooManyRequests, w.Body.String())
	}
}

func TestConfigureTrustedProxies_RejectsSpoofedForwardingHeadersByDefault(t *testing.T) {
	router := gin.New()
	cfg := testHTTPConfigWithRateLimit()
	cfg.Server.TrustedProxies = nil
	configureTrustedProxies(router, cfg)
	applyGlobalMiddleware(router, cfg)
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := httptest.NewRequest(http.MethodGet, "/limited", nil)
	first.RemoteAddr = "198.51.100.10:1234"
	first.Header.Set("X-Forwarded-For", "203.0.113.1")
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", firstResponse.Code, http.StatusNoContent)
	}

	second := httptest.NewRequest(http.MethodGet, "/limited", nil)
	second.RemoteAddr = "198.51.100.10:5678"
	second.Header.Set("X-Forwarded-For", "203.0.113.2")
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, second)

	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed forwarding header changed rate-limit key: status = %d, want %d", secondResponse.Code, http.StatusTooManyRequests)
	}
}

func TestConfigureTrustedProxies_UsesForwardedClientIPFromConfiguredProxy(t *testing.T) {
	router := gin.New()
	cfg := testHTTPConfigWithRateLimit()
	cfg.Server.TrustedProxies = []string{"198.51.100.10"}
	configureTrustedProxies(router, cfg)
	applyGlobalMiddleware(router, cfg)
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, clientIP := range []string{"203.0.113.1", "203.0.113.2"} {
		request := httptest.NewRequest(http.MethodGet, "/limited", nil)
		request.RemoteAddr = "198.51.100.10:1234"
		request.Header.Set("X-Forwarded-For", clientIP)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusNoContent {
			t.Fatalf("client %s status = %d, want %d", clientIP, response.Code, http.StatusNoContent)
		}
	}
}

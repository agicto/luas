package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHelmetWithConfig_AppliesConfiguredHeaders(t *testing.T) {
	router := gin.New()
	router.Use(HelmetWithConfig(HelmetConfig{
		XSSProtection:             "0",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "DENY",
		HSTSMaxAge:                60,
		HSTSIncludeSubdomains:     true,
		HSTSPreload:               true,
		ContentSecurityPolicy:     "default-src 'self'",
		ReferrerPolicy:            "no-referrer",
		PermissionsPolicy:         "camera=()",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-site",
	}))
	router.GET("/headers", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/headers", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	want := map[string]string{
		"X-XSS-Protection":             "0",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Strict-Transport-Security":    "max-age=60; includeSubDomains; preload",
		"Content-Security-Policy":      "default-src 'self'",
		"Referrer-Policy":              "no-referrer",
		"Permissions-Policy":           "camera=()",
		"Cross-Origin-Embedder-Policy": "require-corp",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-site",
	}
	for name, value := range want {
		if got := response.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestHelmetWithConfig_DisablesEmptyAndZeroValueHeaders(t *testing.T) {
	router := gin.New()
	router.Use(HelmetWithConfig(HelmetConfig{}))
	router.GET("/headers", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/headers", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if len(response.Header()) != 0 {
		t.Fatalf("response headers = %v, want no Helmet headers", response.Header())
	}
}

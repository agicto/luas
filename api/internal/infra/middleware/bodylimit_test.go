package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimitWithConfig_UsesDefaultLimitForZeroValue(t *testing.T) {
	router := gin.New()
	router.Use(BodyLimitWithConfig(BodyLimitConfig{}))
	router.POST("/upload", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	body := strings.NewReader(strings.Repeat("a", 5*1024*1024))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestBodyLimitWithConfig_DoesNotWrapEmptyBody(t *testing.T) {
	router := gin.New()
	router.Use(BodyLimitWithConfig(BodyLimitConfig{MaxSize: 4}))
	router.GET("/empty", func(c *gin.Context) {
		if c.Request.Body != http.NoBody {
			t.Fatalf("empty request body was wrapped as %T", c.Request.Body)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestBodyLimitWithConfig_ConstrainsUnknownLengthBody(t *testing.T) {
	router := gin.New()
	router.Use(BodyLimitWithConfig(BodyLimitConfig{MaxSize: 4}))
	router.POST("/upload", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			_ = c.Error(err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("12345"))
	req.ContentLength = -1
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}
}

func TestBodyLimitWithConfig_UsesErrorContract(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDWithConfig(RequestIDConfig{
		Generator: func() string { return "req_body_limit" },
	}))
	router.Use(BodyLimitWithConfig(BodyLimitConfig{
		MaxSize: 4,
	}))
	router.POST("/upload", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("12345"))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"error_code":"COMMON.REQUEST_TOO_LARGE"`) {
		t.Fatalf("body = %s, want COMMON.REQUEST_TOO_LARGE", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"request_id":"req_body_limit"`) {
		t.Fatalf("body = %s, want request_id", w.Body.String())
	}
}

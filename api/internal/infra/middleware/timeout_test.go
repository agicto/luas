package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/pkg/response"
)

func TestTimeoutWithConfig_UsesErrorContract(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDWithConfig(RequestIDConfig{
		Generator: func() string { return "req_timeout" },
	}))
	router.Use(TimeoutWithConfig(TimeoutConfig{
		Timeout:      5 * time.Millisecond,
		ErrorMessage: "Request timeout",
	}))
	router.GET("/slow", func(c *gin.Context) {
		<-c.Request.Context().Done()
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}

	var payload response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload.ErrorCode != response.ErrorCodeTimeout {
		t.Fatalf("error_code = %q, want %q", payload.ErrorCode, response.ErrorCodeTimeout)
	}
	if payload.RequestID != "req_timeout" {
		t.Fatalf("request_id = %q, want req_timeout", payload.RequestID)
	}
}

func TestTimeoutWithConfig_DoesNotOverrideWrittenResponse(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutWithConfig(TimeoutConfig{
		Timeout: 5 * time.Millisecond,
	}))
	router.GET("/slow-write", func(c *gin.Context) {
		time.Sleep(20 * time.Millisecond)
		c.String(http.StatusOK, "late but written")
	})

	req := httptest.NewRequest(http.MethodGet, "/slow-write", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if got := w.Body.String(); got != "late but written" {
		t.Fatalf("body = %q, want late but written", got)
	}
}

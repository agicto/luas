package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDWithConfig_CanonicalizesCustomHeader(t *testing.T) {
	router := gin.New()
	router.Use(RequestIDWithConfig(RequestIDConfig{
		Header:    "x-correlation-ID",
		Generator: func() string { return "generated" },
	}))
	router.GET("/request-id", func(c *gin.Context) {
		if got := GetRequestID(c); got != "provided" {
			t.Fatalf("request ID = %q, want provided", got)
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	request.Header.Set("X-Correlation-Id", "provided")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get("X-Correlation-ID"); got != "provided" {
		t.Fatalf("response request ID = %q, want provided", got)
	}
}

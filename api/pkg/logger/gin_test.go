package logger

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGinLoggerDoesNotRecordQueryValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	memory := NewMemoryHandler(10)
	runtimeLogger := New(ProductionConfig())
	runtimeLogger.handlers = []Handler{memory}
	previous := Default()
	SetDefault(runtimeLogger)
	t.Cleanup(func() { SetDefault(previous) })

	engine := gin.New()
	engine.Use(GinLogger())
	engine.GET("/search/:term", func(c *gin.Context) {
		_ = c.Error(errors.New("free-form-sensitive-error"))
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(
		http.MethodGet,
		"/search/customer%40example.com?page=2&access_token=super-secret-access-token",
		nil,
	)
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	entries := memory.Recent(1)
	require.Len(t, entries, 1)
	assert.Equal(t, "/search/:term", entries[0].Context["path"])
	assert.NotContains(t, entries[0].Context, "query")
	assert.NotContains(t, entries[0].Context, "raw_query")
	assert.NotContains(t, entries[0].Context, "access_token")
	assert.NotContains(t, entries[0].Context, "super-secret-access-token")
	assert.NotContains(t, entries[0].Context, "customer@example.com")
	assert.NotContains(t, entries[0].Context, "errors")
	assert.Equal(t, 1, entries[0].Context["error_count"])
	assert.NotContains(t, entries[0].Message, "free-form-sensitive-error")
}

func TestLoggerRedactsSensitiveContextRecursively(t *testing.T) {
	memory := NewMemoryHandler(10)
	runtimeLogger := New(ProductionConfig())
	runtimeLogger.handlers = []Handler{memory}

	runtimeLogger.Info("auth.context", map[string]any{
		"request_id": "req-redaction",
		"password":   "correct horse battery staple",
		"nested": map[string]any{
			"access_token": "nested-access-token",
			"outcome":      "denied",
		},
	})

	entries := memory.Recent(1)
	require.Len(t, entries, 1)
	assert.Equal(t, "[REDACTED]", entries[0].Context["password"])
	nested, ok := entries[0].Context["nested"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[REDACTED]", nested["access_token"])
	assert.Equal(t, "denied", nested["outcome"])
	assert.Equal(t, "req-redaction", entries[0].RequestID)
}

func TestGinLoggerBoundsUnmatchedRouteCardinality(t *testing.T) {
	gin.SetMode(gin.TestMode)
	memory := NewMemoryHandler(10)
	runtimeLogger := New(ProductionConfig())
	runtimeLogger.handlers = []Handler{memory}
	previous := Default()
	SetDefault(runtimeLogger)
	t.Cleanup(func() { SetDefault(previous) })

	engine := gin.New()
	engine.Use(GinLogger())
	request := httptest.NewRequest(http.MethodGet, "/unknown/private-value?token=secret", nil)
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	entries := memory.Recent(1)
	require.Len(t, entries, 1)
	assert.Equal(t, "unmatched", entries[0].Context["path"])
	assert.NotContains(t, entries[0].Context, "private-value")
	assert.NotContains(t, entries[0].Context, "secret")
}

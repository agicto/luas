package errors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/pkg/logger"
)

func TestDefaultConfigDoesNotExposeDebugDetails(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Debug || cfg.ShowStack {
		t.Fatalf("DefaultConfig() exposes debug details: debug=%v stack=%v", cfg.Debug, cfg.ShowStack)
	}
	if !cfg.LogErrors {
		t.Fatal("DefaultConfig() must keep error logging enabled")
	}
}

func TestHandlerLogsRouteShapeWithoutInternalErrorValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	memory := logger.NewMemoryHandler(10)
	runtimeLogger := logger.New(logger.ProductionConfig())
	runtimeLogger.AddHandler(memory)
	previous := logger.Default()
	logger.SetDefault(runtimeLogger)
	t.Cleanup(func() { logger.SetDefault(previous) })

	engine := gin.New()
	engine.Use(Handler(Config{LogErrors: true}))
	engine.GET("/resources/:resource", func(c *gin.Context) {
		Abort(c, LegacyInternal("safe public message").WithInternal(assert.AnError))
	})
	request := httptest.NewRequest(http.MethodGet, "/resources/private-customer", nil)
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	entries := memory.Recent(1)
	require.Len(t, entries, 1)
	assert.Equal(t, "/resources/:resource", entries[0].Context["path"])
	assert.Equal(t, "*errors.errorString", entries[0].Context["internal_error_type"])
	assert.NotContains(t, entries[0].Context, "private-customer")
	assert.NotContains(t, entries[0].Context, assert.AnError.Error())
}

func TestRenderDebugHTMLRedactsAtFinalBoundary(t *testing.T) {
	output := renderDebugHTML(DebugPageData{
		Title:   "<script>alert(1)</script>",
		Message: "safe message",
		Request: RequestInfo{
			URL: "/callback?access_token=url-secret&invalid=%zz",
			Headers: map[string]string{
				"Authorization": "Bearer header-secret",
			},
			Query: map[string]string{
				"code": "query-secret",
			},
		},
	})

	assert.NotContains(t, output, "<script>alert(1)</script>")
	assert.Contains(t, output, "&lt;script&gt;alert(1)&lt;/script&gt;")
	assert.NotContains(t, output, "url-secret")
	assert.NotContains(t, output, "header-secret")
	assert.NotContains(t, output, "query-secret")
	assert.Contains(t, output, "[REDACTED]")
}

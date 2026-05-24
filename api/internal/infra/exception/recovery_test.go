package exception_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/database"
	"github.com/zgiai/zgo/internal/infra/exception"
	infraMiddleware "github.com/zgiai/zgo/internal/infra/middleware"
	"github.com/zgiai/zgo/internal/infra/tracing"
	"github.com/zgiai/zgo/pkg/logger"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestRecoveryRendersDebugPageWithRouteTraceSQLAndLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		otel.SetTracerProvider(previousProvider)
		_ = tp.Shutdown(context.Background())
	}()

	logger.SetDefault(logger.New(logger.DefaultConfig()))

	cfg := &config.Config{}
	cfg.App.Debug = true
	cfg.App.Env = "development"
	cfg.Database.Enabled = true
	cfg.Database.Driver = "sqlite"
	cfg.Database.Memory = true
	cfg.Database.SlowThreshold = time.Second
	cfg.Database.IgnoreRecordNotFound = true

	db, err := database.NewDB(cfg)
	require.NoError(t, err)

	engine := gin.New()
	engine.Use(infraMiddleware.RequestID())
	engine.Use(tracing.Middleware("zgo-test"))
	engine.Use(tracing.InjectTraceID())
	engine.Use(logger.GinLogger())
	engine.Use(exception.Recovery(true))
	engine.GET("/panic", func(c *gin.Context) {
		c.Set("route_name", "debug.panic")
		logger.Info("before panic", map[string]any{
			"request_id": c.GetString("request_id"),
			"phase":      "before_panic",
		})

		var result int
		if err := db.WithContext(c.Request.Context()).Raw("SELECT 1").Scan(&result).Error; err != nil {
			panic(err)
		}
		panic("boom from test")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic?hello=world", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("X-Request-ID", "req-debug-page-1")

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")

	body := recorder.Body.String()
	require.Contains(t, body, "debug.panic")
	require.Contains(t, body, "req-debug-page-1")
	require.Contains(t, body, "SELECT 1")
	require.Contains(t, body, "before panic")
	require.Contains(t, body, "hello")
	require.Contains(t, body, "world")

	traceID := strings.TrimSpace(recorder.Header().Get("X-Trace-ID"))
	require.NotEmpty(t, traceID)
	require.Contains(t, body, traceID)
}

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
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/infra/exception"
	infraMiddleware "github.com/zgiai/luas/api/internal/infra/middleware"
	"github.com/zgiai/luas/api/internal/infra/tracing"
	"github.com/zgiai/luas/api/pkg/logger"
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
	cfg.Database.MaxIdleConns = 1
	cfg.Database.MaxOpenConns = 1
	cfg.Database.ConnMaxIdleTime = config.DefaultDatabaseConnMaxIdleTime
	cfg.Database.ConnMaxLifetime = config.DefaultDatabaseConnMaxLifetime
	cfg.Database.ConnectTimeout = config.DefaultDatabaseConnectTimeout
	cfg.Database.SlowThreshold = time.Second
	cfg.Database.IgnoreRecordNotFound = true

	db, err := database.NewDB(cfg)
	require.NoError(t, err)

	engine := gin.New()
	engine.Use(infraMiddleware.RequestID())
	engine.Use(tracing.Middleware("luas-test"))
	engine.Use(tracing.InjectTraceID())
	engine.Use(logger.GinLogger())
	engine.Use(exception.Recovery(true))
	engine.GET("/panic/:subject", func(c *gin.Context) {
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

	req := httptest.NewRequest(
		http.MethodGet,
		"/panic/private-customer?hello=%3Cscript%3Ealert%281%29%3C%2Fscript%3E&access_token=query-secret",
		nil,
	)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("X-Request-ID", "req-debug-page-1")
	req.Header.Set("Authorization", "Bearer authorization-secret")
	req.Header.Set("Cookie", "session=cookie-secret")
	req.Header.Set("X-API-Key", "luas_header.secret")

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
	require.Contains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt;")
	require.Contains(t, body, "[REDACTED]")
	require.NotContains(t, body, "query-secret")
	require.NotContains(t, body, "authorization-secret")
	require.NotContains(t, body, "cookie-secret")
	require.NotContains(t, body, "luas_header.secret")
	require.NotContains(t, body, "private-customer")
	require.NotContains(t, body, "<script>alert(1)</script>")

	traceID := strings.TrimSpace(recorder.Header().Get("X-Trace-ID"))
	require.NotEmpty(t, traceID)
	require.Contains(t, body, traceID)
}

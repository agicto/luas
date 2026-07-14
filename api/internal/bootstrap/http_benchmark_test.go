package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/metrics"
	infraMiddleware "github.com/zgiai/luas/api/internal/infra/middleware"
)

type benchmarkResponseWriter struct {
	header http.Header
	status int
	size   int
}

func newBenchmarkResponseWriter() *benchmarkResponseWriter {
	return &benchmarkResponseWriter{header: make(http.Header)}
}

func (w *benchmarkResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchmarkResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *benchmarkResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.size += len(body)
	return len(body), nil
}

func (w *benchmarkResponseWriter) reset() {
	w.status = 0
	w.size = 0
}

func benchmarkHTTPRouter(metricsEnabled bool) *gin.Engine {
	router := gin.New()
	router.Use(infraMiddleware.RequestIDWithConfig(infraMiddleware.RequestIDConfig{
		Generator: func() string { return "req_benchmark" },
	}))
	if metricsEnabled {
		router.Use(metrics.Middleware())
	}
	applyGlobalMiddleware(router, testHTTPConfig())
	router.GET("/v1/items/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return router
}

func BenchmarkHTTPMiddlewareChain(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		name := "metrics_disabled"
		if enabled {
			name = "metrics_enabled"
		}

		b.Run(name, func(b *testing.B) {
			router := benchmarkHTTPRouter(enabled)
			request := httptest.NewRequest(http.MethodGet, "/v1/items/42", nil)
			request.Header.Set("Origin", "http://localhost:3000")
			writer := newBenchmarkResponseWriter()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				request.Body = http.NoBody
				writer.reset()
				router.ServeHTTP(writer, request)
			}
		})
	}
}

func TestHTTPMiddlewareChainAllocationBudget(t *testing.T) {
	const maxAllocsPerRequest = 21.0

	for _, enabled := range []bool{false, true} {
		name := "metrics_disabled"
		if enabled {
			name = "metrics_enabled"
		}

		t.Run(name, func(t *testing.T) {
			router := benchmarkHTTPRouter(enabled)
			request := httptest.NewRequest(http.MethodGet, "/v1/items/42", nil)
			request.Header.Set("Origin", "http://localhost:3000")
			writer := newBenchmarkResponseWriter()

			allocs := testing.AllocsPerRun(1000, func() {
				request.Body = http.NoBody
				writer.reset()
				router.ServeHTTP(writer, request)
			})
			if allocs > maxAllocsPerRequest {
				t.Fatalf("steady-state allocations = %.0f/request, budget = %.0f", allocs, maxAllocsPerRequest)
			}
		})
	}
}

var _ http.ResponseWriter = (*benchmarkResponseWriter)(nil)

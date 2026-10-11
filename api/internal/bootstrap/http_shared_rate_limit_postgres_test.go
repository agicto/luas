package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fastagi/fastbuild/api/database/migrations"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/infra/ratelimit"
	testplatform "github.com/fastagi/fastbuild/api/internal/infra/testing"
)

// TestApplyGlobalMiddleware_SharedStoreEnforcesOneBudgetAcrossReplicas proves the kernel hands the
// configured store factory to the global limiter: two engines model two replicas, and the second
// replica sees the hits the first one consumed.
func TestApplyGlobalMiddleware_SharedStoreEnforcesOneBudgetAcrossReplicas(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil)
	if err := migrations.All()[migrations.RateLimitBucketsMigration].Up(db); err != nil {
		t.Fatalf("migrate rate_limit_buckets: %v", err)
	}
	cfg := testHTTPConfigWithRateLimit()
	cfg.Middleware.RateLimit.Max = 2
	cfg.Middleware.RateLimitStore = config.RateLimitStoreConfig{
		Driver:  config.RateLimitStorePostgres,
		Timeout: 5 * time.Second,
		Failure: config.RateLimitStoreFailureClosed,
	}

	replicas := make([]*gin.Engine, 2)
	for index := range replicas {
		stores, err := ratelimit.ProvideStores(cfg, db)
		if err != nil {
			t.Fatalf("ProvideStores: %v", err)
		}
		if !stores.Shared() {
			t.Fatal("RATE_LIMIT_STORE=postgres must provide shared stores")
		}
		engine := gin.New()
		applyGlobalMiddleware(engine, cfg, stores)
		engine.GET("/limited", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		replicas[index] = engine
	}

	want := []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests}
	for index, status := range want {
		w := httptest.NewRecorder()
		replicas[index%2].ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/limited", nil))
		if w.Code != status {
			t.Fatalf("request %d on replica %d status = %d, want %d", index+1, index%2, w.Code, status)
		}
	}
}

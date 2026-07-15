package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/internal/starter"
)

type testAuditMiddlewareModule struct{}

func (testAuditMiddlewareModule) Name() string { return "audit" }

func (testAuditMiddlewareModule) RegisterMiddleware(r *router.Router) {
	r.AliasMiddleware("audit", func(c *gin.Context) { c.Next() })
}

func testStarterRegistry() *starter.Registry {
	registry := starter.NewRegistry()
	registry.RegisterModule(testAuditMiddlewareModule{})
	return registry
}

func TestSetupDoesNotExposeUnwiredOperationalSurfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	Setup(engine, testStarterRegistry())

	for _, route := range engine.Routes() {
		for _, prefix := range []string{"/monitor", "/swagger"} {
			if strings.HasPrefix(route.Path, prefix) {
				t.Fatalf("route %s %s exposes unwired operational surface %q", route.Method, route.Path, prefix)
			}
		}
	}
}

func TestWelcomeDoesNotLinkUnwiredOperationalSurfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	Setup(engine, testStarterRegistry())

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, path := range []string{"/monitor", "/swagger"} {
		if strings.Contains(response.Body.String(), path) {
			t.Fatalf("welcome page links unwired operational surface %q", path)
		}
	}
}

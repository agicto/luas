package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/starter"
)

func TestSetupDoesNotExposeUnwiredOperationalSurfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	Setup(engine, starter.NewRegistry())

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
	Setup(engine, starter.NewRegistry())

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

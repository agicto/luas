package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestDiagnosticsServerIsOffByDefault(t *testing.T) {
	assert.Nil(t, newDiagnosticsServer(&config.Config{}))
	assert.Nil(t, newDiagnosticsServer(nil))
}

func TestDiagnosticsServerExposesOnlyRuntimeProfiles(t *testing.T) {
	srv := newDiagnosticsServer(&config.Config{Server: config.ServerConfig{DiagnosticsAddr: "127.0.0.1:6060"}})
	require.NotNil(t, srv)

	index := httptest.NewRecorder()
	srv.Handler.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	assert.Equal(t, http.StatusOK, index.Code)
	assert.Contains(t, index.Body.String(), "goroutine")

	other := httptest.NewRecorder()
	srv.Handler.ServeHTTP(other, httptest.NewRequest(http.MethodGet, "/v1/health", nil))
	assert.Equal(t, http.StatusNotFound, other.Code)
}

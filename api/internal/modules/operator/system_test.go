package operator

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/modules/setting"
)

func TestSystemReportsStartersAndDatabase(t *testing.T) {
	f := newFixture(t)
	f.handler.cfg.Starters.Optional = []string{config.StarterOperator, config.StarterOrganization}
	f.engine.GET("/v1/operator/system", f.handler.requireOperator, f.handler.System)

	recorder := f.do(t, call{method: http.MethodGet, path: "/v1/operator/system", cookie: testCredential})
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var body struct {
		Data systemResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, []string{"audit", "apikey", "user", "operator", "organization"}, body.Data.Starters)
	assert.Equal(t, "ok", body.Data.Database)
	assert.NotEmpty(t, body.Data.GoVersion)
}

func TestSystemReportsDatabaseOutage(t *testing.T) {
	f := newFixture(t)
	f.engine.GET("/v1/operator/system", f.handler.System)
	f.grants.err = errors.New("connection refused")

	recorder := f.do(t, call{method: http.MethodGet, path: "/v1/operator/system"})
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"database":"unavailable"`)
}

func TestSettingRoutesMountOnlyWhenSettingStarterIsSelected(t *testing.T) {
	settings := &setting.Handler{}
	without := &config.Config{}
	assert.Nil(t, NewHandler(nil, nil, without, settings).settings)

	with := &config.Config{}
	with.Starters.Optional = []string{config.StarterOrganization, config.StarterSetting}
	assert.Same(t, settings, NewHandler(nil, nil, with, settings).settings)
}

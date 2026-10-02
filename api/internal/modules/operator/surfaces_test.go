package operator

import (
	"net/http"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/internal/modules/notification"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/webhook"
)

var starterSurfaceRoutes = map[string][]string{
	config.StarterOrganization: {
		"operator.organizations.index",
		"operator.organizations.show",
		"operator.organization-members.index",
	},
	config.StarterWebhook: {
		"operator.webhook-endpoints.index",
		"operator.webhook-deliveries.index",
		"operator.webhook-delivery-attempts.index",
		"operator.webhook-deliveries.replay",
	},
	config.StarterNotification: {"operator.notification-deliveries.index"},
}

func allSurfaces() Surfaces {
	return Surfaces{
		Organizations: &organization.OperatorHandler{},
		Webhooks:      &webhook.OperatorHandler{},
		Notifications: &notification.OperatorHandler{},
	}
}

func TestStarterSurfacesMountOnlyForSelectedStarters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		optional []string
		mounted  []string
	}{
		{name: "operator alone", optional: []string{config.StarterOperator}},
		{
			name:     "organization",
			optional: []string{config.StarterOperator, config.StarterOrganization},
			mounted:  []string{config.StarterOrganization},
		},
		{
			name:     "notification without organization",
			optional: []string{config.StarterOperator, config.StarterNotification},
			mounted:  []string{config.StarterNotification},
		},
		{
			name: "all",
			optional: []string{
				config.StarterOperator, config.StarterOrganization, config.StarterWebhook, config.StarterNotification,
			},
			mounted: []string{config.StarterOrganization, config.StarterWebhook, config.StarterNotification},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Starters.Optional = test.optional
			routes := router.New(gin.New()).Prefix("/v1")
			NewHandler(nil, nil, cfg, allSurfaces()).RegisterRoutes(routes)
			names := routes.Routes()
			for starter, routeNames := range starterSurfaceRoutes {
				for _, name := range routeNames {
					_, exists := names[name]
					assert.Equal(t, slices.Contains(test.mounted, starter), exists, name)
				}
			}
		})
	}
}

func TestWebhookSurfaceNeedsTheOrganizationSurface(t *testing.T) {
	cfg := &config.Config{}
	cfg.Starters.Optional = []string{config.StarterOperator, config.StarterOrganization, config.StarterWebhook}
	surfaces := allSurfaces()
	surfaces.Organizations = nil
	assert.Nil(t, NewHandler(nil, nil, cfg, surfaces).webhooks)
}

// The surface handlers here have no store, so reaching one would panic: a rejected caller proves the
// operator guard runs before any starter lookup.
func TestStarterSurfacesRejectCallersBeforeAnyLookup(t *testing.T) {
	f := newFixture(t)
	cfg := &config.Config{Operator: config.OperatorConfig{AllowedOrigins: []string{testOrigin}}}
	cfg.App.Env = "production"
	cfg.Starters.Optional = []string{
		config.StarterOperator, config.StarterOrganization, config.StarterWebhook, config.StarterNotification,
	}
	engine := gin.New()
	NewHandler(f.handler.service, nil, cfg, allSurfaces()).RegisterRoutes(router.New(engine).Prefix("/v1"))
	f.engine = engine

	paths := []call{
		{method: http.MethodGet, path: "/v1/operator/organizations"},
		{method: http.MethodGet, path: "/v1/operator/organizations/1/members"},
		{method: http.MethodGet, path: "/v1/operator/organizations/1/webhook-deliveries"},
		{method: http.MethodPost, path: "/v1/operator/organizations/1/webhook-deliveries/1/replay"},
		{method: http.MethodGet, path: "/v1/operator/notification-deliveries"},
	}
	for _, request := range paths {
		anonymous := f.do(t, request)
		require.Equal(t, http.StatusUnauthorized, anonymous.Code, request.path)
	}

	delete(f.grants.operators, 1)
	for _, request := range paths {
		request.cookie = testCredential
		request.origin = testOrigin
		request.csrf = csrfToken(testCredential)
		forbidden := f.do(t, request)
		require.Equal(t, http.StatusForbidden, forbidden.Code, request.path)
		assert.Equal(t, "OPERATOR.FORBIDDEN", errorCode(t, forbidden))
	}

	f.grants.operators[1] = true
	withoutCSRF := f.do(t, call{
		method: http.MethodPost,
		path:   "/v1/operator/organizations/1/webhook-deliveries/1/replay",
		cookie: testCredential,
		origin: testOrigin,
	})
	require.Equal(t, http.StatusForbidden, withoutCSRF.Code)
	assert.Equal(t, "OPERATOR.CSRF_REJECTED", errorCode(t, withoutCSRF))
}

package webhook

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches organization-scoped webhook management resources.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth", "organization_context")
		auth.GET("/webhook-event-types", h.EventTypes).Name("webhook-event-types.index")
		auth.GET("/webhook-endpoints", h.ListEndpoints).Name("webhook-endpoints.index")
		auth.POST("/webhook-endpoints", h.CreateEndpoint).Name("webhook-endpoints.store")
		auth.PATCH("/webhook-endpoints/:id", h.UpdateEndpoint).Name("webhook-endpoints.update").WhereNumber("id")
		auth.DELETE("/webhook-endpoints/:id", h.DeleteEndpoint).Name("webhook-endpoints.destroy").WhereNumber("id")
		auth.PUT("/webhook-endpoints/:id/status", h.ReplaceEndpointStatus).
			Name("webhook-endpoint-status.update").WhereNumber("id")
		auth.POST("/webhook-endpoints/:id/secret-rotations", h.RotateEndpointSecret).
			Name("webhook-endpoint-secret-rotations.store").WhereNumber("id")
		auth.POST("/webhook-endpoints/:id/tests", h.TestEndpoint).
			Name("webhook-endpoint-tests.store").WhereNumber("id")
		auth.GET("/webhook-deliveries", h.ListDeliveries).Name("webhook-deliveries.index")
		auth.GET("/webhook-deliveries/:id/attempts", h.ListAttempts).
			Name("webhook-delivery-attempts.index").WhereNumber("id")
	})
}

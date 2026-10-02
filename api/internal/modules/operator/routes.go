package operator

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches the operator browser session and operator-only routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	signIn := r.POST("/operator/session", h.SignIn).Name("operator.session.store")
	if h.guard != nil {
		if limit := h.guard.LoginIPMiddleware(); limit != nil {
			signIn.Middleware(limit)
		}
	}
	r.DELETE("/operator/session", h.SignOut).Name("operator.session.destroy")

	r.Group("/operator", func(operator *router.Router) {
		operator.Use(h.requireOperator)
		operator.GET("/session", h.Current).Name("operator.session.show")

		operator.GET("/users", h.ListUsers).Name("operator.users.index")
		operator.GET("/users/:id", h.GetUser).Name("operator.users.show").WhereNumber("id")
		operator.POST("/users/:id/disable", h.DisableUser).Name("operator.users.disable").WhereNumber("id")
		operator.POST("/users/:id/enable", h.EnableUser).Name("operator.users.enable").WhereNumber("id")
		operator.POST("/users/:id/sessions/revoke", h.RevokeUserSessions).
			Name("operator.user-sessions.revoke").WhereNumber("id")

		operator.GET("/audit-logs", h.ListAuditLogs).Name("operator.audit-logs.index")
		operator.GET("/system", h.System).Name("operator.system.show")

		if h.settings != nil {
			operator.GET("/settings", h.settings.AppList).Name("operator.settings.index")
			operator.PATCH("/settings/:key", h.settings.AppSet).Name("operator.settings.update")
			operator.DELETE("/settings/:key", h.settings.AppReset).Name("operator.settings.destroy")
		}

		if h.organizations != nil {
			operator.GET("/organizations", h.organizations.List).Name("operator.organizations.index")
			operator.GET("/organizations/:id", h.organizations.Get).
				Name("operator.organizations.show").WhereNumber("id")
			operator.GET("/organizations/:id/members", h.organizations.ListMembers).
				Name("operator.organization-members.index").WhereNumber("id")
		}
		if h.webhooks != nil {
			operator.Group("/organizations/:id", func(organization *router.Router) {
				organization.Use(h.organizations.Require)
				organization.GET("/webhook-endpoints", h.webhooks.ListEndpoints).
					Name("operator.webhook-endpoints.index").WhereNumber("id")
				organization.GET("/webhook-deliveries", h.webhooks.ListDeliveries).
					Name("operator.webhook-deliveries.index").WhereNumber("id")
				organization.GET("/webhook-deliveries/:delivery_id/attempts", h.webhooks.ListAttempts).
					Name("operator.webhook-delivery-attempts.index").WhereNumber("id").WhereNumber("delivery_id")
				organization.POST("/webhook-deliveries/:delivery_id/replay", h.webhooks.ReplayDelivery).
					Name("operator.webhook-deliveries.replay").WhereNumber("id").WhereNumber("delivery_id")
			})
		}
		if h.notifications != nil {
			operator.GET("/notification-deliveries", h.notifications.ListDeliveries).
				Name("operator.notification-deliveries.index")
		}
	})
}

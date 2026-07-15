package setting

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches public app, private user, and active-organization setting routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.GET("/settings/public", h.PublicApp).Name("settings.public.index")

	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.GET("/settings/user", h.UserList).Name("settings.user.index")
		auth.PATCH("/settings/user/:key", h.UserSet).Name("settings.user.update")
		auth.DELETE("/settings/user/:key", h.UserReset).Name("settings.user.destroy")

		auth.Group("", func(contextual *router.Router) {
			contextual.WithMiddleware("organization_context")
			contextual.GET("/organization-settings", h.OrganizationList).Name("organization-settings.index")
			contextual.PATCH("/organization-settings/:key", h.OrganizationSet).Name("organization-settings.update")
			contextual.DELETE("/organization-settings/:key", h.OrganizationReset).Name("organization-settings.destroy")
		})
	})
}

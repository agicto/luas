package usage

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches private user and active-organization usage reads.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.GET("/usage/user", h.UserList).Name("usage.user.index")

		auth.Group("", func(contextual *router.Router) {
			contextual.WithMiddleware("organization_context")
			contextual.GET("/organization-usage", h.OrganizationList).Name("organization-usage.index")
		})
	})
}

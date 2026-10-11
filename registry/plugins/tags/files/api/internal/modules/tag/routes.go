package tag

import "github.com/fastagi/fastbuild/api/internal/infra/router"

// RegisterRoutes attaches organization-scoped tag resources.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth", "organization_context")
		auth.GET("/tags", h.List).Name("tags.index")
		auth.POST("/tags", h.Create).Name("tags.store")
		auth.GET("/tags/:id", h.Get).Name("tags.show").WhereNumber("id")
		auth.PATCH("/tags/:id", h.Update).Name("tags.update").WhereNumber("id")
		auth.DELETE("/tags/:id", h.Delete).Name("tags.destroy").WhereNumber("id")
		auth.GET("/tags/:id/resources", h.Resources).Name("tags.resources.index").WhereNumber("id")
		auth.GET("/resource-tags/:resource_type/:resource_id", h.ResourceTags).Name("resource-tags.show")
		auth.PUT("/resource-tags/:resource_type/:resource_id", h.ReplaceResourceTags).Name("resource-tags.update")
	})
}

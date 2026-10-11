package knowledge

import "github.com/fastagi/fastbuild/api/internal/infra/router"

// RegisterRoutes attaches organization-scoped knowledge resources.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth", "organization_context")
		auth.GET("/knowledge-articles", h.List).Name("knowledge-articles.index")
		auth.POST("/knowledge-articles", h.Create).Name("knowledge-articles.store")
		auth.GET("/knowledge-articles/:id", h.Get).Name("knowledge-articles.show").WhereNumber("id")
		auth.PATCH("/knowledge-articles/:id", h.Update).Name("knowledge-articles.update").WhereNumber("id")
		auth.DELETE("/knowledge-articles/:id", h.Delete).Name("knowledge-articles.destroy").WhereNumber("id")
		auth.GET("/knowledge-search", h.Search).Name("knowledge-search.index")
	})
}

package team

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes registers HTTP routes for the team module.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("/teams", func(group *router.Router) {
		group.WithMiddleware("auth")
		group.GET("", h.List).Name("team.index")
		group.POST("", h.Create).Name("team.store")
		group.GET("/:id", h.Get).Name("team.show").WhereNumber("id")
		group.PUT("/:id", h.Update).Name("team.update").WhereNumber("id")
		group.DELETE("/:id", h.Delete).Name("team.destroy").WhereNumber("id")
	})
}

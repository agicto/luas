package access

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes registers HTTP routes for the access module.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.GET("/permissions", h.Permissions).Name("access.permissions")
		auth.Group("/teams/:id/roles", func(group *router.Router) {
			group.GET("", h.ListRoles).Name("access.roles.index").WhereNumber("id")
			group.POST("", h.CreateRole).Name("access.roles.store").WhereNumber("id")
			group.GET("/:role_id", h.GetRole).Name("access.roles.show").WhereNumber("id").WhereNumber("role_id")
			group.PUT("/:role_id", h.UpdateRole).Name("access.roles.update").WhereNumber("id").WhereNumber("role_id")
			group.DELETE("/:role_id", h.DeleteRole).Name("access.roles.destroy").WhereNumber("id").WhereNumber("role_id")
		})
	})
}

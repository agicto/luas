package permission

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches permission routes behind authentication and organization context.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth", "organization_context")
		auth.GET("/permission-context", h.GetEffective).Name("permission-context.show")
		auth.GET("/permissions", h.ListPermissions).Name("permissions.index")
		auth.GET("/access-roles", h.ListRoles).Name("access-roles.index")
		auth.POST("/access-roles", h.CreateRole).Name("access-roles.store")
		auth.GET("/access-roles/:id", h.GetRole).Name("access-roles.show").WhereNumber("id")
		auth.PATCH("/access-roles/:id", h.UpdateRole).Name("access-roles.update").WhereNumber("id")
		auth.DELETE("/access-roles/:id", h.DeleteRole).Name("access-roles.destroy").WhereNumber("id")
		auth.GET("/organization-members/:member_id/access-roles", h.GetMemberRoles).
			Name("member-access-roles.show").
			WhereNumber("member_id")
		auth.PUT("/organization-members/:member_id/access-roles", h.ReplaceMemberRoles).
			Name("member-access-roles.update").
			WhereNumber("member_id")
	})
}

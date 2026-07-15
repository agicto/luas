package organization

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterMiddleware exposes verified active organization context to downstream modules.
func (h *Handler) RegisterMiddleware(r *router.Router) {
	r.MiddlewareGroup("organization_context", h.contextResolver.Middleware())
}

// RegisterRoutes registers organization ownership routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.Group("", func(contextual *router.Router) {
			contextual.WithMiddleware("organization_context")
			contextual.GET("/organization-context", h.GetActiveContext).Name("organization-context.show")
		})

		auth.GET("/organizations", h.List).Name("organizations.index")
		auth.POST("/organizations", h.Create).Name("organizations.store")
		auth.GET("/organizations/:id", h.Get).Name("organizations.show").WhereNumber("id")
		auth.PATCH("/organizations/:id", h.Update).Name("organizations.update").WhereNumber("id")
		auth.GET("/organizations/:id/members", h.ListMembers).Name("organization-members.index").WhereNumber("id")
		auth.PATCH("/organizations/:id/members/:member_id", h.ChangeMemberRole).
			Name("organization-members.update").
			WhereNumber("id").
			WhereNumber("member_id")
		auth.DELETE("/organizations/:id/members/:member_id", h.RemoveMember).
			Name("organization-members.destroy").
			WhereNumber("id").
			WhereNumber("member_id")
		auth.POST("/organizations/:id/ownership-transfer", h.TransferOwnership).
			Name("organization-ownership.transfer").
			WhereNumber("id")
		auth.GET("/organizations/:id/invitations", h.ListInvitations).Name("organization-invitations.index").WhereNumber("id")
		auth.POST("/organizations/:id/invitations", h.Invite).Name("organization-invitations.store").WhereNumber("id")
		auth.DELETE("/organizations/:id/invitations/:invitation_id", h.RevokeInvitation).
			Name("organization-invitations.destroy").
			WhereNumber("id").
			WhereNumber("invitation_id")
		auth.POST("/organization-invitations/accept", h.AcceptInvitation).Name("organization-invitations.accept")
	})
}

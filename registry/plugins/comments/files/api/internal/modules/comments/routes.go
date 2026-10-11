package comments

import "github.com/fastagi/fastbuild/api/internal/infra/router"

// RegisterRoutes attaches organization-scoped comment resources.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth", "organization_context")
		auth.GET("/comments", h.List).Name("comments.index")
		auth.POST("/comments", h.Create).Name("comments.store")
		auth.PATCH("/comments/:id", h.Update).Name("comments.update").WhereNumber("id")
		auth.DELETE("/comments/:id", h.Delete).Name("comments.destroy").WhereNumber("id")
		auth.GET("/comments/:id/replies", h.ListReplies).Name("comments.replies.index").WhereNumber("id")
	})
}

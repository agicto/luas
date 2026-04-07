package apikey

import "github.com/zgiai/zgo/internal/infra/router"

// RegisterRoutes registers API key management routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")

		auth.GET("/api-keys", h.List).Name("api_keys.index")
		auth.POST("/api-keys", h.Create).Name("api_keys.store")
		auth.DELETE("/api-keys/:id", h.Revoke).Name("api_keys.destroy").WhereNumber("id")
	})
}

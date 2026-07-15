package notification

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches notification-center resources behind authentication.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.GET("/notifications", h.List).Name("notifications.index")
		auth.PATCH("/notifications/:id", h.ReplaceReadState).
			Name("notifications.update").
			WhereNumber("id")
		auth.GET("/notification-status", h.Status).Name("notification-status.show")
		auth.PUT("/notification-read-state", h.MarkReadThrough).Name("notification-read-state.update")
		auth.GET("/notification-preferences", h.GetPreference).Name("notification-preferences.show")
		auth.PUT("/notification-preferences", h.ReplacePreference).Name("notification-preferences.update")
	})
}

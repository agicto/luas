package operator

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches the operator browser session and operator-only routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	signIn := r.POST("/operator/session", h.SignIn).Name("operator.session.store")
	if h.guard != nil {
		if limit := h.guard.LoginIPMiddleware(); limit != nil {
			signIn.Middleware(limit)
		}
	}
	r.DELETE("/operator/session", h.SignOut).Name("operator.session.destroy")

	r.Group("/operator", func(operator *router.Router) {
		operator.Use(h.requireOperator)
		operator.GET("/session", h.Current).Name("operator.session.show")
	})
}

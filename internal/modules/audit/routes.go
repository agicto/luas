package audit

import "github.com/zgiai/zgo/internal/infra/router"

// RegisterRoutes registers the audit log routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.GET("/audit-logs", h.List).Name("audit_logs.index")
	})
}

package user

import (
	"github.com/zgiai/zgo/internal/infra/middleware"
	"github.com/zgiai/zgo/internal/infra/router"
)

// RegisterMiddleware registers the auth middleware group for JWT-protected routes.
func (h *Handler) RegisterMiddleware(r *router.Router) {
	r.MiddlewareGroup("auth", middleware.JWTAuth(h.jwtService))
	r.AliasMiddleware("jwt", middleware.JWTAuth(h.jwtService))
}

// RegisterRoutes registers the user module routes
// It uses the injected handler instance instead of creating a new one
func (h *Handler) RegisterRoutes(r *router.Router) {
	// Public routes
	r.POST("/register", h.Register).Name("auth.register")
	r.POST("/login", h.Login).Name("auth.login")
	r.POST("/password/reset", h.RequestPasswordReset).Name("auth.password.reset.request")
	r.POST("/password/reset/confirm", h.ConfirmPasswordReset).Name("auth.password.reset.confirm")

	// Protected routes
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")

		// Profile
		auth.GET("/users/profile", h.GetProfile).Name("users.profile")
		auth.PUT("/users/profile", h.UpdateProfile).Name("users.profile.update")
		auth.PUT("/users/password", h.ChangePassword).Name("users.password.update")
		auth.DELETE("/users/account", h.DeleteAccount).Name("users.account.delete")
	})
}

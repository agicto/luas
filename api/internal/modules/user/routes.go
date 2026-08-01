package user

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/router"
)

// RegisterMiddleware registers the persistent authentication-session middleware.
func (h *Handler) RegisterMiddleware(r *router.Router) {
	r.MiddlewareGroup("auth", sessionAuth(h.sessions))
}

// RegisterRoutes registers the user module routes
// It uses the injected handler instance instead of creating a new one
func (h *Handler) RegisterRoutes(r *router.Router) {
	// Public routes
	h.protectPublicRoute(r.POST("/register", h.Register).Name("auth.register"), authEndpointRegister)
	h.protectPublicRoute(r.POST("/login", h.Login).Name("auth.login"), authEndpointLogin)
	h.protectPublicRoute(r.POST("/password/reset", h.RequestPasswordReset).Name("auth.password.reset.request"), authEndpointPasswordReset)
	h.protectPublicRoute(r.POST("/password/reset/confirm", h.ConfirmPasswordReset).Name("auth.password.reset.confirm"), authEndpointPasswordResetConfirm)

	if h.browser != nil {
		browserLogin := r.POST("/browser/auth/login", func(c *gin.Context) {
			h.browser.login(c, h.authGuard)
		}).Name("browser.auth.login")
		if h.browser.enabled() {
			h.protectPublicRoute(browserLogin, authEndpointLogin)
		}
		r.GET("/browser/auth/me", h.browser.current).Name("browser.auth.current")
		r.POST("/browser/auth/logout", h.browser.logout).Name("browser.auth.logout")
	}

	// Protected routes
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")

		auth.POST("/logout", h.Logout).Name("auth.logout")

		// Profile
		auth.GET("/users/profile", h.GetProfile).Name("users.profile")
		auth.PUT("/users/profile", h.UpdateProfile).Name("users.profile.update")
		auth.PUT("/users/password", h.ChangePassword).Name("users.password.update")
		auth.DELETE("/users/account", h.DeleteAccount).Name("users.account.delete")
	})
}

func (h *Handler) protectPublicRoute(route *router.Route, endpoint authEndpoint) {
	if middleware := h.authGuard.perIPMiddleware(endpoint); middleware != nil {
		route.Middleware(middleware)
	}
}

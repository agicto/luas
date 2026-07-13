package user

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/ratelimit"
	"github.com/zgiai/luas/api/pkg/response"
)

type authEndpoint string

const (
	authEndpointLogin                authEndpoint = "login"
	authEndpointRegister             authEndpoint = "register"
	authEndpointPasswordReset        authEndpoint = "password_reset"
	authEndpointPasswordResetConfirm authEndpoint = "password_reset_confirm"
)

type authEndpointGuard struct {
	perIP      gin.HandlerFunc
	perSubject ratelimit.Limiter
}

// AuthAbuseGuard applies independent source and subject quotas to public
// authentication operations. Subject keys are normalized and hashed so raw
// account identifiers and reset tokens never become store keys.
type AuthAbuseGuard struct {
	enabled   bool
	endpoints map[authEndpoint]authEndpointGuard
}

// NewAuthAbuseGuard resolves the user starter's route protection from global
// configuration while keeping the policy enforcement owned by this module.
func NewAuthAbuseGuard(cfg *config.Config) *AuthAbuseGuard {
	if cfg == nil {
		return newAuthAbuseGuard(config.AuthenticationRateLimitConfig{})
	}
	return newAuthAbuseGuard(cfg.Middleware.AuthenticationRateLimit)
}

func newAuthAbuseGuard(cfg config.AuthenticationRateLimitConfig) *AuthAbuseGuard {
	guard := &AuthAbuseGuard{
		enabled:   cfg.Enabled,
		endpoints: make(map[authEndpoint]authEndpointGuard),
	}
	if !cfg.Enabled {
		return guard
	}

	guard.endpoints[authEndpointLogin] = buildAuthEndpointGuard(authEndpointLogin, cfg.Login)
	guard.endpoints[authEndpointRegister] = buildAuthEndpointGuard(authEndpointRegister, cfg.Register)
	guard.endpoints[authEndpointPasswordReset] = buildAuthEndpointGuard(authEndpointPasswordReset, cfg.PasswordReset)
	guard.endpoints[authEndpointPasswordResetConfirm] = buildAuthEndpointGuard(authEndpointPasswordResetConfirm, cfg.PasswordResetConfirm)
	return guard
}

func buildAuthEndpointGuard(endpoint authEndpoint, cfg config.AuthenticationEndpointRateLimitConfig) authEndpointGuard {
	guard := authEndpointGuard{}
	if cfg.PerIP.Max > 0 && cfg.PerIP.Window > 0 {
		guard.perIP = ratelimit.Middleware(ratelimit.Config{
			Max:             cfg.PerIP.Max,
			Duration:        cfg.PerIP.Window,
			Store:           ratelimit.NewMemoryStore(cfg.PerIP.Max, cfg.PerIP.Window),
			SuppressHeaders: true,
			KeyFunc: func(c *gin.Context) string {
				return "auth:" + string(endpoint) + ":ip:" + c.ClientIP()
			},
			ErrorHandler: authRateLimitError,
		})
	}
	if cfg.PerSubject.Max > 0 && cfg.PerSubject.Window > 0 {
		guard.perSubject = ratelimit.NewMemoryStore(cfg.PerSubject.Max, cfg.PerSubject.Window)
	}
	return guard
}

func (g *AuthAbuseGuard) perIPMiddleware(endpoint authEndpoint) gin.HandlerFunc {
	if g == nil || !g.enabled {
		return nil
	}
	return g.endpoints[endpoint].perIP
}

func (g *AuthAbuseGuard) allowSubject(c *gin.Context, endpoint authEndpoint, subject string) bool {
	if g == nil || !g.enabled {
		return true
	}

	limiter := g.endpoints[endpoint].perSubject
	if limiter == nil {
		return true
	}

	canonical := strings.TrimSpace(subject)
	if endpoint != authEndpointPasswordResetConfirm {
		canonical = strings.ToLower(canonical)
	}
	key := "auth:" + string(endpoint) + ":subject:" + crypto.SHA256Hex(canonical)
	allowed, _, _ := limiter.Take(c.Request.Context(), key)
	if !allowed {
		authRateLimitError(c, time.Time{})
		return false
	}
	return true
}

func authRateLimitError(c *gin.Context, _ time.Time) {
	response.AbortWithCode(c, http.StatusTooManyRequests, response.ErrorCodeRateLimited, "Too many requests")
}

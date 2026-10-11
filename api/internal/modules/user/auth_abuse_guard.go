package user

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fastagi/fastbuild/api/internal/capabilities/crypto"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/infra/ratelimit"
	"github.com/fastagi/fastbuild/api/pkg/response"
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
	perSubject ratelimit.Store
}

// AuthAbuseGuard applies independent source and subject quotas to public
// authentication operations. Subject keys are normalized and hashed so raw
// account identifiers and reset tokens never become store keys.
type AuthAbuseGuard struct {
	enabled   bool
	endpoints map[authEndpoint]authEndpointGuard
}

// NewAuthAbuseGuard resolves the user starter's route protection from global
// configuration while keeping the policy enforcement owned by this module. The
// stores factory decides whether quotas are process-local or shared across replicas.
func NewAuthAbuseGuard(cfg *config.Config, stores *ratelimit.Stores) *AuthAbuseGuard {
	if cfg == nil {
		return newAuthAbuseGuardWithStores(config.AuthenticationRateLimitConfig{}, stores)
	}
	return newAuthAbuseGuardWithStores(cfg.Middleware.AuthenticationRateLimit, stores)
}

func newAuthAbuseGuard(cfg config.AuthenticationRateLimitConfig) *AuthAbuseGuard {
	return newAuthAbuseGuardWithStores(cfg, nil)
}

func newAuthAbuseGuardWithStores(cfg config.AuthenticationRateLimitConfig, stores *ratelimit.Stores) *AuthAbuseGuard {
	guard := &AuthAbuseGuard{
		enabled:   cfg.Enabled,
		endpoints: make(map[authEndpoint]authEndpointGuard),
	}
	if !cfg.Enabled {
		return guard
	}

	guard.endpoints[authEndpointLogin] = buildAuthEndpointGuard(stores, authEndpointLogin, cfg.Login, cfg.MaxBucketsPerRule)
	guard.endpoints[authEndpointRegister] = buildAuthEndpointGuard(stores, authEndpointRegister, cfg.Register, cfg.MaxBucketsPerRule)
	guard.endpoints[authEndpointPasswordReset] = buildAuthEndpointGuard(stores, authEndpointPasswordReset, cfg.PasswordReset, cfg.MaxBucketsPerRule)
	guard.endpoints[authEndpointPasswordResetConfirm] = buildAuthEndpointGuard(stores, authEndpointPasswordResetConfirm, cfg.PasswordResetConfirm, cfg.MaxBucketsPerRule)
	return guard
}

func buildAuthEndpointGuard(
	stores *ratelimit.Stores,
	endpoint authEndpoint,
	cfg config.AuthenticationEndpointRateLimitConfig,
	maxBuckets int,
) authEndpointGuard {
	guard := authEndpointGuard{}
	if cfg.PerIP.Max > 0 && cfg.PerIP.Window > 0 {
		guard.perIP = ratelimit.Middleware(ratelimit.Config{
			Max:             cfg.PerIP.Max,
			Duration:        cfg.PerIP.Window,
			MaxBuckets:      maxBuckets,
			SuppressHeaders: true,
			Store: stores.New(ratelimit.Rule{
				Name:       "auth_" + string(endpoint) + "_ip",
				Max:        cfg.PerIP.Max,
				Window:     cfg.PerIP.Window,
				MaxBuckets: maxBuckets,
			}),
			KeyFunc: func(c *gin.Context) string {
				return "auth:" + string(endpoint) + ":ip:" + c.ClientIP()
			},
			ErrorHandler: authRateLimitError,
		})
	}
	if cfg.PerSubject.Max > 0 && cfg.PerSubject.Window > 0 {
		guard.perSubject = stores.New(ratelimit.Rule{
			Name:       "auth_" + string(endpoint) + "_subject",
			Max:        cfg.PerSubject.Max,
			Window:     cfg.PerSubject.Window,
			MaxBuckets: maxBuckets,
		})
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

// LoginIPMiddleware returns the per-IP login quota so other sign-in surfaces share the login budget.
// It returns nil when abuse protection is disabled.
func (g *AuthAbuseGuard) LoginIPMiddleware() gin.HandlerFunc {
	return g.perIPMiddleware(authEndpointLogin)
}

// signInThrottle counts failed sign-ins per account. A sign-in takes one attempt before the
// password check and a success resets the account, so only failures accumulate.
type signInThrottle interface {
	Take(ctx context.Context, key string) (allowed bool, remaining int, resetAt time.Time)
	Reset(ctx context.Context, key string)
}

// loginThrottle returns the per-account sign-in failure budget, or nil when it is disabled.
func (g *AuthAbuseGuard) loginThrottle() signInThrottle {
	if g == nil || !g.enabled {
		return nil
	}
	if store, ok := g.endpoints[authEndpointLogin].perSubject.(signInThrottle); ok {
		return store
	}
	return nil
}

func authRateLimitError(c *gin.Context, _ time.Time) {
	response.AbortWithCode(c, http.StatusTooManyRequests, response.ErrorCodeRateLimited, "Too many requests")
}

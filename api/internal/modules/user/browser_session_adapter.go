package user

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/response"
)

const (
	browserSessionCookieName           = "luas_browser_session"
	productionBrowserSessionCookieName = "__Host-luas_browser_session"
)

// BrowserSessionAdapter maps the persistent user session to a same-origin HttpOnly cookie.
type BrowserSessionAdapter struct {
	auth       AuthService
	profile    ProfileService
	sessions   *SessionService
	policy     config.BrowserSessionConfig
	production bool
	now        func() time.Time
}

// BrowserLoginRequest is the minimal login input accepted from browser applications.
type BrowserLoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=6,max=50"`
}

// BrowserUser is the minimum identity view exposed to browser applications.
type BrowserUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewBrowserSessionAdapter creates the optional Go browser-session transport.
func NewBrowserSessionAdapter(
	auth AuthService,
	profile ProfileService,
	sessions *SessionService,
	cfg *config.Config,
) *BrowserSessionAdapter {
	policy := config.BrowserSessionConfig{}
	production := false
	if cfg != nil {
		policy = cfg.BrowserSession
		production = cfg.IsProduction()
	}
	return &BrowserSessionAdapter{
		auth:       auth,
		profile:    profile,
		sessions:   sessions,
		policy:     policy,
		production: production,
		now:        time.Now,
	}
}

func (a *BrowserSessionAdapter) enabled() bool {
	return a != nil && a.policy.Enabled
}

func (a *BrowserSessionAdapter) login(c *gin.Context, authGuard *AuthAbuseGuard) {
	a.privateResponse(c)
	if !a.enabled() {
		a.writeServiceUnavailable(c)
		return
	}
	if !a.allowMutation(c) {
		return
	}

	var req BrowserLoginRequest
	if !handler.BindJSON(c, &req) {
		return
	}
	if authGuard != nil && !authGuard.allowSubject(c, authEndpointLogin, req.Email) {
		return
	}
	if a.auth == nil || a.sessions == nil {
		a.writeServiceUnavailable(c)
		return
	}

	result, err := a.auth.Login(c.Request.Context(), &UserLoginRequest{
		Username: req.Email,
		Password: req.Password,
	})
	if err != nil {
		a.writeAuthError(c, err)
		return
	}
	if result == nil || !validBrowserUser(result.User) || result.TokenType != "Bearer" ||
		!validAuthenticationCredential(result.AccessToken) || result.ExpiresIn <= 0 {
		a.writeServiceUnavailable(c)
		return
	}

	a.setCookie(c, result.AccessToken, result.ExpiresIn)
	response.Success(c, gin.H{"user": browserUser(result.User)})
}

func (a *BrowserSessionAdapter) current(c *gin.Context) {
	a.privateResponse(c)
	if !a.enabled() {
		a.writeServiceUnavailable(c)
		return
	}
	if a.sessions == nil {
		a.writeServiceUnavailable(c)
		return
	}
	credential, err := c.Cookie(a.cookieName())
	if err != nil || !validAuthenticationCredential(credential) {
		a.clearCookie(c)
		a.writeUnauthorized(c)
		return
	}

	identity, err := a.sessions.Authenticate(c.Request.Context(), credential)
	if err != nil || identity == nil {
		if errors.Is(err, domain.ErrServiceUnavailable) {
			a.writeServiceUnavailable(c)
			return
		}
		if errors.Is(err, domain.ErrAccountDisabled) {
			response.ErrorWithCode(c, http.StatusForbidden, domain.CodeAccountDisabled, "Account access is disabled")
			return
		}
		a.clearCookie(c)
		a.writeUnauthorized(c)
		return
	}
	if a.profile == nil {
		a.writeServiceUnavailable(c)
		return
	}

	user, err := a.profile.GetProfile(c.Request.Context(), identity.UserID)
	if err != nil || user == nil {
		if errors.Is(err, domain.ErrServiceUnavailable) {
			a.writeServiceUnavailable(c)
			return
		}
		a.clearCookie(c)
		a.writeUnauthorized(c)
		return
	}
	response.Success(c, gin.H{"user": browserUser(user)})
}

func (a *BrowserSessionAdapter) logout(c *gin.Context) {
	a.privateResponse(c)
	if !a.enabled() {
		a.writeServiceUnavailable(c)
		return
	}
	if !a.allowMutation(c) {
		return
	}

	credential, err := c.Cookie(a.cookieName())
	a.clearCookie(c)
	if err == nil && validAuthenticationCredential(credential) {
		if a.sessions == nil {
			a.writeServiceUnavailable(c)
			return
		}
		if err := a.sessions.Revoke(c.Request.Context(), credential, sessionRevocationLogout); err != nil {
			a.writeServiceUnavailable(c)
			return
		}
	}
	response.Success(c, gin.H{"success": true})
}

func (a *BrowserSessionAdapter) allowMutation(c *gin.Context) bool {
	fetchSite := strings.TrimSpace(c.GetHeader("Sec-Fetch-Site"))
	if fetchSite != "" && fetchSite != "same-origin" && fetchSite != "none" {
		a.writeForbidden(c)
		return false
	}
	if strings.TrimSpace(c.GetHeader("Origin")) != a.policy.Origin {
		a.writeForbidden(c)
		return false
	}
	return true
}

func (a *BrowserSessionAdapter) setCookie(c *gin.Context, credential string, maxAgeSeconds int64) {
	maxAge := int(maxAgeSeconds)
	now := a.currentTime()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     a.cookieName(),
		Value:    credential,
		Path:     "/",
		Expires:  now.Add(time.Duration(maxAge) * time.Second),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.production,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *BrowserSessionAdapter) clearCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     a.cookieName(),
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.production,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *BrowserSessionAdapter) cookieName() string {
	if a != nil && a.production {
		return productionBrowserSessionCookieName
	}
	return browserSessionCookieName
}

func (a *BrowserSessionAdapter) currentTime() time.Time {
	if a != nil && a.now != nil {
		return a.now().UTC()
	}
	return time.Now().UTC()
}

func (a *BrowserSessionAdapter) privateResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	addBrowserVary(c.Writer.Header(), "Cookie", "Origin")
}

func (a *BrowserSessionAdapter) writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		response.ErrorWithCode(c, http.StatusUnauthorized, domain.CodeInvalidCredentials, "Invalid credentials")
	case errors.Is(err, domain.ErrAccountDisabled):
		response.ErrorWithCode(c, http.StatusForbidden, domain.CodeAccountDisabled, "Account access is disabled")
	case errors.Is(err, domain.ErrServiceUnavailable):
		a.writeServiceUnavailable(c)
	default:
		response.ErrorWithCode(c, http.StatusInternalServerError, response.ErrorCodeInternal, "Login failed")
	}
}

func (a *BrowserSessionAdapter) writeUnauthorized(c *gin.Context) {
	response.ErrorWithCode(c, http.StatusUnauthorized, response.ErrorCodeUnauthorized, "Authentication required")
}

func (a *BrowserSessionAdapter) writeForbidden(c *gin.Context) {
	response.ErrorWithCode(c, http.StatusForbidden, response.ErrorCodeForbidden, "Cross-origin mutation is not allowed")
}

func (a *BrowserSessionAdapter) writeServiceUnavailable(c *gin.Context) {
	response.ErrorWithCode(c, http.StatusServiceUnavailable, response.ErrorCodeServiceUnavailable, "Authentication service unavailable")
}

func browserUser(user *domain.User) BrowserUser {
	name := strings.TrimSpace(user.Nickname)
	if name == "" {
		name = user.Username
	}
	return BrowserUser{
		ID:    strconv.FormatUint(uint64(user.ID), 10),
		Email: user.Email,
		Name:  name,
	}
}

func validBrowserUser(user *domain.User) bool {
	if user == nil || user.ID == 0 || strings.TrimSpace(user.Email) == "" {
		return false
	}
	return strings.TrimSpace(user.Nickname) != "" || strings.TrimSpace(user.Username) != ""
}

func addBrowserVary(header http.Header, fields ...string) {
	existing := make(map[string]struct{})
	for _, value := range header.Values("Vary") {
		for _, field := range strings.Split(value, ",") {
			existing[strings.ToLower(strings.TrimSpace(field))] = struct{}{}
		}
	}
	for _, field := range fields {
		if _, ok := existing[strings.ToLower(field)]; ok {
			continue
		}
		header.Add("Vary", field)
		existing[strings.ToLower(field)] = struct{}{}
	}
}

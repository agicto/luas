package operator

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/config"
)

const (
	defaultProductionCookieName  = "__Host-luas_operator"
	defaultDevelopmentCookieName = "luas_operator"
	csrfHeader                   = "X-CSRF-Token"
	csrfContext                  = "luas-operator-csrf/v1"
	maxCredentialLength          = 256
)

// browserSession owns how the operator session credential travels to and from the Admin Console:
// an HttpOnly SameSite=Strict cookie, a session-bound CSRF token, and exact Origin matching.
type browserSession struct {
	cookieName     string
	secure         bool
	allowedOrigins map[string]struct{}
}

func newBrowserSession(cfg *config.Config) *browserSession {
	session := &browserSession{
		cookieName:     defaultDevelopmentCookieName,
		allowedOrigins: make(map[string]struct{}),
	}
	if cfg == nil {
		return session
	}
	session.secure = cfg.IsProduction()
	if session.secure {
		session.cookieName = defaultProductionCookieName
	}
	if name := strings.TrimSpace(cfg.Operator.SessionCookieName); name != "" {
		session.cookieName = name
	}
	allHTTPS := true
	for _, origin := range cfg.Operator.AllowedOrigins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			session.allowedOrigins[trimmed] = struct{}{}
			allHTTPS = allHTTPS && strings.HasPrefix(trimmed, "https://")
		}
	}
	// An HTTPS-only deployment outside production, such as staging, still gets a Secure cookie.
	if len(session.allowedOrigins) > 0 && allHTTPS {
		session.secure = true
	}
	return session
}

// credential reads the session credential from the cookie.
func (s *browserSession) credential(c *gin.Context) (string, bool) {
	cookie, err := c.Request.Cookie(s.cookieName)
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" || len(value) > maxCredentialLength {
		return "", false
	}
	return value, true
}

func (s *browserSession) store(c *gin.Context, credential string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt) / time.Second)
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cookieName,
		Value:    credential,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt.UTC(),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *browserSession) clear(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// originAllowed requires the request Origin to exactly match a configured Admin Console origin.
// foreignOrigin reports a request that names an origin outside the allowed list. Browsers omit
// Origin on same-origin reads, so an absent header is not foreign; a present one must be allowed.
// Without this, a page on another origin trusted by the kernel CORS policy could read operator
// responses with the operator's cookie.
func (s *browserSession) foreignOrigin(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		return false
	}
	_, ok := s.allowedOrigins[origin]
	return !ok
}

func (s *browserSession) originAllowed(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		return false
	}
	_, ok := s.allowedOrigins[origin]
	return ok
}

// csrfToken derives a per-session token from the credential. It needs no storage, changes with every
// session, and cannot be computed by a cross-site page that never sees the HttpOnly credential.
func csrfToken(credential string) string {
	mac := hmac.New(sha256.New, []byte(credential))
	mac.Write([]byte(csrfContext))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func csrfValid(credential, presented string) bool {
	presented = strings.TrimSpace(presented)
	if presented == "" {
		return false
	}
	expected := csrfToken(credential)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) == 1
}

func unsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

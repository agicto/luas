package user

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

const authenticationSessionContextKey = "authentication_session_id"

// sessionAuth resolves an opaque bearer credential through the user starter's session authority.
func sessionAuth(authenticator domain.SessionAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticator == nil {
			response.AbortWithCode(
				c,
				http.StatusServiceUnavailable,
				response.ErrorCodeServiceUnavailable,
				"Authentication service unavailable",
			)
			return
		}

		credential, ok := bearerCredential(c.GetHeader("Authorization"))
		if !ok {
			response.AbortWithCode(
				c,
				http.StatusUnauthorized,
				response.ErrorCodeUnauthorized,
				"Authentication required",
			)
			return
		}

		identity, err := authenticator.Authenticate(c.Request.Context(), credential)
		if err != nil || identity == nil {
			switch {
			case errors.Is(err, domain.ErrServiceUnavailable):
				response.AbortWithCode(
					c,
					http.StatusServiceUnavailable,
					response.ErrorCodeServiceUnavailable,
					"Authentication service unavailable",
				)
			case errors.Is(err, domain.ErrAccountDisabled):
				response.AbortWithCode(
					c,
					http.StatusForbidden,
					domain.CodeAccountDisabled,
					"Account access is disabled",
				)
			default:
				response.AbortWithCode(
					c,
					http.StatusUnauthorized,
					response.ErrorCodeUnauthorized,
					"Authentication required",
				)
			}
			return
		}

		c.Set("userID", identity.UserID)
		c.Set("username", identity.Username)
		c.Set(authenticationSessionContextKey, identity.SessionID)
		c.Next()
	}
}

func authenticationSessionID(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	value, ok := c.Get(authenticationSessionContextKey)
	if !ok {
		return "", false
	}
	sessionID, ok := value.(string)
	return sessionID, ok && sessionID != ""
}

func bearerCredential(header string) (string, bool) {
	header = strings.TrimSpace(header)
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	credential := strings.TrimSpace(parts[1])
	if credential == "" || strings.ContainsAny(credential, " \t\r\n") {
		return "", false
	}
	return credential, true
}

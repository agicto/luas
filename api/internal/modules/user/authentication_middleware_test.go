package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeSessionAuthenticator struct {
	identity   *domain.AuthenticationIdentity
	err        error
	credential string
}

func (f *fakeSessionAuthenticator) Authenticate(
	_ context.Context,
	credential string,
) (*domain.AuthenticationIdentity, error) {
	f.credential = credential
	return f.identity, f.err
}

func TestSessionAuthMapsPublicFailureSemanticsWithoutCredentialDisclosure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const credential = "secret-session-credential"

	tests := []struct {
		name          string
		authenticator domain.SessionAuthenticator
		header        string
		wantStatus    int
		wantCode      string
	}{
		{name: "missing authenticator", header: "Bearer " + credential, wantStatus: 503, wantCode: domain.CodeServiceUnavailable},
		{name: "missing credential", authenticator: &fakeSessionAuthenticator{}, wantStatus: 401, wantCode: "AUTH.UNAUTHORIZED"},
		{name: "invalid credential", authenticator: &fakeSessionAuthenticator{err: domain.ErrAuthenticationRequired}, header: "Bearer " + credential, wantStatus: 401, wantCode: "AUTH.UNAUTHORIZED"},
		{name: "disabled account", authenticator: &fakeSessionAuthenticator{err: domain.ErrAccountDisabled}, header: "Bearer " + credential, wantStatus: 403, wantCode: domain.CodeAccountDisabled},
		{name: "persistence unavailable", authenticator: &fakeSessionAuthenticator{err: domain.ErrServiceUnavailable}, header: "Bearer " + credential, wantStatus: 503, wantCode: domain.CodeServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(sessionAuth(test.authenticator))
			engine.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodGet, "/private", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()

			engine.ServeHTTP(response, request)

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Contains(t, response.Body.String(), `"error_code":"`+test.wantCode+`"`)
			assert.NotContains(t, response.Body.String(), credential)
		})
	}
}

func TestSessionAuthSetsResolvedIdentityAndKeepsSessionIDPrivate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authenticator := &fakeSessionAuthenticator{identity: &domain.AuthenticationIdentity{
		UserID: 7, Username: "ada", SessionID: "private-session-id",
	}}
	engine := gin.New()
	engine.Use(sessionAuth(authenticator))
	engine.GET("/private", func(c *gin.Context) {
		sessionID, ok := authenticationSessionID(c)
		require.True(t, ok)
		assert.Equal(t, "private-session-id", sessionID)
		assert.Equal(t, uint(7), c.MustGet("userID"))
		assert.Equal(t, "ada", c.MustGet("username"))
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "bEaReR credential")
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "credential", authenticator.credential)
	assert.False(t, strings.Contains(response.Body.String(), "private-session-id"))
}

func TestBearerCredentialRejectsAmbiguousHeaders(t *testing.T) {
	for _, value := range []string{
		"", "Bearer", "Basic value", "Bearer first second", "Bearer first\tsecond",
	} {
		_, ok := bearerCredential(value)
		assert.False(t, ok, value)
	}
	_, ok := bearerCredential("Bearer value")
	assert.True(t, ok)
}

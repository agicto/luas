package operator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

const (
	testOrigin     = "https://admin.example.test"
	testCredential = "operator-session-credential-0123456789"
)

type fakeGrants struct {
	operators map[uint]bool
	err       error
}

func (f *fakeGrants) isOperator(_ context.Context, userID uint) (bool, error) {
	return f.operators[userID], f.err
}

func (f *fakeGrants) operatorIDs(_ context.Context, ids []uint) (map[uint]bool, error) {
	result := map[uint]bool{}
	for _, id := range ids {
		if f.operators[id] {
			result[id] = true
		}
	}
	return result, f.err
}

func (f *fakeGrants) insertGrant(_ context.Context, userID uint, now time.Time) (bool, time.Time, error) {
	created := !f.operators[userID]
	f.operators[userID] = true
	return created, now, f.err
}

func (f *fakeGrants) deleteGrant(_ context.Context, userID uint) (bool, error) {
	existed := f.operators[userID]
	delete(f.operators, userID)
	return existed, f.err
}

func (f *fakeGrants) listGrants(context.Context) ([]grantPO, error) { return nil, f.err }

func (f *fakeGrants) ping(context.Context) error { return f.err }

type fakeUsers struct {
	byID map[uint]*domain.User
}

func (f *fakeUsers) FindByID(_ context.Context, id uint) (*domain.User, error) {
	if user, ok := f.byID[id]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUsers) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

// fakeSignIn mirrors the user starter: it verifies credentials, runs authorize, and only then issues.
type fakeSignIn struct {
	users  map[string]*domain.User
	issued int
}

func (f *fakeSignIn) SignIn(
	ctx context.Context,
	identifier string,
	password string,
	authorize func(context.Context, *domain.User) error,
) (*domain.IssuedSession, error) {
	user, ok := f.users[identifier]
	if !ok || password != "secret" {
		return nil, domain.ErrInvalidCredentials
	}
	if err := authorize(ctx, user); err != nil {
		return nil, err
	}
	f.issued++
	return &domain.IssuedSession{Credential: testCredential, ExpiresAt: time.Now().Add(time.Hour), User: user}, nil
}

type fakeAuthenticator struct {
	identity *domain.AuthenticationIdentity
	err      error
}

func (f *fakeAuthenticator) Authenticate(context.Context, string) (*domain.AuthenticationIdentity, error) {
	return f.identity, f.err
}

type fakeRevoker struct{ revoked []string }

func (f *fakeRevoker) RevokeSession(_ context.Context, credential string) error {
	f.revoked = append(f.revoked, credential)
	return nil
}

type fixture struct {
	handler *Handler
	grants  *fakeGrants
	signIn  *fakeSignIn
	auth    *fakeAuthenticator
	revoker *fakeRevoker
	admin   *fakeAdmin
	audit   *fakeAuditQuery
	engine  *gin.Engine
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	operatorUser := &domain.User{ID: 1, Username: "ops", Email: "ops@example.test", Status: 1}
	memberUser := &domain.User{ID: 2, Username: "member", Email: "member@example.test", Status: 1}
	f := &fixture{
		grants:  &fakeGrants{operators: map[uint]bool{1: true}},
		signIn:  &fakeSignIn{users: map[string]*domain.User{"ops": operatorUser, "member": memberUser}},
		auth:    &fakeAuthenticator{identity: &domain.AuthenticationIdentity{UserID: 1, Username: "ops"}},
		revoker: &fakeRevoker{},
		admin:   newFakeAdmin(operatorUser, memberUser),
		audit:   &fakeAuditQuery{},
	}
	svc := &service{
		grants:        f.grants,
		users:         &fakeUsers{byID: map[uint]*domain.User{1: operatorUser, 2: memberUser}},
		signIn:        f.signIn,
		authenticator: f.auth,
		revoker:       f.revoker,
		admin:         f.admin,
		audit:         f.audit,
		now:           time.Now,
	}
	cfg := &config.Config{Operator: config.OperatorConfig{AllowedOrigins: []string{testOrigin}}}
	cfg.App.Env = "production"
	f.handler = NewHandler(svc, nil, cfg, Surfaces{})
	f.engine = gin.New()
	api := f.engine.Group("/v1")
	api.POST("/operator/session", f.handler.SignIn)
	api.DELETE("/operator/session", f.handler.SignOut)
	protected := api.Group("/operator", f.handler.requireOperator)
	protected.GET("/session", f.handler.Current)
	protected.POST("/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	protected.GET("/users", f.handler.ListUsers)
	protected.GET("/users/:id", f.handler.GetUser)
	protected.POST("/users/:id/disable", f.handler.DisableUser)
	protected.POST("/users/:id/enable", f.handler.EnableUser)
	protected.POST("/users/:id/sessions/revoke", f.handler.RevokeUserSessions)
	protected.GET("/audit-logs", f.handler.ListAuditLogs)
	return f
}

type call struct {
	method string
	path   string
	body   string
	origin string
	cookie string
	csrf   string
}

func (f *fixture) do(t *testing.T, request call) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(request.method, request.path, strings.NewReader(request.body))
	if request.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if request.origin != "" {
		req.Header.Set("Origin", request.origin)
	}
	if request.cookie != "" {
		req.AddCookie(&http.Cookie{Name: defaultProductionCookieName, Value: request.cookie})
	}
	if request.csrf != "" {
		req.Header.Set(csrfHeader, request.csrf)
	}
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, req)
	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.ErrorCode
}

func TestSignInIssuesHttpOnlyStrictCookieAndCSRFTokenForOperator(t *testing.T) {
	f := newFixture(t)

	recorder := f.do(t, call{method: http.MethodPost, path: "/v1/operator/session", origin: testOrigin,
		body: `{"identifier":"ops","password":"secret"}`})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	assert.Equal(t, defaultProductionCookieName, cookie.Name)
	assert.Equal(t, testCredential, cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.Empty(t, cookie.Domain)
	assert.NotContains(t, recorder.Body.String(), testCredential, "credential must never reach browser JavaScript")
	assert.Contains(t, recorder.Body.String(), csrfToken(testCredential))
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
}

func TestSignInRejectsNonOperatorBeforeIssuingSession(t *testing.T) {
	f := newFixture(t)

	recorder := f.do(t, call{method: http.MethodPost, path: "/v1/operator/session", origin: testOrigin,
		body: `{"identifier":"member","password":"secret"}`})

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, domain.CodeOperatorForbidden, errorCode(t, recorder))
	assert.Zero(t, f.signIn.issued, "no session may exist for a non-operator")
	assert.Empty(t, recorder.Result().Cookies())
}

func TestSignInFailureCases(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		body     string
		wantCode int
		wantErr  string
	}{
		{name: "foreign origin", origin: "https://evil.example.test", body: `{"identifier":"ops","password":"secret"}`,
			wantCode: http.StatusForbidden, wantErr: domain.CodeOperatorOriginRejected},
		{name: "missing origin", body: `{"identifier":"ops","password":"secret"}`,
			wantCode: http.StatusForbidden, wantErr: domain.CodeOperatorOriginRejected},
		{name: "wrong password", origin: testOrigin, body: `{"identifier":"ops","password":"nope"}`,
			wantCode: http.StatusUnauthorized, wantErr: domain.CodeInvalidCredentials},
		{name: "unknown account", origin: testOrigin, body: `{"identifier":"ghost","password":"secret"}`,
			wantCode: http.StatusUnauthorized, wantErr: domain.CodeInvalidCredentials},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			recorder := f.do(t, call{method: http.MethodPost, path: "/v1/operator/session", origin: test.origin, body: test.body})
			assert.Equal(t, test.wantCode, recorder.Code)
			assert.Equal(t, test.wantErr, errorCode(t, recorder))
			assert.Zero(t, f.signIn.issued)
		})
	}
}

func TestCurrentSessionRequiresValidOperatorSession(t *testing.T) {
	tests := []struct {
		name     string
		cookie   string
		auth     *fakeAuthenticator
		grantErr error
		revoke   bool
		wantCode int
		wantErr  string
	}{
		{name: "no cookie", wantCode: http.StatusUnauthorized, wantErr: "AUTH.UNAUTHORIZED"},
		{name: "revoked session", cookie: testCredential, auth: &fakeAuthenticator{err: domain.ErrAuthenticationRequired},
			wantCode: http.StatusUnauthorized, wantErr: "AUTH.UNAUTHORIZED"},
		{name: "disabled account", cookie: testCredential, auth: &fakeAuthenticator{err: domain.ErrAccountDisabled},
			wantCode: http.StatusForbidden, wantErr: domain.CodeAccountDisabled},
		{name: "grant revoked mid-session", cookie: testCredential, revoke: true,
			wantCode: http.StatusForbidden, wantErr: domain.CodeOperatorForbidden},
		{name: "database unavailable", cookie: testCredential, auth: &fakeAuthenticator{err: domain.ErrServiceUnavailable},
			wantCode: http.StatusServiceUnavailable, wantErr: domain.CodeServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			if test.auth != nil {
				*f.auth = *test.auth
			}
			if test.revoke {
				delete(f.grants.operators, 1)
			}
			recorder := f.do(t, call{method: http.MethodGet, path: "/v1/operator/session", cookie: test.cookie})
			assert.Equal(t, test.wantCode, recorder.Code)
			assert.Equal(t, test.wantErr, errorCode(t, recorder))
		})
	}

	t.Run("valid session", func(t *testing.T) {
		f := newFixture(t)
		recorder := f.do(t, call{method: http.MethodGet, path: "/v1/operator/session", cookie: testCredential})
		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"username":"ops"`)
		assert.Contains(t, recorder.Body.String(), csrfToken(testCredential))
	})
}

func TestUnsafeOperatorRequestsRequireOriginAndCSRF(t *testing.T) {
	valid := csrfToken(testCredential)
	tests := []struct {
		name     string
		origin   string
		csrf     string
		auth     *fakeAuthenticator
		wantCode int
		wantErr  string
	}{
		{name: "missing csrf", origin: testOrigin, wantCode: http.StatusForbidden, wantErr: domain.CodeOperatorCSRFRejected},
		{name: "csrf from another session", origin: testOrigin, csrf: csrfToken("another-credential"),
			wantCode: http.StatusForbidden, wantErr: domain.CodeOperatorCSRFRejected},
		{name: "foreign origin", origin: "https://evil.example.test", csrf: valid,
			wantCode: http.StatusForbidden, wantErr: domain.CodeOperatorOriginRejected},
		{name: "revoked session wins over bad csrf", origin: testOrigin,
			auth: &fakeAuthenticator{err: domain.ErrAuthenticationRequired}, wantCode: http.StatusUnauthorized, wantErr: "AUTH.UNAUTHORIZED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			if test.auth != nil {
				*f.auth = *test.auth
			}
			recorder := f.do(t, call{method: http.MethodPost, path: "/v1/operator/probe", cookie: testCredential,
				origin: test.origin, csrf: test.csrf})
			assert.Equal(t, test.wantCode, recorder.Code)
			assert.Equal(t, test.wantErr, errorCode(t, recorder))
		})
	}

	t.Run("allowed", func(t *testing.T) {
		f := newFixture(t)
		recorder := f.do(t, call{method: http.MethodPost, path: "/v1/operator/probe", cookie: testCredential,
			origin: testOrigin, csrf: valid})
		assert.Equal(t, http.StatusNoContent, recorder.Code)
	})
}

func TestSignOutRevokesSessionAndExpiresCookie(t *testing.T) {
	t.Run("revokes with valid csrf", func(t *testing.T) {
		f := newFixture(t)
		recorder := f.do(t, call{method: http.MethodDelete, path: "/v1/operator/session", cookie: testCredential,
			origin: testOrigin, csrf: csrfToken(testCredential)})
		require.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Equal(t, []string{testCredential}, f.revoker.revoked)
		cookies := recorder.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Negative(t, cookies[0].MaxAge)
	})

	t.Run("without cookie is idempotent", func(t *testing.T) {
		f := newFixture(t)
		recorder := f.do(t, call{method: http.MethodDelete, path: "/v1/operator/session"})
		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, f.revoker.revoked)
	})

	t.Run("rejects missing csrf", func(t *testing.T) {
		f := newFixture(t)
		recorder := f.do(t, call{method: http.MethodDelete, path: "/v1/operator/session", cookie: testCredential, origin: testOrigin})
		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Equal(t, domain.CodeOperatorCSRFRejected, errorCode(t, recorder))
		assert.Empty(t, f.revoker.revoked)
	})
}

func TestCSRFTokenIsBoundToSession(t *testing.T) {
	first := csrfToken("credential-a")
	assert.Equal(t, first, csrfToken("credential-a"))
	assert.NotEqual(t, first, csrfToken("credential-b"))
	assert.True(t, csrfValid("credential-a", first))
	assert.False(t, csrfValid("credential-b", first))
	assert.False(t, csrfValid("credential-a", ""))
}

func TestBrowserSessionCookieNameFollowsEnvironment(t *testing.T) {
	development := newBrowserSession(&config.Config{})
	assert.Equal(t, defaultDevelopmentCookieName, development.cookieName)
	assert.False(t, development.secure)

	custom := &config.Config{Operator: config.OperatorConfig{SessionCookieName: "admin_sid"}}
	assert.Equal(t, "admin_sid", newBrowserSession(custom).cookieName)
}

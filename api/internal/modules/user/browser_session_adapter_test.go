package user

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/pkg/response"
)

type browserAuthService struct {
	loginCalls int
	result     *UserLoginResponse
	err        error
}

func (s *browserAuthService) Register(context.Context, *UserRegisterRequest) (*domain.User, error) {
	return nil, errors.New("registration is not exposed by browser session adapter")
}

func (s *browserAuthService) Login(context.Context, *UserLoginRequest) (*UserLoginResponse, error) {
	s.loginCalls++
	return s.result, s.err
}

func (s *browserAuthService) RequestPasswordReset(context.Context, *UserPasswordResetRequest) error {
	return nil
}

func (s *browserAuthService) ConfirmPasswordReset(context.Context, *UserPasswordResetConfirmRequest) error {
	return nil
}

type browserProfileService struct {
	user *domain.User
	err  error
}

func (s *browserProfileService) GetProfile(context.Context, uint) (*domain.User, error) {
	return s.user, s.err
}

func (s *browserProfileService) UpdateProfile(context.Context, uint, *UserUpdateRequest) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (s *browserProfileService) ChangePassword(context.Context, uint, *UserChangePasswordRequest) error {
	return errors.New("not implemented")
}

func (s *browserProfileService) DeleteAccount(context.Context, uint) error {
	return errors.New("not implemented")
}

func TestBrowserSessionAdapterLoginCurrentLogoutLifecycle(t *testing.T) {
	sessions, _, user := newAuthenticationSessionFixture(t)
	issued, err := sessions.Issue(context.Background(), user)
	require.NoError(t, err)
	auth := &browserAuthService{result: &UserLoginResponse{
		AccessToken: issued.AccessToken,
		TokenType:   issued.TokenType,
		ExpiresIn:   issued.ExpiresIn,
		User:        user,
	}}
	profile := &browserProfileService{user: user}
	engine := browserSessionTestEngine(auth, profile, sessions, true)

	login := browserMutationRequest("/v1/browser/auth/login", `{"email":"alice@example.com","password":"password123"}`, browserTestOrigin)
	loginResponse := httptest.NewRecorder()
	engine.ServeHTTP(loginResponse, login)
	require.Equal(t, http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	assert.Equal(t, "private, no-store", loginResponse.Header().Get("Cache-Control"))
	assert.ElementsMatch(t, []string{"Accept-Encoding", "Cookie", "Origin"}, loginResponse.Header().Values("Vary"))
	assert.NotContains(t, loginResponse.Body.String(), issued.AccessToken)

	cookies := loginResponse.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	assert.Equal(t, browserSessionCookieName, cookie.Name)
	assert.True(t, cookie.HttpOnly)
	assert.False(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)

	current := httptest.NewRequest(http.MethodGet, "/v1/browser/auth/me", nil)
	current.AddCookie(cookie)
	currentResponse := httptest.NewRecorder()
	engine.ServeHTTP(currentResponse, current)
	require.Equal(t, http.StatusOK, currentResponse.Code, currentResponse.Body.String())
	assert.Contains(t, currentResponse.Body.String(), `"email":"alice@example.com"`)
	assert.NotContains(t, currentResponse.Body.String(), issued.AccessToken)

	logout := browserMutationRequest("/v1/browser/auth/logout", "", browserTestOrigin)
	logout.AddCookie(cookie)
	logoutResponse := httptest.NewRecorder()
	engine.ServeHTTP(logoutResponse, logout)
	require.Equal(t, http.StatusOK, logoutResponse.Code, logoutResponse.Body.String())
	require.Len(t, logoutResponse.Result().Cookies(), 1)
	assert.Less(t, logoutResponse.Result().Cookies()[0].MaxAge, 0)

	follow := httptest.NewRequest(http.MethodGet, "/v1/browser/auth/me", nil)
	follow.AddCookie(cookie)
	followResponse := httptest.NewRecorder()
	engine.ServeHTTP(followResponse, follow)
	assert.Equal(t, http.StatusUnauthorized, followResponse.Code)
	assertErrorCode(t, followResponse, response.ErrorCodeUnauthorized)
}

func TestBrowserSessionAdapterRejectsUntrustedMutationBeforeLogin(t *testing.T) {
	auth := &browserAuthService{}
	engine := browserSessionTestEngine(auth, &browserProfileService{}, nil, true)

	for _, origin := range []string{"", "https://evil.example.com"} {
		request := browserMutationRequest("/v1/browser/auth/login", `{"email":"alice@example.com","password":"password123"}`, origin)
		result := httptest.NewRecorder()
		engine.ServeHTTP(result, request)
		assert.Equal(t, http.StatusForbidden, result.Code)
		assertErrorCode(t, result, response.ErrorCodeForbidden)
		assert.Empty(t, result.Header().Values("Set-Cookie"))
	}
	assert.Zero(t, auth.loginCalls)
}

func TestBrowserSessionAdapterMapsInvalidCredentialsWithoutCookie(t *testing.T) {
	auth := &browserAuthService{err: domain.ErrInvalidCredentials}
	engine := browserSessionTestEngine(auth, &browserProfileService{}, &SessionService{}, true)
	request := browserMutationRequest("/v1/browser/auth/login", `{"email":"alice@example.com","password":"wrong-password"}`, browserTestOrigin)
	result := httptest.NewRecorder()
	engine.ServeHTTP(result, request)

	assert.Equal(t, http.StatusUnauthorized, result.Code)
	assertErrorCode(t, result, domain.CodeInvalidCredentials)
	assert.Empty(t, result.Header().Values("Set-Cookie"))
}

func TestBrowserSessionAdapterLogoutWithoutCookieIsIdempotent(t *testing.T) {
	engine := browserSessionTestEngine(&browserAuthService{}, &browserProfileService{}, nil, true)
	request := browserMutationRequest("/v1/browser/auth/logout", "", browserTestOrigin)
	result := httptest.NewRecorder()
	engine.ServeHTTP(result, request)

	require.Equal(t, http.StatusOK, result.Code, result.Body.String())
	assert.Contains(t, result.Body.String(), `"success":true`)
	require.Len(t, result.Result().Cookies(), 1)
	assert.Less(t, result.Result().Cookies()[0].MaxAge, 0)
}

func TestBrowserSessionAdapterRoutesFailClosedByDefault(t *testing.T) {
	auth := &browserAuthService{}
	engine := browserSessionTestEngine(auth, &browserProfileService{}, nil, false)
	for range 2 {
		request := browserMutationRequest("/v1/browser/auth/login", `{"email":"alice@example.com","password":"password123"}`, browserTestOrigin)
		result := httptest.NewRecorder()
		engine.ServeHTTP(result, request)
		assert.Equal(t, http.StatusServiceUnavailable, result.Code)
		assertErrorCode(t, result, response.ErrorCodeServiceUnavailable)
	}
	assert.Zero(t, auth.loginCalls)
}

func TestBrowserSessionAdapterFailsClosedForMissingSessionAuthority(t *testing.T) {
	engine := browserSessionTestEngine(&browserAuthService{}, &browserProfileService{}, nil, true)
	request := httptest.NewRequest(http.MethodGet, "/v1/browser/auth/me", nil)
	result := httptest.NewRecorder()
	engine.ServeHTTP(result, request)
	assert.Equal(t, http.StatusServiceUnavailable, result.Code)
	assertErrorCode(t, result, response.ErrorCodeServiceUnavailable)
}

func TestBrowserSessionAdapterProductionCookieUsesHostPrefix(t *testing.T) {
	adapter := &BrowserSessionAdapter{production: true, now: func() time.Time {
		return time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	}}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	adapter.setCookie(ginContext, strings.Repeat("a", 43), 60)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, productionBrowserSessionCookieName, cookies[0].Name)
	assert.True(t, cookies[0].Secure)
	assert.True(t, cookies[0].HttpOnly)
	assert.Empty(t, cookies[0].Domain)
}

const browserTestOrigin = "http://127.0.0.1:4173"

func browserSessionTestEngine(
	auth AuthService,
	profile ProfileService,
	sessions *SessionService,
	enabled bool,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	adapter := NewBrowserSessionAdapter(auth, profile, sessions, &config.Config{
		BrowserSession: config.BrowserSessionConfig{Enabled: enabled, Origin: browserTestOrigin},
	})
	h := NewHandler(auth, profile, nil, sessions, nil, newAuthAbuseGuard(config.AuthenticationRateLimitConfig{}), adapter)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Header("Vary", "Accept-Encoding")
		c.Next()
	})
	routes := router.New(engine).Prefix("/v1")
	h.RegisterMiddleware(routes)
	h.RegisterRoutes(routes)
	return engine
}

func browserMutationRequest(path, body, origin string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	return request
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	var failure response.ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &failure))
	assert.Equal(t, want, failure.ErrorCode)
}

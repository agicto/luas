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
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/router"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestAuthenticationSessionIsOpaqueHashOnlyAndImmediatelyRevocable(t *testing.T) {
	service, db, user := newAuthenticationSessionFixture(t)

	issued, err := service.Issue(context.Background(), user)
	require.NoError(t, err)
	require.NotEmpty(t, issued.AccessToken)
	assert.Equal(t, "Bearer", issued.TokenType)
	assert.Positive(t, issued.ExpiresIn)
	assert.NotContains(t, issued.AccessToken, ".")

	var stored AuthenticationSessionPO
	require.NoError(t, db.First(&stored).Error)
	assert.NotEqual(t, issued.AccessToken, stored.TokenHash)
	assert.Equal(t, crypto.SHA256Hex(issued.AccessToken), stored.TokenHash)
	assert.NotContains(t, strings.Join([]string{
		stored.ID,
		stored.TokenHash,
		stored.RevocationReason,
	}, "\x00"), issued.AccessToken)

	identity, err := service.Authenticate(context.Background(), issued.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, identity.UserID)
	assert.Equal(t, user.Username, identity.Username)
	assert.Equal(t, stored.ID, identity.SessionID)

	require.NoError(t, service.Revoke(context.Background(), issued.AccessToken, "logout"))
	_, err = service.Authenticate(context.Background(), issued.AccessToken)
	assert.ErrorIs(t, err, domain.ErrAuthenticationRequired)
}

func TestAuthenticationSessionEnforcesIdleAbsoluteAndAccountState(t *testing.T) {
	service, db, user := newAuthenticationSessionFixture(t)
	base := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return base }

	idleSession, err := service.Issue(context.Background(), user)
	require.NoError(t, err)
	service.now = func() time.Time { return base.Add(31 * time.Minute) }
	_, err = service.Authenticate(context.Background(), idleSession.AccessToken)
	assert.ErrorIs(t, err, domain.ErrAuthenticationRequired)

	service.now = func() time.Time { return base }
	absoluteSession, err := service.Issue(context.Background(), user)
	require.NoError(t, err)
	service.now = func() time.Time { return base.Add(2*time.Hour + time.Second) }
	_, err = service.Authenticate(context.Background(), absoluteSession.AccessToken)
	assert.ErrorIs(t, err, domain.ErrAuthenticationRequired)

	service.now = func() time.Time { return base }
	disabledSession, err := service.Issue(context.Background(), user)
	require.NoError(t, err)
	require.NoError(t, db.Model(&UserPO{}).Where("id = ?", user.ID).Update("status", 0).Error)
	_, err = service.Authenticate(context.Background(), disabledSession.AccessToken)
	assert.ErrorIs(t, err, domain.ErrAccountDisabled)
}

func TestAuthenticationSessionTouchIsWriteThrottled(t *testing.T) {
	service, db, user := newAuthenticationSessionFixture(t)
	base := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return base }

	issued, err := service.Issue(context.Background(), user)
	require.NoError(t, err)

	service.now = func() time.Time { return base.Add(4 * time.Minute) }
	_, err = service.Authenticate(context.Background(), issued.AccessToken)
	require.NoError(t, err)

	var before AuthenticationSessionPO
	require.NoError(t, db.First(&before).Error)
	assert.True(t, base.Equal(before.LastSeenAt))

	service.now = func() time.Time { return base.Add(6 * time.Minute) }
	_, err = service.Authenticate(context.Background(), issued.AccessToken)
	require.NoError(t, err)

	var after AuthenticationSessionPO
	require.NoError(t, db.First(&after).Error)
	assert.True(t, base.Add(6*time.Minute).Equal(after.LastSeenAt))
	assert.True(t, base.Add(36*time.Minute).Equal(after.IdleExpiresAt))
}

func TestAuthenticationSessionLogoutRouteRevokesPresentedCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sessions, _, user := newAuthenticationSessionFixture(t)
	issued, err := sessions.Issue(context.Background(), user)
	require.NoError(t, err)

	handler := NewHandler(nil, nil, nil, sessions, nil, newAuthAbuseGuard(config.AuthenticationRateLimitConfig{}))
	engine := gin.New()
	routes := router.New(engine).Prefix("/v1")
	handler.RegisterMiddleware(routes)
	handler.RegisterRoutes(routes)

	logout := httptest.NewRequest(http.MethodPost, "/v1/logout", nil)
	logout.Header.Set("Authorization", "Bearer "+issued.AccessToken)
	logoutResponse := httptest.NewRecorder()
	engine.ServeHTTP(logoutResponse, logout)
	require.Equal(t, http.StatusOK, logoutResponse.Code, logoutResponse.Body.String())

	profile := httptest.NewRequest(http.MethodGet, "/v1/users/profile", nil)
	profile.Header.Set("Authorization", "Bearer "+issued.AccessToken)
	profileResponse := httptest.NewRecorder()
	engine.ServeHTTP(profileResponse, profile)
	require.Equal(t, http.StatusUnauthorized, profileResponse.Code, profileResponse.Body.String())
	var failure response.ErrorResponse
	require.NoError(t, json.Unmarshal(profileResponse.Body.Bytes(), &failure))
	assert.Equal(t, response.ErrorCodeUnauthorized, failure.ErrorCode)
}

func TestPasswordSecurityEventsRevokeExistingAuthenticationSessions(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *service, *repository, *domain.User) error
	}{
		{
			name: "password change",
			run: func(ctx context.Context, svc *service, _ *repository, user *domain.User) error {
				return svc.ChangePassword(ctx, user.ID, &UserChangePasswordRequest{
					OldPassword: "password123",
					NewPassword: "new-password-123",
				})
			},
		},
		{
			name: "password reset",
			run: func(ctx context.Context, svc *service, repo *repository, user *domain.User) error {
				const token = "reset-token-with-enough-entropy"
				if err := repo.StorePasswordResetToken(
					ctx,
					user.ID,
					crypto.SHA256Hex(token),
					time.Now().Add(time.Hour),
				); err != nil {
					return err
				}
				return svc.ConfirmPasswordReset(ctx, &UserPasswordResetConfirmRequest{
					Token:       token,
					NewPassword: "new-password-123",
				})
			},
		},
		{
			name: "account deletion",
			run: func(ctx context.Context, svc *service, _ *repository, user *domain.User) error {
				return svc.DeleteAccount(ctx, user.ID)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sessions, db, user := newAuthenticationSessionFixture(t)
			repo := NewRepository(db)
			svc := NewService(
				repo,
				repo,
				sessions,
				events.NewEventBus(),
				&fakeUserMailer{},
				NewAccountDeletionPolicy(),
			)
			issued, err := sessions.Issue(context.Background(), user)
			require.NoError(t, err)

			require.NoError(t, test.run(context.Background(), svc, repo, user))
			_, err = sessions.Authenticate(context.Background(), issued.AccessToken)
			assert.ErrorIs(t, err, domain.ErrAuthenticationRequired)
		})
	}
}

func newAuthenticationSessionFixture(t *testing.T) (*SessionService, *gorm.DB, *domain.User) {
	t.Helper()

	db := testplatform.OpenPostgres(
		t,
		nil,
		&UserPO{},
		&PasswordResetTokenPO{},
		&AuthenticationSessionPO{},
	)

	passwordHash, err := crypto.HashPassword("password123")
	require.NoError(t, err)
	userPO := &UserPO{
		Username: "alice",
		Email:    "alice@example.com",
		Password: passwordHash,
		Status:   1,
	}
	require.NoError(t, db.Create(userPO).Error)

	cfg := &config.Config{
		Authentication: config.AuthenticationConfig{
			SessionTTL:           2 * time.Hour,
			SessionIdleTimeout:   30 * time.Minute,
			SessionTouchInterval: 5 * time.Minute,
			SessionRetention:     24 * time.Hour,
		},
	}
	service := NewSessionService(NewRepository(db), cfg)
	return service, db, userPO.toDomain()
}

func TestAuthenticationSessionFailsClosedWithoutPersistence(t *testing.T) {
	service := NewSessionService(NewRepository(nil), &config.Config{
		Authentication: config.AuthenticationConfig{
			SessionTTL:           time.Hour,
			SessionIdleTimeout:   30 * time.Minute,
			SessionTouchInterval: 5 * time.Minute,
			SessionRetention:     time.Hour,
		},
	})

	_, err := service.Authenticate(context.Background(), strings.Repeat("x", 43))
	if !errors.Is(err, domain.ErrServiceUnavailable) {
		t.Fatalf("Authenticate() error = %v, want service unavailable", err)
	}
}

func TestAuthenticationSessionPruneHonorsRetentionAndBatch(t *testing.T) {
	service, db, user := newAuthenticationSessionFixture(t)
	base := time.Date(2026, 7, 15, 22, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return base }

	for range 3 {
		_, err := service.Issue(context.Background(), user)
		require.NoError(t, err)
	}
	var sessions []AuthenticationSessionPO
	require.NoError(t, db.Order("id ASC").Find(&sessions).Error)
	require.Len(t, sessions, 3)
	old := base.Add(-25 * time.Hour)
	recent := base.Add(-time.Hour)
	require.NoError(t, db.Model(&AuthenticationSessionPO{}).
		Where("id = ?", sessions[0].ID).
		Updates(map[string]any{"revoked_at": old, "updated_at": old}).Error)
	require.NoError(t, db.Model(&AuthenticationSessionPO{}).
		Where("id = ?", sessions[1].ID).
		Updates(map[string]any{"revoked_at": recent, "updated_at": recent}).Error)
	require.NoError(t, db.Model(&AuthenticationSessionPO{}).
		Where("id = ?", sessions[2].ID).
		Updates(map[string]any{
			"expires_at":      old,
			"idle_expires_at": old,
			"updated_at":      old,
		}).Error)

	count, err := service.PruneAuthenticationSessions(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	count, err = service.PruneAuthenticationSessions(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	count, err = service.PruneAuthenticationSessions(context.Background(), 1)
	require.NoError(t, err)
	assert.Zero(t, count)

	var remaining []AuthenticationSessionPO
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, sessions[1].ID, remaining[0].ID)
}

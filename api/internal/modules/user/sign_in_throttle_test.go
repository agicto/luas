package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

func throttledTestService(t *testing.T, failures int, users map[string]*domain.User) *service {
	t.Helper()
	svc := newTestService(&fakeRepo{
		findByLoginFn: func(_ context.Context, identifier string) (*domain.User, error) {
			if user, ok := users[identifier]; ok {
				copied := *user
				return &copied, nil
			}
			return nil, gorm.ErrRecordNotFound
		},
	})
	svc.signIns = newAuthAbuseGuard(config.AuthenticationRateLimitConfig{
		Enabled: true,
		Login: config.AuthenticationEndpointRateLimitConfig{
			PerSubject: config.RateLimitRuleConfig{Max: failures, Window: time.Minute},
		},
	}).loginThrottle()
	if svc.signIns == nil {
		t.Fatal("login throttle is not configured")
	}
	return svc
}

func login(svc *service, identifier, password string) error {
	_, err := svc.Login(context.Background(), &UserLoginRequest{Username: identifier, Password: password})
	return err
}

func TestSignInThrottleCountsOnlyFailures(t *testing.T) {
	alice := &domain.User{ID: 7, Username: "alice", Email: "alice@example.com", Password: mustHashTestPassword(t), Status: 1}
	svc := throttledTestService(t, 2, map[string]*domain.User{"alice": alice})

	for range 5 {
		assert.NoError(t, login(svc, "alice", "password123"), "successful sign-ins must not spend the budget")
	}
	assert.ErrorIs(t, login(svc, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.NoError(t, login(svc, "alice", "password123"), "a success clears earlier failures")
	assert.ErrorIs(t, login(svc, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(svc, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(svc, "alice", "password123"), domain.ErrSignInThrottled,
		"an exhausted account is throttled even with the right password")
}

func TestSignInThrottleSharesOneBudgetAcrossAnAccountsIdentifiers(t *testing.T) {
	alice := &domain.User{ID: 7, Username: "alice", Email: "alice@example.com", Password: mustHashTestPassword(t), Status: 1}
	svc := throttledTestService(t, 2, map[string]*domain.User{"alice": alice, "alice@example.com": alice})

	assert.ErrorIs(t, login(svc, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(svc, "alice@example.com", "wrong-password"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(svc, "alice", "wrong-password"), domain.ErrSignInThrottled)
}

func TestSignInThrottleTreatsUnknownIdentifiersLikeAccounts(t *testing.T) {
	svc := throttledTestService(t, 1, nil)

	assert.ErrorIs(t, login(svc, "Ghost@Example.com", "guess"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(svc, " ghost@example.com ", "guess"), domain.ErrSignInThrottled,
		"unknown identifiers are throttled like accounts, so a 429 reveals nothing")
	assert.ErrorIs(t, login(svc, "other@example.com", "guess"), domain.ErrInvalidCredentials)
}

package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/database/migrations"
	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/infra/ratelimit"
	testplatform "github.com/fastagi/fastbuild/api/internal/infra/testing"
)

// TestSignInThrottleSharesFailuresAcrossReplicas proves the per-account failure budget and its
// reset-on-success use the shared store when RATE_LIMIT_STORE=postgres.
func TestSignInThrottleSharesFailuresAcrossReplicas(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil)
	require.NoError(t, migrations.All()[migrations.RateLimitBucketsMigration].Up(db))
	sqlDB, err := db.DB()
	require.NoError(t, err)

	alice := &domain.User{ID: 7, Username: "alice", Email: "alice@example.com", Password: mustHashTestPassword(t), Status: 1}
	replica := func() *service {
		stores, storesErr := ratelimit.NewPostgresStores(sqlDB, ratelimit.PostgresOptions{
			Timeout: 5 * time.Second,
			Failure: ratelimit.FailureClosed,
		})
		require.NoError(t, storesErr)
		svc := newTestService(&fakeRepo{
			findByLoginFn: func(_ context.Context, identifier string) (*domain.User, error) {
				if identifier == "alice" {
					copied := *alice
					return &copied, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
		})
		svc.signIns = newAuthAbuseGuardWithStores(config.AuthenticationRateLimitConfig{
			Enabled: true,
			Login: config.AuthenticationEndpointRateLimitConfig{
				PerSubject: config.RateLimitRuleConfig{Max: 2, Window: time.Minute},
			},
		}, stores).loginThrottle()
		require.NotNil(t, svc.signIns)
		return svc
	}
	first, second := replica(), replica()

	assert.ErrorIs(t, login(first, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.NoError(t, login(second, "alice", "password123"), "a success on any replica clears the shared failures")
	assert.ErrorIs(t, login(first, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(second, "alice", "wrong-password"), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, login(first, "alice", "password123"), domain.ErrSignInThrottled,
		"failures on both replicas exhaust one shared account budget")
}

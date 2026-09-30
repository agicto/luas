package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
)

func TestUserAdministrationPostgres(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &UserPO{}, &AuthenticationSessionPO{})
	ctx := context.Background()
	repo := NewRepository(db)
	svc := &service{repo: repo, admin: repo}

	seed := []UserPO{
		{Username: "alice", Email: "alice@example.test", Password: "x", Status: userStatusActive},
		{Username: "Bob", Email: "bob@EXAMPLE.test", Password: "x", Status: userStatusDisabled},
		{Username: "carol_100%", Email: "carol@example.test", Password: "x", Status: userStatusActive},
	}
	for index := range seed {
		require.NoError(t, db.Create(&seed[index]).Error)
	}
	// The status column defaults to active, so a zero value on insert is replaced; persist it explicitly.
	require.NoError(t, db.Model(&UserPO{}).Where("id = ?", seed[1].ID).Update("status", userStatusDisabled).Error)
	alice := seed[0]
	for index := range 2 {
		require.NoError(t, db.Create(&AuthenticationSessionPO{
			ID:            fmt.Sprintf("session-%d", index),
			UserID:        alice.ID,
			TokenHash:     fmt.Sprintf("%064d", index),
			ExpiresAt:     time.Now().Add(time.Hour),
			IdleExpiresAt: time.Now().Add(time.Hour),
			LastSeenAt:    time.Now(),
		}).Error)
	}

	t.Run("lists newest first with a bounded page", func(t *testing.T) {
		users, total, err := svc.ListUsers(ctx, domain.UserListFilter{}, 1, 2)
		require.NoError(t, err)
		assert.EqualValues(t, 3, total)
		require.Len(t, users, 2)
		assert.Equal(t, "carol_100%", users[0].Username)
		assert.Equal(t, "Bob", users[1].Username)
	})

	t.Run("matches username or email case-insensitively", func(t *testing.T) {
		users, total, err := svc.ListUsers(ctx, domain.UserListFilter{Query: "example.TEST"}, 1, 20)
		require.NoError(t, err)
		assert.EqualValues(t, 3, total)
		assert.Len(t, users, 3)

		users, _, err = svc.ListUsers(ctx, domain.UserListFilter{Query: "BOB"}, 1, 20)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "Bob", users[0].Username)
	})

	t.Run("treats LIKE wildcards as literals", func(t *testing.T) {
		users, _, err := svc.ListUsers(ctx, domain.UserListFilter{Query: "%"}, 1, 20)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "carol_100%", users[0].Username)

		users, _, err = svc.ListUsers(ctx, domain.UserListFilter{Query: "l_1"}, 1, 20)
		require.NoError(t, err)
		require.Len(t, users, 1, "underscore must not match any character")
	})

	t.Run("filters by status", func(t *testing.T) {
		active := false
		users, total, err := svc.ListUsers(ctx, domain.UserListFilter{Active: &active}, 1, 20)
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		assert.Equal(t, "Bob", users[0].Username)
	})

	t.Run("rejects unbounded input", func(t *testing.T) {
		_, _, err := svc.ListUsers(ctx, domain.UserListFilter{}, 1, 101)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("disabling revokes every session atomically and repeats safely", func(t *testing.T) {
		user, err := svc.SetUserActive(ctx, alice.ID, false)
		require.NoError(t, err)
		assert.False(t, user.IsActive())

		var active int64
		require.NoError(t, db.Model(&AuthenticationSessionPO{}).
			Where("user_id = ? AND revoked_at IS NULL", alice.ID).Count(&active).Error)
		assert.Zero(t, active)
		var reasons []string
		require.NoError(t, db.Model(&AuthenticationSessionPO{}).
			Where("user_id = ?", alice.ID).Pluck("revocation_reason", &reasons).Error)
		assert.Equal(t, []string{"account_disabled", "account_disabled"}, reasons)

		user, err = svc.SetUserActive(ctx, alice.ID, false)
		require.NoError(t, err)
		assert.False(t, user.IsActive())

		user, err = svc.SetUserActive(ctx, alice.ID, true)
		require.NoError(t, err)
		assert.True(t, user.IsActive())
		require.NoError(t, db.Model(&AuthenticationSessionPO{}).
			Where("user_id = ? AND revoked_at IS NULL", alice.ID).Count(&active).Error)
		assert.Zero(t, active, "enabling does not restore sessions")
	})

	t.Run("unknown accounts are not found", func(t *testing.T) {
		_, err := svc.SetUserActive(ctx, 999999, false)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
		require.ErrorIs(t, svc.RevokeUserSessions(ctx, 999999), domain.ErrUserNotFound)
	})
}

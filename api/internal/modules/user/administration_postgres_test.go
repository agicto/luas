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

// A sign-in or profile update that read the account before a password change, disable, or deletion
// committed must not write the old values back.
func TestStaleAccountWritesDoNotRevertSecurityState(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &UserPO{}, &AuthenticationSessionPO{})
	ctx := context.Background()
	repo := NewRepository(db)
	seed := UserPO{Username: "dana", Email: "dana@example.test", Password: "old-hash", Status: userStatusActive}
	require.NoError(t, db.Create(&seed).Error)

	stale, err := repo.FindByID(ctx, seed.ID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdatePasswordAndRevokeSessions(ctx, seed.ID, "new-hash", time.Now()))
	require.NoError(t, repo.setUserStatus(ctx, seed.ID, userStatusDisabled, time.Now()))

	stale.Nickname = "Dana"
	require.NoError(t, repo.Update(ctx, stale))
	require.NoError(t, repo.RecordLogin(ctx, stale.ID, time.Now()))

	var current UserPO
	require.NoError(t, db.First(&current, seed.ID).Error)
	assert.Equal(t, "new-hash", current.Password, "a stale write must not restore the old password")
	assert.Equal(t, userStatusDisabled, current.Status, "a stale write must not re-enable the account")
	assert.Equal(t, "Dana", current.Nickname)
	assert.NotNil(t, current.LastLogin)

	require.NoError(t, db.Delete(&UserPO{}, seed.ID).Error)
	stale.Nickname = "Resurrected"
	require.ErrorIs(t, repo.Update(ctx, stale), domain.ErrUserNotFound)
	require.NoError(t, repo.RecordLogin(ctx, stale.ID, time.Now()))
	var live int64
	require.NoError(t, db.Model(&UserPO{}).Where("id = ?", seed.ID).Count(&live).Error)
	assert.Zero(t, live, "a stale write must not undelete the account")
}

// An account whose username equals another account's email must not capture that account's sign-in.
func TestLoginIdentifierPrefersEmailForAddresses(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &UserPO{}, &AuthenticationSessionPO{})
	ctx := context.Background()
	repo := NewRepository(db)
	victim := UserPO{Username: "victim", Email: "victim@example.test", Password: "x", Status: userStatusActive}
	squatter := UserPO{Username: "victim@example.test", Email: "squatter@example.test", Password: "x", Status: userStatusActive}
	plain := UserPO{Username: "plain", Email: "other@example.test", Password: "x", Status: userStatusActive}
	for _, po := range []*UserPO{&squatter, &victim, &plain} {
		require.NoError(t, db.Create(po).Error)
	}

	byEmail, err := repo.FindByLoginIdentifier(ctx, "victim@example.test")
	require.NoError(t, err)
	assert.Equal(t, victim.ID, byEmail.ID, "an address resolves to the account that owns the email")

	byUsername, err := repo.FindByLoginIdentifier(ctx, "plain")
	require.NoError(t, err)
	assert.Equal(t, plain.ID, byUsername.ID)

	legacy, err := repo.FindByLoginIdentifier(ctx, "squatter@example.test")
	require.NoError(t, err)
	assert.Equal(t, squatter.ID, legacy.ID)
}

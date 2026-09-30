package operator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestGrantRepositoryPostgres(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &user.UserPO{})
	require.NoError(t, db.Exec(`CREATE TABLE platform_operators (
		user_id bigint NOT NULL,
		granted_at timestamptz NOT NULL,
		CONSTRAINT platform_operators_pkey PRIMARY KEY (user_id)
	)`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE platform_operators ADD CONSTRAINT fk_platform_operators_user
		FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`).Error)

	operator := &user.UserPO{Username: "ops", Email: "ops@example.test", Password: "x", Status: 1}
	member := &user.UserPO{Username: "member", Email: "member@example.test", Password: "x", Status: 1}
	require.NoError(t, db.Create(operator).Error)
	require.NoError(t, db.Create(member).Error)

	ctx := context.Background()
	repo := NewRepository(db)
	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

	created, grantedAt, err := repo.insertGrant(ctx, operator.ID, now)
	require.NoError(t, err)
	assert.True(t, created)
	assert.True(t, grantedAt.Equal(now))

	created, grantedAt, err = repo.insertGrant(ctx, operator.ID, now.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, created, "repeating a grant is a no-op")
	assert.True(t, grantedAt.Equal(now), "the original grant time is kept")

	ok, err := repo.isOperator(ctx, operator.ID)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = repo.isOperator(ctx, member.ID)
	require.NoError(t, err)
	assert.False(t, ok)

	ids, err := repo.operatorIDs(ctx, []uint{operator.ID, member.ID})
	require.NoError(t, err)
	assert.Equal(t, map[uint]bool{operator.ID: true}, ids)

	_, _, err = repo.insertGrant(ctx, 999999, now)
	require.Error(t, err, "a grant must reference an existing user")

	require.NoError(t, db.Unscoped().Delete(&user.UserPO{}, operator.ID).Error)
	ok, err = repo.isOperator(ctx, operator.ID)
	require.NoError(t, err)
	assert.False(t, ok, "deleting the account removes its grant")

	removed, err := repo.deleteGrant(ctx, member.ID)
	require.NoError(t, err)
	assert.False(t, removed)
}

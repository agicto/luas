package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
)

func TestRepositoryDeleteAccountRunsPolicyAndSoftDeleteAtomically(t *testing.T) {
	db, err := infradatabase.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&UserPO{}, &AuthenticationSessionPO{}))
	po := &UserPO{
		Username: "atomic-delete",
		Email:    "atomic-delete@example.com",
		Password: "not-used",
		Nickname: "before",
		Status:   1,
	}
	require.NoError(t, db.Create(po).Error)
	repo := NewRepository(db)
	policyErr := errors.New("policy rejected deletion")

	err = repo.DeleteAccount(context.Background(), po.ID, func(ctx context.Context) error {
		tx, ok := infradatabase.TransactionFromContext(ctx)
		require.True(t, ok)
		require.NoError(t, tx.Model(&UserPO{}).
			Where("id = ?", po.ID).
			Update("nickname", "inside-policy").Error)
		return policyErr
	})
	require.ErrorIs(t, err, policyErr)

	var active UserPO
	require.NoError(t, db.First(&active, po.ID).Error)
	assert.Equal(t, "before", active.Nickname)

	err = repo.DeleteAccount(context.Background(), po.ID, func(ctx context.Context) error {
		_, ok := infradatabase.TransactionFromContext(ctx)
		require.True(t, ok)
		return nil
	})
	require.NoError(t, err)
	require.ErrorIs(t, db.First(&UserPO{}, po.ID).Error, gorm.ErrRecordNotFound)

	var deleted UserPO
	require.NoError(t, db.Unscoped().First(&deleted, po.ID).Error)
	assert.True(t, deleted.DeletedAt.Valid)
}

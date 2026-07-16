package user

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
)

func TestRepositoryFindAllUsesStableOrderAndPreservesEmptyPageTotal(t *testing.T) {
	db, err := infradatabase.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&UserPO{}))
	repo := NewRepository(db)
	for index := 1; index <= 3; index++ {
		candidate := &domain.User{
			Username: fmt.Sprintf("list-user-%d", index),
			Email:    fmt.Sprintf("list-user-%d@example.com", index),
			Password: "not-used",
			Status:   1,
		}
		require.NoError(t, repo.Create(context.Background(), candidate))
	}

	countedRepo := NewRepository(db.Session(&gorm.Session{Logger: statementCountingLogger{}}))
	firstPageCtx, firstPageStatements := contextWithStatementCounter(context.Background())
	firstPage, total, err := countedRepo.FindAll(firstPageCtx, 1, 2)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	assert.Equal(t, int64(3), total)
	assert.Greater(t, firstPage[0].ID, firstPage[1].ID)
	assert.Empty(t, firstPage[0].Password)
	assert.Equal(t, int64(1), firstPageStatements.Load())

	emptyPageCtx, emptyPageStatements := contextWithStatementCounter(context.Background())
	emptyPage, total, err := countedRepo.FindAll(emptyPageCtx, 3, 2)
	require.NoError(t, err)
	assert.Empty(t, emptyPage)
	assert.NotNil(t, emptyPage)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, int64(2), emptyPageStatements.Load())
}

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

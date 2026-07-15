package asset

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestRepositoryReadsAccountDeletionGuardInsideOwningTransaction(t *testing.T) {
	db, err := database.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&user.UserPO{}, &AssetPO{}))
	owner := &user.UserPO{
		Username: "asset-transaction-owner",
		Email:    "asset-transaction-owner@example.test",
		Password: "hashed-password",
		Status:   1,
	}
	require.NoError(t, db.Create(owner).Error)
	assetRepository := NewRepository(db)
	userRepository := user.NewRepository(db)
	now := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)

	err = userRepository.DeleteAccount(context.Background(), owner.ID, func(ctx context.Context) error {
		tx, ok := database.TransactionFromContext(ctx)
		if !ok {
			return domain.ErrServiceUnavailable
		}
		if createErr := tx.Create(&AssetPO{
			ID:               "019bf6d8-17c5-7a98-a084-6d45793f5f0c",
			UserID:           owner.ID,
			IdempotencyKey:   "account-delete-transaction",
			RequestHash:      "hash",
			OriginalName:     "private.txt",
			MediaType:        "text/plain",
			SizeBytes:        7,
			Status:           string(domain.AssetStatusPending),
			StagingKey:       "asset-uploads/019bf6d8-17c5-7a98-a084-6d45793f5f0c/object",
			ObjectKey:        "assets/019bf6d8-17c5-7a98-a084-6d45793f5f0c/object",
			PendingExpiresAt: now.Add(time.Hour),
			CreatedAt:        now,
			UpdatedAt:        now,
		}).Error; createErr != nil {
			return createErr
		}
		count, countErr := assetRepository.CountActiveForUser(ctx, owner.ID)
		if countErr != nil {
			return countErr
		}
		if count != 1 {
			return domain.ErrServiceUnavailable
		}
		return domain.ErrAssetCleanupRequired
	})
	assert.ErrorIs(t, err, domain.ErrAssetCleanupRequired)

	var activeUsers int64
	require.NoError(t, db.Model(&user.UserPO{}).Where("id = ?", owner.ID).Count(&activeUsers).Error)
	assert.EqualValues(t, 1, activeUsers)
	var assets int64
	require.NoError(t, db.Model(&AssetPO{}).Count(&assets).Error)
	assert.Zero(t, assets)
}

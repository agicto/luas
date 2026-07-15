package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/database"
)

func newAPIKeyRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&APIKeyPO{}))
	return db
}

func TestRepositoryStaleUsageWriteCannotClearRevocation(t *testing.T) {
	db := newAPIKeyRepositoryTestDB(t)
	repo := NewRepository(db)
	key := &domain.APIKey{
		UserID:    7,
		Name:      "deploy",
		KeyPrefix: "luas_test",
		KeyHash:   "hash",
		Scopes:    []string{"models:invoke"},
	}
	require.NoError(t, repo.Create(context.Background(), key))

	stale, err := repo.FindByHash(context.Background(), key.KeyHash)
	require.NoError(t, err)
	revokedAt := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	changed, err := repo.Revoke(context.Background(), key.UserID, key.ID, revokedAt)
	require.NoError(t, err)
	assert.True(t, changed)

	usedAt := revokedAt.Add(time.Second)
	require.NoError(t, repo.RecordUse(
		context.Background(),
		stale.ID,
		usedAt,
		usedAt.Add(-time.Minute),
	))

	stored, err := repo.FindByHash(context.Background(), key.KeyHash)
	require.NoError(t, err)
	require.NotNil(t, stored.RevokedAt)
	assert.Equal(t, revokedAt, *stored.RevokedAt)
	assert.Nil(t, stored.LastUsedAt)
}

func TestAPIKeyScopesUseStructuredStorage(t *testing.T) {
	po, err := newAPIKeyPO(&domain.APIKey{Scopes: []string{"models:invoke", "models:read"}})
	require.NoError(t, err)
	assert.Equal(t, `["models:invoke","models:read"]`, po.Scopes)
}

func TestAPIKeyScopesReadStructuredAndLegacyStorage(t *testing.T) {
	tests := []struct {
		name   string
		stored string
		want   []string
	}{
		{
			name:   "structured",
			stored: `["models:invoke","models:read"]`,
			want:   []string{"models:invoke", "models:read"},
		},
		{
			name:   "legacy comma separated",
			stored: "models:invoke, models:read",
			want:   []string{"models:invoke", "models:read"},
		},
		{
			name:   "empty",
			stored: "",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := (&APIKeyPO{Scopes: tt.stored}).toDomain()
			require.NoError(t, err)
			assert.Equal(t, tt.want, key.Scopes)
		})
	}
}

func TestAPIKeyScopesRejectMalformedStructuredStorage(t *testing.T) {
	_, err := (&APIKeyPO{Scopes: `["models:read"`}).toDomain()
	require.Error(t, err)
}

func TestRepositoryRecordUseIsThrottled(t *testing.T) {
	db := newAPIKeyRepositoryTestDB(t)
	repo := NewRepository(db)
	key := &domain.APIKey{
		UserID:    7,
		Name:      "deploy",
		KeyPrefix: "luas_throttle",
		KeyHash:   "throttle-hash",
		Scopes:    []string{"models:invoke"},
	}
	require.NoError(t, repo.Create(context.Background(), key))

	firstUse := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.RecordUse(context.Background(), key.ID, firstUse, firstUse.Add(-time.Minute)))
	secondUse := firstUse.Add(30 * time.Second)
	require.NoError(t, repo.RecordUse(context.Background(), key.ID, secondUse, secondUse.Add(-time.Minute)))

	stored, err := repo.FindByHash(context.Background(), key.KeyHash)
	require.NoError(t, err)
	require.NotNil(t, stored.LastUsedAt)
	assert.Equal(t, firstUse, *stored.LastUsedAt)
}

func TestRepositoryRevokeIsOwnedAndIdempotent(t *testing.T) {
	db := newAPIKeyRepositoryTestDB(t)
	repo := NewRepository(db)
	key := &domain.APIKey{
		UserID:    7,
		Name:      "deploy",
		KeyPrefix: "luas_owned",
		KeyHash:   "owned-hash",
		Scopes:    []string{"models:invoke"},
	}
	require.NoError(t, repo.Create(context.Background(), key))

	_, err := repo.Revoke(context.Background(), 8, key.ID, time.Now())
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)

	revokedAt := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	changed, err := repo.Revoke(context.Background(), key.UserID, key.ID, revokedAt)
	require.NoError(t, err)
	assert.True(t, changed)

	changed, err = repo.Revoke(context.Background(), key.UserID, key.ID, revokedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, changed)

	stored, err := repo.FindByHash(context.Background(), key.KeyHash)
	require.NoError(t, err)
	require.NotNil(t, stored.RevokedAt)
	assert.Equal(t, revokedAt, *stored.RevokedAt)
}

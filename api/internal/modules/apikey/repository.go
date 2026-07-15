package apikey

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return r.db.WithContext(ctx), nil
}

func (r *repository) Create(ctx context.Context, key *domain.APIKey) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	po, err := newAPIKeyPO(key)
	if err != nil {
		return err
	}
	if err := db.Create(po).Error; err != nil {
		return err
	}

	key.ID = po.ID
	key.CreatedAt = po.CreatedAt
	key.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) Revoke(
	ctx context.Context,
	userID, id uint,
	revokedAt time.Time,
) (bool, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return false, err
	}

	result := db.Model(&APIKeyPO{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Updates(map[string]any{
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	var existing APIKeyPO
	err = db.Select("id", "revoked_at").
		Where("id = ? AND user_id = ?", id, userID).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, domain.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func (r *repository) RecordUse(
	ctx context.Context,
	id uint,
	usedAt, staleBefore time.Time,
) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return db.Model(&APIKeyPO{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Where("expires_at IS NULL OR expires_at > ?", usedAt).
		Where("last_used_at IS NULL OR last_used_at <= ?", staleBefore).
		UpdateColumn("last_used_at", usedAt).Error
}

func (r *repository) FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*domain.APIKey, int64, error) {
	db, dbErr := r.withContext(ctx)
	if dbErr != nil {
		return nil, 0, dbErr
	}
	var (
		items []*APIKeyPO
		total int64
	)

	offset := (page - 1) * pageSize
	query := db.Model(&APIKeyPO{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	keys, err := toDomainList(items)
	if err != nil {
		return nil, 0, err
	}
	return keys, total, nil
}

func (r *repository) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var po APIKeyPO
	if err := db.Where("key_hash = ?", hash).First(&po).Error; err != nil {
		return nil, err
	}
	return po.toDomain()
}

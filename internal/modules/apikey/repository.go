package apikey

import (
	"context"

	"github.com/zgiai/zgo/internal/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, key *domain.APIKey) error {
	po := newAPIKeyPO(key)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}

	key.ID = po.ID
	key.CreatedAt = po.CreatedAt
	key.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) Update(ctx context.Context, key *domain.APIKey) error {
	po := newAPIKeyPO(key)
	if err := r.db.WithContext(ctx).Save(po).Error; err != nil {
		return err
	}

	key.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) FindByID(ctx context.Context, id uint) (*domain.APIKey, error) {
	var po APIKeyPO
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*domain.APIKey, int64, error) {
	var (
		items []*APIKeyPO
		total int64
	)

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&APIKeyPO{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return toDomainList(items), total, nil
}

func (r *repository) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	var po APIKeyPO
	if err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&po).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

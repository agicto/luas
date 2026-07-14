package apikey

import (
	"context"

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
	po := newAPIKeyPO(key)
	if err := db.Create(po).Error; err != nil {
		return err
	}

	key.ID = po.ID
	key.CreatedAt = po.CreatedAt
	key.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) Update(ctx context.Context, key *domain.APIKey) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	po := newAPIKeyPO(key)
	if err := db.Save(po).Error; err != nil {
		return err
	}

	key.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) FindByID(ctx context.Context, id uint) (*domain.APIKey, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var po APIKeyPO
	if err := db.First(&po, id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*domain.APIKey, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
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

	return toDomainList(items), total, nil
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
	return po.toDomain(), nil
}

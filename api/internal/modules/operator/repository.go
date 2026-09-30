package operator

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
)

// grantPO is one platform-operator grant. The row either exists or it does not; it is never updated.
type grantPO struct {
	UserID    uint      `gorm:"column:user_id;primaryKey"`
	GrantedAt time.Time `gorm:"column:granted_at"`
}

func (grantPO) TableName() string {
	return "platform_operators"
}

type grantStore interface {
	isOperator(ctx context.Context, userID uint) (bool, error)
	operatorIDs(ctx context.Context, userIDs []uint) (map[uint]bool, error)
	insertGrant(ctx context.Context, userID uint, now time.Time) (created bool, grantedAt time.Time, err error)
	deleteGrant(ctx context.Context, userID uint) (bool, error)
	listGrants(ctx context.Context) ([]grantPO, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates the operator grant repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) conn(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return r.db.WithContext(ctx), nil
}

func (r *repository) isOperator(ctx context.Context, userID uint) (bool, error) {
	db, err := r.conn(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := db.Model(&grantPO{}).Where("user_id = ?", userID).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) operatorIDs(ctx context.Context, userIDs []uint) (map[uint]bool, error) {
	result := make(map[uint]bool, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	db, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	var ids []uint
	if err := db.Model(&grantPO{}).Where("user_id IN ?", userIDs).Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

func (r *repository) insertGrant(ctx context.Context, userID uint, now time.Time) (bool, time.Time, error) {
	db, err := r.conn(ctx)
	if err != nil {
		return false, time.Time{}, err
	}
	row := grantPO{UserID: userID, GrantedAt: now.UTC()}
	insert := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if insert.Error != nil {
		return false, time.Time{}, insert.Error
	}
	if insert.RowsAffected == 1 {
		return true, row.GrantedAt, nil
	}
	var existing grantPO
	if err := db.Where("user_id = ?", userID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, time.Time{}, domain.ErrConflict
		}
		return false, time.Time{}, err
	}
	return false, existing.GrantedAt, nil
}

func (r *repository) deleteGrant(ctx context.Context, userID uint) (bool, error) {
	db, err := r.conn(ctx)
	if err != nil {
		return false, err
	}
	result := db.Where("user_id = ?", userID).Delete(&grantPO{})
	return result.RowsAffected > 0, result.Error
}

func (r *repository) listGrants(ctx context.Context) ([]grantPO, error) {
	db, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	var rows []grantPO
	if err := db.Order("granted_at ASC, user_id ASC").Limit(1000).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

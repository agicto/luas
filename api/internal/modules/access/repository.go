package access

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

type repository struct {
	db *gorm.DB
}

var _ domain.AccessRepository = (*repository)(nil)

// NewRepository creates a new repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreateRole(ctx context.Context, role *domain.AccessRole) error {
	po := newAccessRolePO(role)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrAccessRoleSlugAlreadyExists
		}
		return err
	}

	role.ID = po.ID
	role.CreatedAt = po.CreatedAt
	role.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) UpdateRole(ctx context.Context, role *domain.AccessRole) error {
	po := newAccessRolePO(role)
	if err := r.db.WithContext(ctx).Save(po).Error; err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrAccessRoleSlugAlreadyExists
		}
		return err
	}

	role.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) DeleteRole(ctx context.Context, teamID, roleID uint) error {
	result := r.db.WithContext(ctx).Where("team_id = ?", teamID).Delete(&AccessRolePO{}, roleID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrAccessRoleNotFound
	}
	return nil
}

func (r *repository) FindRoleForUser(ctx context.Context, userID, teamID, roleID uint) (*domain.AccessRole, error) {
	var po AccessRolePO
	err := r.db.WithContext(ctx).
		Joins("join team_members on team_members.team_id = access_roles.team_id and team_members.deleted_at is null").
		Where("team_members.user_id = ? and access_roles.team_id = ? and access_roles.id = ?", userID, teamID, roleID).
		First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAccessRoleNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindRolesForUser(ctx context.Context, userID, teamID uint, page, pageSize int) ([]*domain.AccessRole, int64, error) {
	var (
		rows  []AccessRolePO
		total int64
	)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 15
	}

	query := r.db.WithContext(ctx).
		Model(&AccessRolePO{}).
		Joins("join team_members on team_members.team_id = access_roles.team_id and team_members.deleted_at is null").
		Where("team_members.user_id = ? and access_roles.team_id = ?", userID, teamID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("access_roles.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*domain.AccessRole, 0, len(rows))
	for i := range rows {
		items = append(items, rows[i].toDomain())
	}
	return items, total, nil
}

func (r *repository) UserCanAccessTeam(ctx context.Context, userID, teamID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("team_members").
		Where("user_id = ? and team_id = ? and deleted_at is null", userID, teamID).
		Count(&count).Error
	return count > 0, err
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") ||
		strings.Contains(message, "Duplicate entry") ||
		strings.Contains(message, "duplicate key value")
}

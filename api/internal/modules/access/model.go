package access

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

// AccessRolePO is the persistent object for team-scoped roles.
type AccessRolePO struct {
	ID          uint   `gorm:"primaryKey"`
	TeamID      uint   `gorm:"not null;uniqueIndex:idx_access_roles_team_slug;index"`
	Name        string `gorm:"size:120;not null"`
	Slug        string `gorm:"size:120;not null;uniqueIndex:idx_access_roles_team_slug"`
	Description string `gorm:"type:text"`
	Permissions string `gorm:"type:text"`
	System      bool   `gorm:"not null;default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (AccessRolePO) TableName() string {
	return "access_roles"
}

func (po *AccessRolePO) toDomain() *domain.AccessRole {
	if po == nil {
		return nil
	}

	return &domain.AccessRole{
		ID:          po.ID,
		TeamID:      po.TeamID,
		Name:        po.Name,
		Slug:        po.Slug,
		Description: po.Description,
		Permissions: splitPermissions(po.Permissions),
		System:      po.System,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func newAccessRolePO(role *domain.AccessRole) *AccessRolePO {
	if role == nil {
		return nil
	}

	return &AccessRolePO{
		ID:          role.ID,
		TeamID:      role.TeamID,
		Name:        role.Name,
		Slug:        role.Slug,
		Description: role.Description,
		Permissions: joinPermissions(role.Permissions),
		System:      role.System,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func splitPermissions(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func joinPermissions(values []string) string {
	return strings.Join(values, ",")
}

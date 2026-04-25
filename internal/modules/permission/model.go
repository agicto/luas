package permission

import (
	"time"

	"github.com/zgiai/zgo/internal/domain"
	"gorm.io/gorm"
)

// RolePO is the persistent object for role records.
type RolePO struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"size:50;not null;unique"`
	DisplayName string         `gorm:"size:100"`
	Description string         `gorm:"size:255"`
	IsDefault   bool           `gorm:"default:false"`
}

func (RolePO) TableName() string {
	return "roles"
}

func (po *RolePO) toDomain() *domain.Role {
	if po == nil {
		return nil
	}
	return &domain.Role{
		ID:          po.ID,
		Name:        po.Name,
		DisplayName: po.DisplayName,
		Description: po.Description,
		IsDefault:   po.IsDefault,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func newRolePO(role *domain.Role) *RolePO {
	if role == nil {
		return nil
	}
	return &RolePO{
		ID:          role.ID,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsDefault:   role.IsDefault,
	}
}

func toRoleDomainList(poList []*RolePO) []*domain.Role {
	result := make([]*domain.Role, len(poList))
	for i, po := range poList {
		result[i] = po.toDomain()
	}
	return result
}

// PermissionPO is the persistent object for permission records.
type PermissionPO struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"size:100;not null;unique"`
	DisplayName string         `gorm:"size:100"`
	Description string         `gorm:"size:255"`
	Module      string         `gorm:"size:50"`
}

func (PermissionPO) TableName() string {
	return "permissions"
}

func (po *PermissionPO) toDomain() *domain.Permission {
	if po == nil {
		return nil
	}
	return &domain.Permission{
		ID:          po.ID,
		Name:        po.Name,
		DisplayName: po.DisplayName,
		Description: po.Description,
		Module:      po.Module,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func newPermissionPO(permission *domain.Permission) *PermissionPO {
	if permission == nil {
		return nil
	}
	return &PermissionPO{
		ID:          permission.ID,
		CreatedAt:   permission.CreatedAt,
		UpdatedAt:   permission.UpdatedAt,
		Name:        permission.Name,
		DisplayName: permission.DisplayName,
		Description: permission.Description,
		Module:      permission.Module,
	}
}

func toPermissionDomainList(poList []*PermissionPO) []*domain.Permission {
	result := make([]*domain.Permission, len(poList))
	for i, po := range poList {
		result[i] = po.toDomain()
	}
	return result
}

// RolePermissionPO is the persistent object for role-permission associations.
type RolePermissionPO struct {
	ID           uint `gorm:"primaryKey"`
	RoleID       uint `gorm:"not null;index"`
	PermissionID uint `gorm:"not null;index"`
	CreatedAt    time.Time
}

func (RolePermissionPO) TableName() string {
	return "role_permissions"
}

// UserRolePO is the persistent object for user-role associations.
type UserRolePO struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null;index"`
	RoleID    uint `gorm:"not null;index"`
	CreatedAt time.Time
}

func (UserRolePO) TableName() string {
	return "user_roles"
}

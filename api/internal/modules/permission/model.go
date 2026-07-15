package permission

import (
	"sort"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/organization"
)

// AccessRolePO is an organization-scoped role persistence model.
type AccessRolePO struct {
	ID             uint   `gorm:"primaryKey"`
	OrganizationID uint   `gorm:"not null;uniqueIndex:idx_permission_roles_org_slug,priority:1;index"`
	Name           string `gorm:"size:100;not null"`
	Slug           string `gorm:"size:50;not null;uniqueIndex:idx_permission_roles_org_slug,priority:2"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Organization   organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Permissions    []AccessRolePermissionPO    `gorm:"foreignKey:AccessRoleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName returns the stable permission role table name.
func (AccessRolePO) TableName() string { return "permission_roles" }

// AccessRolePermissionPO stores one exact catalog key granted to a role.
type AccessRolePermissionPO struct {
	AccessRoleID uint   `gorm:"primaryKey;index:idx_permission_role_grants_permission_role,priority:2"`
	Permission   string `gorm:"primaryKey;size:100;index:idx_permission_role_grants_permission_role,priority:1"`
	CreatedAt    time.Time
	AccessRole   AccessRolePO `gorm:"foreignKey:AccessRoleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName returns the stable permission grant table name.
func (AccessRolePermissionPO) TableName() string { return "permission_role_grants" }

// AccessRoleAssignmentPO assigns one role to one organization membership.
type AccessRoleAssignmentPO struct {
	ID             uint `gorm:"primaryKey"`
	OrganizationID uint `gorm:"not null;uniqueIndex:idx_permission_role_assignments_org_member_role,priority:1;index:idx_permission_role_assignments_org_member,priority:1"`
	MembershipID   uint `gorm:"not null;uniqueIndex:idx_permission_role_assignments_org_member_role,priority:2;index:idx_permission_role_assignments_org_member,priority:2"`
	AccessRoleID   uint `gorm:"not null;uniqueIndex:idx_permission_role_assignments_org_member_role,priority:3;index"`
	CreatedAt      time.Time
	Organization   organization.OrganizationPO           `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Membership     organization.OrganizationMembershipPO `gorm:"foreignKey:MembershipID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AccessRole     AccessRolePO                          `gorm:"foreignKey:AccessRoleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName returns the stable permission assignment table name.
func (AccessRoleAssignmentPO) TableName() string { return "permission_role_assignments" }

func newAccessRolePO(role *domain.AccessRole) *AccessRolePO {
	if role == nil {
		return nil
	}
	return &AccessRolePO{
		ID:             role.ID,
		OrganizationID: role.OrganizationID,
		Name:           role.Name,
		Slug:           role.Slug,
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}
}

func (po *AccessRolePO) toDomain() *domain.AccessRole {
	if po == nil {
		return nil
	}
	permissions := make([]domain.PermissionKey, len(po.Permissions))
	for index, permission := range po.Permissions {
		permissions[index] = domain.PermissionKey(permission.Permission)
	}
	sort.Slice(permissions, func(left, right int) bool { return permissions[left] < permissions[right] })
	return &domain.AccessRole{
		ID:             po.ID,
		OrganizationID: po.OrganizationID,
		Name:           po.Name,
		Slug:           po.Slug,
		Permissions:    permissions,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

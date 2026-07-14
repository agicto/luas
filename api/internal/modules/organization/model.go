package organization

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

// OrganizationPO is the organization persistence model.
type OrganizationPO struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100;not null"`
	Slug      string `gorm:"size:50;not null;uniqueIndex"`
	CreatedBy uint   `gorm:"not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Creator   user.UserPO `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// TableName returns the stable organization table name.
func (OrganizationPO) TableName() string {
	return "organizations"
}

// OrganizationMembershipPO is the organization membership persistence model.
type OrganizationMembershipPO struct {
	ID             uint      `gorm:"primaryKey"`
	OrganizationID uint      `gorm:"not null;uniqueIndex:idx_organization_memberships_org_user,priority:1"`
	UserID         uint      `gorm:"not null;uniqueIndex:idx_organization_memberships_org_user,priority:2;index:idx_organization_memberships_user_created,priority:1;index:idx_organization_memberships_user_role,priority:1"`
	Role           string    `gorm:"size:16;not null;index:idx_organization_memberships_user_role,priority:2;check:organization_memberships_role_check,role IN ('owner','admin','member')"`
	CreatedAt      time.Time `gorm:"index:idx_organization_memberships_user_created,priority:2"`
	UpdatedAt      time.Time
	Organization   OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User           user.UserPO    `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// TableName returns the stable organization membership table name.
func (OrganizationMembershipPO) TableName() string {
	return "organization_memberships"
}

func newOrganizationPO(organization *domain.Organization) *OrganizationPO {
	if organization == nil {
		return nil
	}
	return &OrganizationPO{
		ID:        organization.ID,
		Name:      organization.Name,
		Slug:      organization.Slug,
		CreatedBy: organization.CreatedBy,
		CreatedAt: organization.CreatedAt,
		UpdatedAt: organization.UpdatedAt,
	}
}

func newMembershipPO(membership *domain.OrganizationMembership) *OrganizationMembershipPO {
	if membership == nil {
		return nil
	}
	return &OrganizationMembershipPO{
		ID:             membership.ID,
		OrganizationID: membership.OrganizationID,
		UserID:         membership.UserID,
		Role:           string(membership.Role),
		CreatedAt:      membership.CreatedAt,
		UpdatedAt:      membership.UpdatedAt,
	}
}

func (po *OrganizationPO) toDomain() *domain.Organization {
	if po == nil {
		return nil
	}
	return &domain.Organization{
		ID:        po.ID,
		Name:      po.Name,
		Slug:      po.Slug,
		CreatedBy: po.CreatedBy,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

func (po *OrganizationMembershipPO) toDomain() *domain.OrganizationMembership {
	if po == nil {
		return nil
	}
	return &domain.OrganizationMembership{
		ID:             po.ID,
		OrganizationID: po.OrganizationID,
		UserID:         po.UserID,
		Role:           domain.OrganizationRole(po.Role),
		Organization:   po.Organization.toDomain(),
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

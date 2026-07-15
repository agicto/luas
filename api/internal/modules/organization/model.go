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

// OrganizationInvitationPO is the invitation persistence model. PendingKey is
// nullable so every database driver can enforce one active invite while still
// retaining accepted and revoked history.
type OrganizationInvitationPO struct {
	ID             uint       `gorm:"primaryKey"`
	OrganizationID uint       `gorm:"not null;index:idx_organization_invitations_org_created,priority:1;index:idx_organization_invitations_org_email,priority:1"`
	InvitedBy      uint       `gorm:"not null;index"`
	Email          string     `gorm:"size:100;not null;index:idx_organization_invitations_org_email,priority:2"`
	Role           string     `gorm:"size:16;not null;check:organization_invitations_role_check,role IN ('admin','member')"`
	TokenHash      string     `gorm:"size:64;not null;uniqueIndex"`
	PendingKey     *string    `gorm:"size:64;uniqueIndex"`
	ExpiresAt      time.Time  `gorm:"not null;index"`
	AcceptedAt     *time.Time `gorm:"index"`
	AcceptedBy     *uint      `gorm:"index"`
	RevokedAt      *time.Time `gorm:"index"`
	RevokedBy      *uint      `gorm:"index"`
	CreatedAt      time.Time  `gorm:"index:idx_organization_invitations_org_created,priority:2"`
	UpdatedAt      time.Time
	Organization   OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Inviter        user.UserPO    `gorm:"foreignKey:InvitedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Acceptor       *user.UserPO   `gorm:"foreignKey:AcceptedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Revoker        *user.UserPO   `gorm:"foreignKey:RevokedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// TableName returns the stable organization invitation table name.
func (OrganizationInvitationPO) TableName() string {
	return "organization_invitations"
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

func newInvitationPO(invitation *domain.OrganizationInvitation) *OrganizationInvitationPO {
	if invitation == nil {
		return nil
	}
	return &OrganizationInvitationPO{
		ID:             invitation.ID,
		OrganizationID: invitation.OrganizationID,
		InvitedBy:      invitation.InvitedBy,
		Email:          invitation.Email,
		Role:           string(invitation.Role),
		TokenHash:      invitation.TokenHash,
		ExpiresAt:      invitation.ExpiresAt,
		AcceptedAt:     invitation.AcceptedAt,
		AcceptedBy:     invitation.AcceptedBy,
		RevokedAt:      invitation.RevokedAt,
		RevokedBy:      invitation.RevokedBy,
		CreatedAt:      invitation.CreatedAt,
		UpdatedAt:      invitation.UpdatedAt,
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
	membership := &domain.OrganizationMembership{
		ID:             po.ID,
		OrganizationID: po.OrganizationID,
		UserID:         po.UserID,
		Role:           domain.OrganizationRole(po.Role),
		Organization:   po.Organization.toDomain(),
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
	if po.User.ID != 0 {
		membership.User = &domain.User{
			ID:       po.User.ID,
			Username: po.User.Username,
			Nickname: po.User.Nickname,
			Avatar:   po.User.Avatar,
		}
	}
	return membership
}

func (po *OrganizationInvitationPO) toDomain() *domain.OrganizationInvitation {
	if po == nil {
		return nil
	}
	return &domain.OrganizationInvitation{
		ID:             po.ID,
		OrganizationID: po.OrganizationID,
		InvitedBy:      po.InvitedBy,
		Email:          po.Email,
		Role:           domain.OrganizationRole(po.Role),
		TokenHash:      po.TokenHash,
		ExpiresAt:      po.ExpiresAt,
		AcceptedAt:     po.AcceptedAt,
		AcceptedBy:     po.AcceptedBy,
		RevokedAt:      po.RevokedAt,
		RevokedBy:      po.RevokedBy,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

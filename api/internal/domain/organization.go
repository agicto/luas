package domain

import (
	"context"
	"time"
)

// OrganizationRole is scoped to one organization and is not a global RBAC role.
type OrganizationRole string

const (
	OrganizationRoleOwner  OrganizationRole = "owner"
	OrganizationRoleAdmin  OrganizationRole = "admin"
	OrganizationRoleMember OrganizationRole = "member"
)

// IsValid reports whether the role belongs to the organization vocabulary.
func (r OrganizationRole) IsValid() bool {
	switch r {
	case OrganizationRoleOwner, OrganizationRoleAdmin, OrganizationRoleMember:
		return true
	default:
		return false
	}
}

// CanManageOrganization reports whether the role may change organization settings.
func (r OrganizationRole) CanManageOrganization() bool {
	return r == OrganizationRoleOwner || r == OrganizationRoleAdmin
}

// CanBeInvited reports whether the role may be granted by an invitation.
func (r OrganizationRole) CanBeInvited() bool {
	return r == OrganizationRoleAdmin || r == OrganizationRoleMember
}

// Organization is the tenant/account ownership boundary.
type Organization struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrganizationMembership links one user to one organization-scoped role.
type OrganizationMembership struct {
	ID             uint             `json:"id"`
	OrganizationID uint             `json:"organization_id"`
	UserID         uint             `json:"user_id"`
	Role           OrganizationRole `json:"role"`
	Organization   *Organization    `json:"-"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// OrganizationInvitationStatus is the externally visible invitation lifecycle state.
type OrganizationInvitationStatus string

const (
	OrganizationInvitationStatusPending  OrganizationInvitationStatus = "pending"
	OrganizationInvitationStatusAccepted OrganizationInvitationStatus = "accepted"
	OrganizationInvitationStatusRevoked  OrganizationInvitationStatus = "revoked"
	OrganizationInvitationStatusExpired  OrganizationInvitationStatus = "expired"
)

// OrganizationInvitation is a one-time organization membership offer.
type OrganizationInvitation struct {
	ID             uint             `json:"id"`
	OrganizationID uint             `json:"organization_id"`
	InvitedBy      uint             `json:"invited_by"`
	Email          string           `json:"email"`
	Role           OrganizationRole `json:"role"`
	TokenHash      string           `json:"-"`
	ExpiresAt      time.Time        `json:"expires_at"`
	AcceptedAt     *time.Time       `json:"accepted_at,omitempty"`
	AcceptedBy     *uint            `json:"accepted_by,omitempty"`
	RevokedAt      *time.Time       `json:"revoked_at,omitempty"`
	RevokedBy      *uint            `json:"revoked_by,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// Status derives expiry without mutating invitation history on read.
func (i *OrganizationInvitation) Status(now time.Time) OrganizationInvitationStatus {
	if i != nil && i.AcceptedAt != nil {
		return OrganizationInvitationStatusAccepted
	}
	if i != nil && i.RevokedAt != nil {
		return OrganizationInvitationStatusRevoked
	}
	if i == nil || !now.Before(i.ExpiresAt) {
		return OrganizationInvitationStatusExpired
	}
	return OrganizationInvitationStatusPending
}

// OrganizationRepository is the persistence seam for membership-scoped organization access.
type OrganizationRepository interface {
	CreateWithOwner(ctx context.Context, organization *Organization, owner *OrganizationMembership) error
	FindForUser(ctx context.Context, organizationID, userID uint) (*OrganizationMembership, error)
	ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*OrganizationMembership, int64, error)
	Update(ctx context.Context, organization *Organization) error
	CountOwnedByUser(ctx context.Context, userID uint) (int64, error)
}

// OrganizationInvitationRepository owns invitation persistence and atomic acceptance.
type OrganizationInvitationRepository interface {
	CreateInvitation(ctx context.Context, invitation *OrganizationInvitation, now time.Time) error
	ListInvitations(ctx context.Context, organizationID uint, page, pageSize int) ([]*OrganizationInvitation, int64, error)
	RevokeInvitation(ctx context.Context, organizationID, invitationID, revokedBy uint, now time.Time) (*OrganizationInvitation, error)
	AcceptInvitation(ctx context.Context, tokenHash string, userID uint, now time.Time) (*OrganizationMembership, *OrganizationInvitation, error)
}

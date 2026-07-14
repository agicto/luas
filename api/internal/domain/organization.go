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

// OrganizationRepository is the persistence seam for membership-scoped organization access.
type OrganizationRepository interface {
	CreateWithOwner(ctx context.Context, organization *Organization, owner *OrganizationMembership) error
	FindForUser(ctx context.Context, organizationID, userID uint) (*OrganizationMembership, error)
	ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*OrganizationMembership, int64, error)
	Update(ctx context.Context, organization *Organization) error
	CountOwnedByUser(ctx context.Context, userID uint) (int64, error)
}

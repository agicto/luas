package domain

import (
	"context"
	"regexp"
	"sort"
	"time"
)

var permissionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

// PermissionKey is one exact, code-owned authorization capability.
type PermissionKey string

// IsValid reports whether the key uses the canonical lowercase dotted grammar.
func (k PermissionKey) IsValid() bool {
	return len(k) <= 100 && permissionKeyPattern.MatchString(string(k))
}

// AccessRole groups exact permissions inside one organization.
type AccessRole struct {
	ID             uint            `json:"id"`
	OrganizationID uint            `json:"organization_id"`
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	Permissions    []PermissionKey `json:"permissions"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// PermissionContext is the current persisted authorization view for one membership.
type PermissionContext struct {
	OrganizationID   uint             `json:"organization_id"`
	MembershipID     uint             `json:"membership_id"`
	OrganizationRole OrganizationRole `json:"-"`
	IsOwner          bool             `json:"is_owner"`
	AccessRoleIDs    []uint           `json:"access_role_ids"`
	Permissions      []PermissionKey  `json:"permissions"`
}

// Has reports exact permission membership. Permission arrays are canonical and sorted.
func (c *PermissionContext) Has(key PermissionKey) bool {
	if c == nil {
		return false
	}
	index := sort.Search(len(c.Permissions), func(index int) bool {
		return c.Permissions[index] >= key
	})
	return index < len(c.Permissions) && c.Permissions[index] == key
}

// PermissionRepository is the persistence seam for organization-scoped access roles.
type PermissionRepository interface {
	WithinTransaction(ctx context.Context, operation func(context.Context) error) error
	Effective(ctx context.Context, expected OrganizationContext) (*PermissionContext, error)
	ListRoles(ctx context.Context, organizationID uint, page, pageSize int) ([]*AccessRole, int64, error)
	FindRole(ctx context.Context, organizationID, roleID uint) (*AccessRole, error)
	FindRoles(ctx context.Context, organizationID uint, roleIDs []uint) ([]*AccessRole, error)
	CreateRole(ctx context.Context, role *AccessRole) error
	UpdateRole(ctx context.Context, role *AccessRole) error
	DeleteRole(ctx context.Context, organizationID, roleID uint) error
	MemberRoleIDs(ctx context.Context, organizationID, memberID uint) ([]uint, error)
	ReplaceMemberRoleIDs(ctx context.Context, organizationID, memberID uint, roleIDs []uint) error
}

// PermissionAuthorizer is the replaceable route/service policy-check seam.
type PermissionAuthorizer interface {
	Effective(ctx context.Context, organization OrganizationContext) (*PermissionContext, error)
	Authorize(ctx context.Context, organization OrganizationContext, permission PermissionKey) error
}

package permission

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/domain"
)

const maxRoleValues = 100

var accessRoleSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,48}[a-z0-9])$`)

// CreateAccessRoleRequest creates one organization access role.
type CreateAccessRoleRequest struct {
	Name        string                 `json:"name" binding:"required,min=2,max=100"`
	Slug        string                 `json:"slug" binding:"required,min=3,max=50"`
	Permissions []domain.PermissionKey `json:"permissions" binding:"max=100"`
}

// UpdateAccessRoleRequest changes the role display name and complete grant set.
type UpdateAccessRoleRequest struct {
	Name        string                 `json:"name" binding:"required,min=2,max=100"`
	Permissions []domain.PermissionKey `json:"permissions" binding:"max=100"`
}

// ReplaceMemberAccessRolesRequest atomically replaces one membership's role set.
type ReplaceMemberAccessRolesRequest struct {
	AccessRoleIDs []uint `json:"access_role_ids" binding:"max=100,dive,gt=0"`
}

// PermissionCatalogResponse returns all code-owned permission keys.
type PermissionCatalogResponse struct {
	Permissions []domain.PermissionKey `json:"permissions"`
}

// MemberAccessRolesResponse is the relationship-only assignment view.
type MemberAccessRolesResponse struct {
	MemberID      uint   `json:"member_id"`
	AccessRoleIDs []uint `json:"access_role_ids"`
}

func (r *CreateAccessRoleRequest) validationErrors() map[string][]string {
	errors := make(map[string][]string)
	if r == nil || !validAccessRoleName(r.Name) {
		errors["name"] = []string{"name must contain between 2 and 100 characters"}
	}
	if r == nil || !accessRoleSlugPattern.MatchString(r.Slug) {
		errors["slug"] = []string{"slug must contain 3-50 lowercase letters, numbers, or hyphens and cannot end with a hyphen"}
	}
	if r == nil || len(r.Permissions) > maxRoleValues || hasDuplicateOrInvalidPermissions(r.Permissions) {
		errors["permissions"] = []string{"permissions must contain at most 100 unique canonical keys"}
	}
	return errors
}

func (r *UpdateAccessRoleRequest) validationErrors() map[string][]string {
	errors := make(map[string][]string)
	if r == nil || !validAccessRoleName(r.Name) {
		errors["name"] = []string{"name must contain between 2 and 100 characters"}
	}
	if r == nil || len(r.Permissions) > maxRoleValues || hasDuplicateOrInvalidPermissions(r.Permissions) {
		errors["permissions"] = []string{"permissions must contain at most 100 unique canonical keys"}
	}
	return errors
}

func (r *ReplaceMemberAccessRolesRequest) validationErrors() map[string][]string {
	if r == nil || len(r.AccessRoleIDs) > maxRoleValues || hasDuplicateOrZeroIDs(r.AccessRoleIDs) {
		return map[string][]string{"access_role_ids": {"access_role_ids must contain at most 100 unique positive IDs"}}
	}
	return nil
}

func validAccessRoleName(value string) bool {
	length := utf8.RuneCountInString(strings.TrimSpace(value))
	return length >= 2 && length <= 100
}

func hasDuplicateOrInvalidPermissions(values []domain.PermissionKey) bool {
	seen := make(map[domain.PermissionKey]struct{}, len(values))
	for _, value := range values {
		if !value.IsValid() {
			return true
		}
		if _, duplicate := seen[value]; duplicate {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func hasDuplicateOrZeroIDs(values []uint) bool {
	seen := make(map[uint]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			return true
		}
		if _, duplicate := seen[value]; duplicate {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

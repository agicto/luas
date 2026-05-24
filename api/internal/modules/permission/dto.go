package permission

import (
	"time"

	"github.com/zgiai/zgo/internal/domain"
)

// CreateRoleRequest is the request for creating a role.
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

func (r *CreateRoleRequest) toDomain() *domain.Role {
	if r == nil {
		return nil
	}
	return &domain.Role{
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		IsDefault:   r.IsDefault,
	}
}

// UpdateRoleRequest is the request for updating a role.
type UpdateRoleRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

func (r *UpdateRoleRequest) applyTo(role *domain.Role) {
	if r == nil || role == nil {
		return
	}
	if r.Name != "" {
		role.Name = r.Name
	}
	if r.DisplayName != "" {
		role.DisplayName = r.DisplayName
	}
	if r.Description != "" {
		role.Description = r.Description
	}
}

// AssignRoleRequest is the request for assigning a role to a user.
type AssignRoleRequest struct {
	UserID uint `json:"user_id" binding:"required"`
	RoleID uint `json:"role_id" binding:"required"`
}

// AssignPermissionRequest is the request for assigning a permission to a role.
type AssignPermissionRequest struct {
	RoleID       uint `json:"role_id" binding:"required"`
	PermissionID uint `json:"permission_id" binding:"required"`
}

// RoleResponse is the response for role data.
type RoleResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
}

// PermissionResponse is the response for permission data.
type PermissionResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Module      string `json:"module"`
}

// UserRolesResponse is the response for user roles.
type UserRolesResponse struct {
	UserID uint            `json:"user_id"`
	Roles  []*RoleResponse `json:"roles"`
}

func toRoleResponse(role *domain.Role) *RoleResponse {
	if role == nil {
		return nil
	}
	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsDefault:   role.IsDefault,
		CreatedAt:   formatTimestamp(role.CreatedAt),
	}
}

func toRoleResponses(roles []*domain.Role) []*RoleResponse {
	result := make([]*RoleResponse, len(roles))
	for i, role := range roles {
		result[i] = toRoleResponse(role)
	}
	return result
}

func toPermissionResponse(permission *domain.Permission) *PermissionResponse {
	if permission == nil {
		return nil
	}
	return &PermissionResponse{
		ID:          permission.ID,
		Name:        permission.Name,
		DisplayName: permission.DisplayName,
		Description: permission.Description,
		Module:      permission.Module,
	}
}

func toPermissionResponses(permissions []*domain.Permission) []*PermissionResponse {
	result := make([]*PermissionResponse, len(permissions))
	for i, permission := range permissions {
		result[i] = toPermissionResponse(permission)
	}
	return result
}

func newUserRolesResponse(userID uint, roles []*domain.Role) UserRolesResponse {
	return UserRolesResponse{
		UserID: userID,
		Roles:  toRoleResponses(roles),
	}
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339)
}

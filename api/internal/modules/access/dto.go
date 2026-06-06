package access

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

// CreateRoleRequest represents the request to create a team role.
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required,max=120"`
	Slug        string   `json:"slug" binding:"omitempty,max=120"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest represents the request to update a team role.
type UpdateRoleRequest struct {
	Name        *string  `json:"name,omitempty" binding:"omitempty,max=120"`
	Description *string  `json:"description,omitempty" binding:"omitempty,max=1000"`
	Permissions []string `json:"permissions"`
}

// AccessRoleResponse represents the API response for a team role.
type AccessRoleResponse struct {
	ID          uint      `json:"id"`
	TeamID      uint      `json:"team_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`
	System      bool      `json:"system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PermissionResponse represents a framework permission key.
type PermissionResponse struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

func toAccessRoleResponse(role *domain.AccessRole) *AccessRoleResponse {
	if role == nil {
		return nil
	}

	return &AccessRoleResponse{
		ID:          role.ID,
		TeamID:      role.TeamID,
		Name:        role.Name,
		Slug:        role.Slug,
		Description: role.Description,
		Permissions: append([]string(nil), role.Permissions...),
		System:      role.System,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func toAccessRoleResponses(items []*domain.AccessRole) []*AccessRoleResponse {
	result := make([]*AccessRoleResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toAccessRoleResponse(item))
	}
	return result
}

func toPermissionResponses(items []domain.AccessPermission) []PermissionResponse {
	result := make([]PermissionResponse, 0, len(items))
	for _, item := range items {
		result = append(result, PermissionResponse(item))
	}
	return result
}

package domain

import (
	"context"
	"time"
)

// AccessPermission is a framework-level permission key exposed to apps.
type AccessPermission struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

// AccessRole is a team-scoped role with a normalized permission set.
type AccessRole struct {
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

// AccessRepository defines team-scoped role persistence.
type AccessRepository interface {
	CreateRole(ctx context.Context, role *AccessRole) error
	UpdateRole(ctx context.Context, role *AccessRole) error
	DeleteRole(ctx context.Context, teamID, roleID uint) error
	FindRoleForUser(ctx context.Context, userID, teamID, roleID uint) (*AccessRole, error)
	FindRolesForUser(ctx context.Context, userID, teamID uint, page, pageSize int) ([]*AccessRole, int64, error)
	UserCanAccessTeam(ctx context.Context, userID, teamID uint) (bool, error)
}

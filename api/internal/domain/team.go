package domain

import (
	"context"
	"time"
)

const (
	TeamStatusActive   = "active"
	TeamStatusArchived = "archived"

	TeamRoleOwner  = "owner"
	TeamRoleAdmin  = "admin"
	TeamRoleMember = "member"

	TeamPlanFree = "free"
)

// Team is the organization boundary for multi-tenant starter apps.
type Team struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerUserID uint      `json:"owner_user_id"`
	Plan        string    `json:"plan"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeamMember records a user's membership in a team.
type TeamMember struct {
	ID        uint      `json:"id"`
	TeamID    uint      `json:"team_id"`
	UserID    uint      `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TeamRepository defines persistence for Team and its owner membership.
type TeamRepository interface {
	CreateWithOwner(ctx context.Context, team *Team, owner *TeamMember) error
	Update(ctx context.Context, team *Team) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Team, error)
	FindBySlug(ctx context.Context, slug string) (*Team, error)
	FindForUser(ctx context.Context, userID, teamID uint) (*Team, error)
	FindAllForUser(ctx context.Context, userID uint, page, pageSize int) ([]*Team, int64, error)
}

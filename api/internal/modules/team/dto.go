package team

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

// CreateTeamRequest represents the request to create a Team.
type CreateTeamRequest struct {
	Name string `json:"name" binding:"required,max=120"`
	Slug string `json:"slug" binding:"omitempty,max=120"`
}

// UpdateTeamRequest represents the request to update a Team.
type UpdateTeamRequest struct {
	Name   *string `json:"name,omitempty" binding:"omitempty,max=120"`
	Status *string `json:"status,omitempty" binding:"omitempty,oneof=active archived"`
}

// TeamResponse represents the API response for Team.
type TeamResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerUserID uint      `json:"owner_user_id"`
	Plan        string    `json:"plan"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toTeamResponse(item *domain.Team) *TeamResponse {
	if item == nil {
		return nil
	}

	return &TeamResponse{
		ID:          item.ID,
		Name:        item.Name,
		Slug:        item.Slug,
		OwnerUserID: item.OwnerUserID,
		Plan:        item.Plan,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func toTeamResponses(items []*domain.Team) []*TeamResponse {
	result := make([]*TeamResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toTeamResponse(item))
	}
	return result
}

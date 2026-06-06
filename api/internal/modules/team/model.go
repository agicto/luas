package team

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"gorm.io/gorm"
)

// TeamPO is the persistent object for Team.
type TeamPO struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Slug        string `gorm:"size:120;not null;uniqueIndex"`
	OwnerUserID uint   `gorm:"not null;index"`
	Plan        string `gorm:"size:40;not null;default:'free'"`
	Status      string `gorm:"size:40;not null;default:'active';index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TeamMemberPO is the persistent object for team memberships.
type TeamMemberPO struct {
	ID        uint `gorm:"primaryKey"`
	TeamID    uint `gorm:"not null;uniqueIndex:idx_team_members_team_user;index"`
	UserID    uint `gorm:"not null;uniqueIndex:idx_team_members_team_user;index"`
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (TeamPO) TableName() string {
	return "teams"
}

func (TeamMemberPO) TableName() string {
	return "team_members"
}

func (po *TeamPO) toDomain() *domain.Team {
	if po == nil {
		return nil
	}

	return &domain.Team{
		ID:          po.ID,
		Name:        po.Name,
		Slug:        po.Slug,
		OwnerUserID: po.OwnerUserID,
		Plan:        po.Plan,
		Status:      po.Status,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func newTeamPO(item *domain.Team) *TeamPO {
	if item == nil {
		return nil
	}

	return &TeamPO{
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

func newTeamMemberPO(item *domain.TeamMember) *TeamMemberPO {
	if item == nil {
		return nil
	}

	return &TeamMemberPO{
		ID:        item.ID,
		TeamID:    item.TeamID,
		UserID:    item.UserID,
		Role:      item.Role,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

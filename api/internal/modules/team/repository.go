package team

import (
	"context"
	"errors"
	"strings"

	"github.com/zgiai/luas/api/internal/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

var _ domain.TeamRepository = (*repository)(nil)

// NewRepository creates a new repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreateWithOwner(ctx context.Context, team *domain.Team, owner *domain.TeamMember) error {
	teamPO := newTeamPO(team)
	ownerPO := newTeamMemberPO(owner)

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(teamPO).Error; err != nil {
			return err
		}
		ownerPO.TeamID = teamPO.ID
		return tx.Create(ownerPO).Error
	}); err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrTeamSlugAlreadyExists
		}
		return err
	}

	team.ID = teamPO.ID
	team.CreatedAt = teamPO.CreatedAt
	team.UpdatedAt = teamPO.UpdatedAt
	owner.ID = ownerPO.ID
	owner.TeamID = ownerPO.TeamID
	owner.CreatedAt = ownerPO.CreatedAt
	owner.UpdatedAt = ownerPO.UpdatedAt
	return nil
}

func (r *repository) Update(ctx context.Context, team *domain.Team) error {
	po := newTeamPO(team)
	if err := r.db.WithContext(ctx).Save(po).Error; err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrTeamSlugAlreadyExists
		}
		return err
	}

	team.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&TeamPO{}, id).Error
}

func (r *repository) FindByID(ctx context.Context, id uint) (*domain.Team, error) {
	var po TeamPO
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (*domain.Team, error) {
	var po TeamPO
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTeamNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindForUser(ctx context.Context, userID, teamID uint) (*domain.Team, error) {
	var po TeamPO
	err := r.db.WithContext(ctx).
		Joins("join team_members on team_members.team_id = teams.id and team_members.deleted_at is null").
		Where("team_members.user_id = ? and teams.id = ?", userID, teamID).
		First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTeamNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindAllForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.Team, int64, error) {
	var (
		rows  []TeamPO
		total int64
	)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 15
	}

	query := r.db.WithContext(ctx).
		Model(&TeamPO{}).
		Joins("join team_members on team_members.team_id = teams.id and team_members.deleted_at is null").
		Where("team_members.user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*domain.Team, 0, len(rows))
	for i := range rows {
		items = append(items, rows[i].toDomain())
	}
	return items, total, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "UNIQUE constraint failed") ||
		strings.Contains(message, "Duplicate entry") ||
		strings.Contains(message, "duplicate key value")
}

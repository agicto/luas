package team

import (
	"context"
	"regexp"
	"strings"

	"github.com/zgiai/luas/api/internal/domain"
)

// Service defines the business interface for Team.
type Service interface {
	CreateForUser(ctx context.Context, userID uint, req *CreateTeamRequest) (*domain.Team, error)
	UpdateForUser(ctx context.Context, userID, id uint, req *UpdateTeamRequest) (*domain.Team, error)
	DeleteForUser(ctx context.Context, userID, id uint) error
	GetForUser(ctx context.Context, userID, id uint) (*domain.Team, error)
	ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.Team, int64, error)
}

type service struct {
	repo domain.TeamRepository
}

var _ Service = (*service)(nil)

// NewService creates a new team service.
func NewService(repo domain.TeamRepository) *service {
	return &service{repo: repo}
}

func (s *service) CreateForUser(ctx context.Context, userID uint, req *CreateTeamRequest) (*domain.Team, error) {
	if userID == 0 || req == nil {
		return nil, domain.ErrInvalidInput
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	slug := normalizeTeamSlug(req.Slug)
	if slug == "" {
		slug = normalizeTeamSlug(name)
	}
	if slug == "" {
		return nil, domain.ErrInvalidInput
	}

	item := &domain.Team{
		Name:        name,
		Slug:        slug,
		OwnerUserID: userID,
		Plan:        domain.TeamPlanFree,
		Status:      domain.TeamStatusActive,
	}
	owner := &domain.TeamMember{
		UserID: userID,
		Role:   domain.TeamRoleOwner,
	}

	if err := s.repo.CreateWithOwner(ctx, item, owner); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *service) UpdateForUser(ctx context.Context, userID, id uint, req *UpdateTeamRequest) (*domain.Team, error) {
	if userID == 0 || req == nil {
		return nil, domain.ErrInvalidInput
	}

	item, err := s.repo.FindForUser(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		item.Name = name
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != domain.TeamStatusActive && status != domain.TeamStatusArchived {
			return nil, domain.ErrInvalidInput
		}
		item.Status = status
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *service) DeleteForUser(ctx context.Context, userID, id uint) error {
	if userID == 0 {
		return domain.ErrInvalidInput
	}
	if _, err := s.repo.FindForUser(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetForUser(ctx context.Context, userID, id uint) (*domain.Team, error) {
	if userID == 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.FindForUser(ctx, userID, id)
}

func (s *service) ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.Team, int64, error) {
	if userID == 0 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.repo.FindAllForUser(ctx, userID, page, pageSize)
}

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeTeamSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlugCharacters.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 120 {
		value = strings.Trim(value[:120], "-")
	}
	return value
}

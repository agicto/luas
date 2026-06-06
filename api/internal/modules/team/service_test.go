package team

import (
	"context"
	"testing"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeRepository struct {
	nextID  uint
	teams   map[uint]*domain.Team
	members map[uint][]*domain.TeamMember
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		nextID:  1,
		teams:   make(map[uint]*domain.Team),
		members: make(map[uint][]*domain.TeamMember),
	}
}

func (r *fakeRepository) CreateWithOwner(ctx context.Context, team *domain.Team, owner *domain.TeamMember) error {
	for _, existing := range r.teams {
		if existing.Slug == team.Slug {
			return domain.ErrTeamSlugAlreadyExists
		}
	}
	team.ID = r.nextID
	r.nextID++
	owner.ID = r.nextID
	r.nextID++
	owner.TeamID = team.ID
	r.teams[team.ID] = cloneTeam(team)
	r.members[team.ID] = append(r.members[team.ID], cloneTeamMember(owner))
	return nil
}

func (r *fakeRepository) Update(ctx context.Context, team *domain.Team) error {
	if _, ok := r.teams[team.ID]; !ok {
		return domain.ErrTeamNotFound
	}
	r.teams[team.ID] = cloneTeam(team)
	return nil
}

func (r *fakeRepository) Delete(ctx context.Context, id uint) error {
	delete(r.teams, id)
	delete(r.members, id)
	return nil
}

func (r *fakeRepository) FindByID(ctx context.Context, id uint) (*domain.Team, error) {
	team, ok := r.teams[id]
	if !ok {
		return nil, domain.ErrTeamNotFound
	}
	return cloneTeam(team), nil
}

func (r *fakeRepository) FindBySlug(ctx context.Context, slug string) (*domain.Team, error) {
	for _, team := range r.teams {
		if team.Slug == slug {
			return cloneTeam(team), nil
		}
	}
	return nil, domain.ErrTeamNotFound
}

func (r *fakeRepository) FindForUser(ctx context.Context, userID, teamID uint) (*domain.Team, error) {
	team, ok := r.teams[teamID]
	if !ok {
		return nil, domain.ErrTeamNotFound
	}
	for _, member := range r.members[teamID] {
		if member.UserID == userID {
			return cloneTeam(team), nil
		}
	}
	return nil, domain.ErrTeamNotFound
}

func (r *fakeRepository) FindAllForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.Team, int64, error) {
	items := make([]*domain.Team, 0)
	for teamID, members := range r.members {
		for _, member := range members {
			if member.UserID == userID {
				items = append(items, cloneTeam(r.teams[teamID]))
			}
		}
	}
	return items, int64(len(items)), nil
}

func cloneTeam(team *domain.Team) *domain.Team {
	if team == nil {
		return nil
	}
	copyTeam := *team
	return &copyTeam
}

func cloneTeamMember(member *domain.TeamMember) *domain.TeamMember {
	if member == nil {
		return nil
	}
	copyMember := *member
	return &copyMember
}

func TestServiceCreateForUserCreatesOwnerMembership(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo)

	created, err := service.CreateForUser(context.Background(), 42, &CreateTeamRequest{Name: "Acme Labs"})
	if err != nil {
		t.Fatalf("CreateForUser() error = %v", err)
	}

	if created.Slug != "acme-labs" {
		t.Fatalf("created.Slug = %q, want acme-labs", created.Slug)
	}
	if created.OwnerUserID != 42 {
		t.Fatalf("created.OwnerUserID = %d, want 42", created.OwnerUserID)
	}
	if created.Plan != domain.TeamPlanFree || created.Status != domain.TeamStatusActive {
		t.Fatalf("unexpected starter defaults: plan=%s status=%s", created.Plan, created.Status)
	}
	member := repo.members[created.ID][0]
	if member.UserID != 42 || member.Role != domain.TeamRoleOwner {
		t.Fatalf("unexpected owner member: %+v", member)
	}
}

func TestServiceRejectsDuplicateSlug(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo)
	ctx := context.Background()

	if _, err := service.CreateForUser(ctx, 1, &CreateTeamRequest{Name: "Acme"}); err != nil {
		t.Fatalf("CreateForUser() error = %v", err)
	}
	_, err := service.CreateForUser(ctx, 2, &CreateTeamRequest{Name: "Acme"})

	if err != domain.ErrTeamSlugAlreadyExists {
		t.Fatalf("CreateForUser() error = %v, want %v", err, domain.ErrTeamSlugAlreadyExists)
	}
}

func TestServiceScopesReadAndUpdateToMembers(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo)
	ctx := context.Background()

	created, err := service.CreateForUser(ctx, 7, &CreateTeamRequest{Name: "Core Team"})
	if err != nil {
		t.Fatalf("CreateForUser() error = %v", err)
	}

	_, err = service.GetForUser(ctx, 8, created.ID)
	if err != domain.ErrTeamNotFound {
		t.Fatalf("GetForUser() error = %v, want %v", err, domain.ErrTeamNotFound)
	}

	archived := domain.TeamStatusArchived
	updated, err := service.UpdateForUser(ctx, 7, created.ID, &UpdateTeamRequest{Status: &archived})
	if err != nil {
		t.Fatalf("UpdateForUser() error = %v", err)
	}
	if updated.Status != domain.TeamStatusArchived {
		t.Fatalf("updated.Status = %q, want archived", updated.Status)
	}
}

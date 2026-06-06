package access

import (
	"context"
	"testing"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeRepository struct {
	nextID      uint
	roles       map[uint]*domain.AccessRole
	memberships map[uint]map[uint]bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		nextID:      1,
		roles:       make(map[uint]*domain.AccessRole),
		memberships: make(map[uint]map[uint]bool),
	}
}

func (r *fakeRepository) grant(userID, teamID uint) {
	if r.memberships[userID] == nil {
		r.memberships[userID] = make(map[uint]bool)
	}
	r.memberships[userID][teamID] = true
}

func (r *fakeRepository) CreateRole(ctx context.Context, role *domain.AccessRole) error {
	for _, existing := range r.roles {
		if existing.TeamID == role.TeamID && existing.Slug == role.Slug {
			return domain.ErrAccessRoleSlugAlreadyExists
		}
	}
	role.ID = r.nextID
	r.nextID++
	r.roles[role.ID] = cloneRole(role)
	return nil
}

func (r *fakeRepository) UpdateRole(ctx context.Context, role *domain.AccessRole) error {
	if _, ok := r.roles[role.ID]; !ok {
		return domain.ErrAccessRoleNotFound
	}
	r.roles[role.ID] = cloneRole(role)
	return nil
}

func (r *fakeRepository) DeleteRole(ctx context.Context, teamID, roleID uint) error {
	role, ok := r.roles[roleID]
	if !ok || role.TeamID != teamID {
		return domain.ErrAccessRoleNotFound
	}
	delete(r.roles, roleID)
	return nil
}

func (r *fakeRepository) FindRoleForUser(ctx context.Context, userID, teamID, roleID uint) (*domain.AccessRole, error) {
	if ok, _ := r.UserCanAccessTeam(ctx, userID, teamID); !ok {
		return nil, domain.ErrAccessRoleNotFound
	}
	role, ok := r.roles[roleID]
	if !ok || role.TeamID != teamID {
		return nil, domain.ErrAccessRoleNotFound
	}
	return cloneRole(role), nil
}

func (r *fakeRepository) FindRolesForUser(ctx context.Context, userID, teamID uint, page, pageSize int) ([]*domain.AccessRole, int64, error) {
	if ok, _ := r.UserCanAccessTeam(ctx, userID, teamID); !ok {
		return nil, 0, domain.ErrTeamNotFound
	}
	items := make([]*domain.AccessRole, 0)
	for _, role := range r.roles {
		if role.TeamID == teamID {
			items = append(items, cloneRole(role))
		}
	}
	return items, int64(len(items)), nil
}

func (r *fakeRepository) UserCanAccessTeam(ctx context.Context, userID, teamID uint) (bool, error) {
	return r.memberships[userID][teamID], nil
}

func cloneRole(role *domain.AccessRole) *domain.AccessRole {
	if role == nil {
		return nil
	}
	copyRole := *role
	copyRole.Permissions = append([]string(nil), role.Permissions...)
	return &copyRole
}

func TestServiceCreatesTeamScopedRole(t *testing.T) {
	repo := newFakeRepository()
	repo.grant(7, 42)
	service := NewService(repo)

	role, err := service.CreateRole(context.Background(), 7, 42, &CreateRoleRequest{
		Name:        "Project Admin",
		Permissions: []string{"roles:manage", "teams:read", "roles:manage"},
	})
	if err != nil {
		t.Fatalf("CreateRole() error = %v", err)
	}

	if role.TeamID != 42 || role.Slug != "project-admin" {
		t.Fatalf("unexpected role: %+v", role)
	}
	if got, want := role.Permissions, []string{"roles:manage", "teams:read"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("role.Permissions = %#v, want %#v", got, want)
	}
}

func TestServiceRejectsUnknownPermission(t *testing.T) {
	repo := newFakeRepository()
	repo.grant(7, 42)
	service := NewService(repo)

	_, err := service.CreateRole(context.Background(), 7, 42, &CreateRoleRequest{
		Name:        "Bad",
		Permissions: []string{"unknown:permission"},
	})

	if err != domain.ErrInvalidInput {
		t.Fatalf("CreateRole() error = %v, want %v", err, domain.ErrInvalidInput)
	}
}

func TestServiceRequiresTeamMembership(t *testing.T) {
	service := NewService(newFakeRepository())

	_, err := service.CreateRole(context.Background(), 7, 42, &CreateRoleRequest{Name: "Admin"})

	if err != domain.ErrTeamNotFound {
		t.Fatalf("CreateRole() error = %v, want %v", err, domain.ErrTeamNotFound)
	}
}

func TestServiceProtectsSystemRoles(t *testing.T) {
	repo := newFakeRepository()
	repo.grant(7, 42)
	repo.roles[1] = &domain.AccessRole{ID: 1, TeamID: 42, Name: "Owner", Slug: "owner", System: true}
	service := NewService(repo)

	name := "Other"
	_, err := service.UpdateRole(context.Background(), 7, 42, 1, &UpdateRoleRequest{Name: &name})

	if err != domain.ErrPermissionDenied {
		t.Fatalf("UpdateRole() error = %v, want %v", err, domain.ErrPermissionDenied)
	}
}

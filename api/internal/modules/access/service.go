package access

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/zgiai/luas/api/internal/domain"
)

// Service defines the business interface for access control.
type Service interface {
	PermissionCatalog(ctx context.Context) []domain.AccessPermission
	CreateRole(ctx context.Context, userID, teamID uint, req *CreateRoleRequest) (*domain.AccessRole, error)
	UpdateRole(ctx context.Context, userID, teamID, roleID uint, req *UpdateRoleRequest) (*domain.AccessRole, error)
	DeleteRole(ctx context.Context, userID, teamID, roleID uint) error
	GetRole(ctx context.Context, userID, teamID, roleID uint) (*domain.AccessRole, error)
	ListRoles(ctx context.Context, userID, teamID uint, page, pageSize int) ([]*domain.AccessRole, int64, error)
}

type service struct {
	repo domain.AccessRepository
}

var _ Service = (*service)(nil)

// NewService creates a new access service.
func NewService(repo domain.AccessRepository) *service {
	return &service{repo: repo}
}

func (s *service) PermissionCatalog(ctx context.Context) []domain.AccessPermission {
	return slices.Clone(defaultPermissionCatalog)
}

func (s *service) CreateRole(ctx context.Context, userID, teamID uint, req *CreateRoleRequest) (*domain.AccessRole, error) {
	if req == nil {
		return nil, domain.ErrInvalidInput
	}
	if err := s.ensureTeamAccess(ctx, userID, teamID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	slug := normalizeAccessSlug(req.Slug)
	if slug == "" {
		slug = normalizeAccessSlug(name)
	}
	permissions, err := normalizePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}

	role := &domain.AccessRole{
		TeamID:      teamID,
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
		Permissions: permissions,
	}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) UpdateRole(ctx context.Context, userID, teamID, roleID uint, req *UpdateRoleRequest) (*domain.AccessRole, error) {
	if req == nil {
		return nil, domain.ErrInvalidInput
	}
	role, err := s.repo.FindRoleForUser(ctx, userID, teamID, roleID)
	if err != nil {
		return nil, err
	}
	if role.System {
		return nil, domain.ErrPermissionDenied
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		role.Name = name
	}
	if req.Description != nil {
		role.Description = strings.TrimSpace(*req.Description)
	}
	if req.Permissions != nil {
		permissions, err := normalizePermissions(req.Permissions)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
	}
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) DeleteRole(ctx context.Context, userID, teamID, roleID uint) error {
	role, err := s.repo.FindRoleForUser(ctx, userID, teamID, roleID)
	if err != nil {
		return err
	}
	if role.System {
		return domain.ErrPermissionDenied
	}
	return s.repo.DeleteRole(ctx, teamID, roleID)
}

func (s *service) GetRole(ctx context.Context, userID, teamID, roleID uint) (*domain.AccessRole, error) {
	return s.repo.FindRoleForUser(ctx, userID, teamID, roleID)
}

func (s *service) ListRoles(ctx context.Context, userID, teamID uint, page, pageSize int) ([]*domain.AccessRole, int64, error) {
	if err := s.ensureTeamAccess(ctx, userID, teamID); err != nil {
		return nil, 0, err
	}
	return s.repo.FindRolesForUser(ctx, userID, teamID, page, pageSize)
}

func (s *service) ensureTeamAccess(ctx context.Context, userID, teamID uint) error {
	if userID == 0 || teamID == 0 {
		return domain.ErrInvalidInput
	}
	ok, err := s.repo.UserCanAccessTeam(ctx, userID, teamID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrTeamNotFound
	}
	return nil
}

var defaultPermissionCatalog = []domain.AccessPermission{
	{Key: "teams:read", Label: "Read teams", Category: "teams"},
	{Key: "teams:update", Label: "Update teams", Category: "teams"},
	{Key: "members:invite", Label: "Invite members", Category: "members"},
	{Key: "members:manage", Label: "Manage members", Category: "members"},
	{Key: "roles:manage", Label: "Manage roles", Category: "access"},
	{Key: "api_keys:manage", Label: "Manage API keys", Category: "api_keys"},
	{Key: "audit:read", Label: "Read audit logs", Category: "audit"},
}

var accessSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeAccessSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = accessSlugRe.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 120 {
		value = strings.Trim(value[:120], "-")
	}
	return value
}

func normalizePermissions(values []string) ([]string, error) {
	catalog := make(map[string]struct{}, len(defaultPermissionCatalog))
	for _, item := range defaultPermissionCatalog {
		catalog[item.Key] = struct{}{}
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := catalog[value]; !ok {
			return nil, domain.ErrInvalidInput
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	return result, nil
}

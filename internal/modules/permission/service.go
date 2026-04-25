package permission

import (
	"context"

	"github.com/zgiai/zgo/internal/domain"
)

// Service defines the interface for permission operations.
type Service interface {
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*domain.Role, error)
	UpdateRole(ctx context.Context, id uint, req *UpdateRoleRequest) (*domain.Role, error)
	DeleteRole(ctx context.Context, id uint) error
	GetRole(ctx context.Context, id uint) (*domain.Role, error)
	ListRoles(ctx context.Context) ([]*domain.Role, error)

	AssignRoleToUser(ctx context.Context, userID, roleID uint) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error
	GetUserRoles(ctx context.Context, userID uint) ([]*domain.Role, error)

	HasPermission(ctx context.Context, userID uint, permission string) (bool, error)
	GetRolePermissions(ctx context.Context, roleID uint) ([]*domain.Permission, error)

	AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error
	ListPermissions(ctx context.Context) ([]*domain.Permission, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) CreateRole(ctx context.Context, req *CreateRoleRequest) (*domain.Role, error) {
	role := req.toDomain()
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) UpdateRole(ctx context.Context, id uint, req *UpdateRoleRequest) (*domain.Role, error) {
	role, err := s.repo.FindRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	req.applyTo(role)

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) DeleteRole(ctx context.Context, id uint) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *service) GetRole(ctx context.Context, id uint) (*domain.Role, error) {
	return s.repo.FindRoleByID(ctx, id)
}

func (s *service) ListRoles(ctx context.Context) ([]*domain.Role, error) {
	return s.repo.FindAllRoles(ctx)
}

func (s *service) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	return s.repo.AssignRoleToUser(ctx, userID, roleID)
}

func (s *service) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	return s.repo.RemoveRoleFromUser(ctx, userID, roleID)
}

func (s *service) GetUserRoles(ctx context.Context, userID uint) ([]*domain.Role, error) {
	return s.repo.FindRolesByUserID(ctx, userID)
}

func (s *service) HasPermission(ctx context.Context, userID uint, permission string) (bool, error) {
	return s.repo.HasPermission(ctx, userID, permission)
}

func (s *service) GetRolePermissions(ctx context.Context, roleID uint) ([]*domain.Permission, error) {
	return s.repo.FindPermissionsByRoleID(ctx, roleID)
}

func (s *service) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	return s.repo.AssignPermissionToRole(ctx, roleID, permissionID)
}

func (s *service) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	return s.repo.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (s *service) ListPermissions(ctx context.Context) ([]*domain.Permission, error) {
	return s.repo.FindAllPermissions(ctx)
}

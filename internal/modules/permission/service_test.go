package permission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgiai/zgo/internal/domain"
)

type fakeRepo struct {
	createRoleFn            func(context.Context, *domain.Role) error
	updateRoleFn            func(context.Context, *domain.Role) error
	deleteRoleFn            func(context.Context, uint) error
	findRoleByIDFn          func(context.Context, uint) (*domain.Role, error)
	findRoleByNameFn        func(context.Context, string) (*domain.Role, error)
	findAllRolesFn          func(context.Context) ([]*domain.Role, error)
	findDefaultRoleFn       func(context.Context) (*domain.Role, error)
	createPermissionFn      func(context.Context, *domain.Permission) error
	findAllPermissionsFn    func(context.Context) ([]*domain.Permission, error)
	findPermissionsByRoleFn func(context.Context, uint) ([]*domain.Permission, error)
	assignRoleToUserFn      func(context.Context, uint, uint) error
	removeRoleFromUserFn    func(context.Context, uint, uint) error
	findRolesByUserIDFn     func(context.Context, uint) ([]*domain.Role, error)
	hasPermissionFn         func(context.Context, uint, string) (bool, error)
}

func (r *fakeRepo) CreateRole(ctx context.Context, role *domain.Role) error {
	if r.createRoleFn != nil {
		return r.createRoleFn(ctx, role)
	}
	return nil
}

func (r *fakeRepo) UpdateRole(ctx context.Context, role *domain.Role) error {
	if r.updateRoleFn != nil {
		return r.updateRoleFn(ctx, role)
	}
	return nil
}

func (r *fakeRepo) DeleteRole(ctx context.Context, id uint) error {
	if r.deleteRoleFn != nil {
		return r.deleteRoleFn(ctx, id)
	}
	return nil
}

func (r *fakeRepo) FindRoleByID(ctx context.Context, id uint) (*domain.Role, error) {
	if r.findRoleByIDFn != nil {
		return r.findRoleByIDFn(ctx, id)
	}
	return nil, domain.ErrRoleNotFound
}

func (r *fakeRepo) FindRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	if r.findRoleByNameFn != nil {
		return r.findRoleByNameFn(ctx, name)
	}
	return nil, domain.ErrRoleNotFound
}

func (r *fakeRepo) FindAllRoles(ctx context.Context) ([]*domain.Role, error) {
	if r.findAllRolesFn != nil {
		return r.findAllRolesFn(ctx)
	}
	return nil, nil
}

func (r *fakeRepo) FindDefaultRole(ctx context.Context) (*domain.Role, error) {
	if r.findDefaultRoleFn != nil {
		return r.findDefaultRoleFn(ctx)
	}
	return nil, domain.ErrRoleNotFound
}

func (r *fakeRepo) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	if r.createPermissionFn != nil {
		return r.createPermissionFn(ctx, permission)
	}
	return nil
}

func (r *fakeRepo) FindAllPermissions(ctx context.Context) ([]*domain.Permission, error) {
	if r.findAllPermissionsFn != nil {
		return r.findAllPermissionsFn(ctx)
	}
	return nil, nil
}

func (r *fakeRepo) FindPermissionsByModule(ctx context.Context, module string) ([]*domain.Permission, error) {
	return nil, nil
}

func (r *fakeRepo) FindPermissionsByRoleID(ctx context.Context, roleID uint) ([]*domain.Permission, error) {
	if r.findPermissionsByRoleFn != nil {
		return r.findPermissionsByRoleFn(ctx, roleID)
	}
	return nil, nil
}

func (r *fakeRepo) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	return nil
}

func (r *fakeRepo) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	return nil
}

func (r *fakeRepo) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	if r.assignRoleToUserFn != nil {
		return r.assignRoleToUserFn(ctx, userID, roleID)
	}
	return nil
}

func (r *fakeRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	if r.removeRoleFromUserFn != nil {
		return r.removeRoleFromUserFn(ctx, userID, roleID)
	}
	return nil
}

func (r *fakeRepo) FindRolesByUserID(ctx context.Context, userID uint) ([]*domain.Role, error) {
	if r.findRolesByUserIDFn != nil {
		return r.findRolesByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (r *fakeRepo) HasPermission(ctx context.Context, userID uint, permissionName string) (bool, error) {
	if r.hasPermissionFn != nil {
		return r.hasPermissionFn(ctx, userID, permissionName)
	}
	return false, nil
}

func TestServiceCreateRoleMapsRequestToDomain(t *testing.T) {
	var created *domain.Role

	svc := NewService(&fakeRepo{
		createRoleFn: func(_ context.Context, role *domain.Role) error {
			created = role
			role.ID = 9
			role.CreatedAt = time.Unix(10, 0)
			return nil
		},
	})

	role, err := svc.CreateRole(context.Background(), &CreateRoleRequest{
		Name:        "admin",
		DisplayName: "Administrator",
		Description: "full access",
		IsDefault:   true,
	})

	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.NotNil(t, created)
	assert.Equal(t, "admin", created.Name)
	assert.Equal(t, "Administrator", created.DisplayName)
	assert.Equal(t, "full access", created.Description)
	assert.True(t, created.IsDefault)
	assert.Equal(t, uint(9), role.ID)
}

func TestServiceUpdateRoleAppliesOnlyProvidedFields(t *testing.T) {
	var updated *domain.Role

	svc := NewService(&fakeRepo{
		findRoleByIDFn: func(context.Context, uint) (*domain.Role, error) {
			return &domain.Role{
				ID:          5,
				Name:        "user",
				DisplayName: "User",
				Description: "original",
				IsDefault:   true,
			}, nil
		},
		updateRoleFn: func(_ context.Context, role *domain.Role) error {
			updated = role
			return nil
		},
	})

	role, err := svc.UpdateRole(context.Background(), 5, &UpdateRoleRequest{
		DisplayName: "Member",
	})

	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, "user", updated.Name)
	assert.Equal(t, "Member", updated.DisplayName)
	assert.Equal(t, "original", updated.Description)
	assert.True(t, updated.IsDefault)
	assert.Equal(t, "Member", role.DisplayName)
}

func TestServiceUpdateRoleReturnsLookupError(t *testing.T) {
	svc := NewService(&fakeRepo{
		findRoleByIDFn: func(context.Context, uint) (*domain.Role, error) {
			return nil, domain.ErrRoleNotFound
		},
	})

	role, err := svc.UpdateRole(context.Background(), 5, &UpdateRoleRequest{
		Name: "admin",
	})

	assert.Nil(t, role)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestServiceListPermissionsReturnsRepositoryError(t *testing.T) {
	svc := NewService(&fakeRepo{
		findAllPermissionsFn: func(context.Context) ([]*domain.Permission, error) {
			return nil, errors.New("db unavailable")
		},
	})

	permissions, err := svc.ListPermissions(context.Background())

	assert.Nil(t, permissions)
	assert.EqualError(t, err, "db unavailable")
}

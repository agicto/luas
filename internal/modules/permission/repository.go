package permission

import (
	"context"
	"errors"

	"github.com/zgiai/zgo/internal/domain"
	"gorm.io/gorm"
)

// Repository defines the interface for permission data operations.
type Repository interface {
	CreateRole(ctx context.Context, role *domain.Role) error
	UpdateRole(ctx context.Context, role *domain.Role) error
	DeleteRole(ctx context.Context, id uint) error
	FindRoleByID(ctx context.Context, id uint) (*domain.Role, error)
	FindRoleByName(ctx context.Context, name string) (*domain.Role, error)
	FindAllRoles(ctx context.Context) ([]*domain.Role, error)
	FindDefaultRole(ctx context.Context) (*domain.Role, error)

	CreatePermission(ctx context.Context, permission *domain.Permission) error
	FindAllPermissions(ctx context.Context) ([]*domain.Permission, error)
	FindPermissionsByModule(ctx context.Context, module string) ([]*domain.Permission, error)
	FindPermissionsByRoleID(ctx context.Context, roleID uint) ([]*domain.Permission, error)

	AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error

	AssignRoleToUser(ctx context.Context, userID, roleID uint) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error
	FindRolesByUserID(ctx context.Context, userID uint) ([]*domain.Role, error)
	HasPermission(ctx context.Context, userID uint, permissionName string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreateRole(ctx context.Context, role *domain.Role) error {
	po := newRolePO(role)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	role.ID = po.ID
	role.CreatedAt = po.CreatedAt
	role.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) UpdateRole(ctx context.Context, role *domain.Role) error {
	po := newRolePO(role)
	if err := r.db.WithContext(ctx).Save(po).Error; err != nil {
		return err
	}
	role.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) DeleteRole(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&RolePO{}, id).Error
}

func (r *repository) FindRoleByID(ctx context.Context, id uint) (*domain.Role, error) {
	var po RolePO
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	var po RolePO
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindAllRoles(ctx context.Context) ([]*domain.Role, error) {
	var poList []*RolePO
	if err := r.db.WithContext(ctx).Find(&poList).Error; err != nil {
		return nil, err
	}
	return toRoleDomainList(poList), nil
}

func (r *repository) FindDefaultRole(ctx context.Context) (*domain.Role, error) {
	var po RolePO
	if err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	po := newPermissionPO(permission)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	permission.ID = po.ID
	permission.CreatedAt = po.CreatedAt
	permission.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) FindAllPermissions(ctx context.Context) ([]*domain.Permission, error) {
	var poList []*PermissionPO
	if err := r.db.WithContext(ctx).Find(&poList).Error; err != nil {
		return nil, err
	}
	return toPermissionDomainList(poList), nil
}

func (r *repository) FindPermissionsByModule(ctx context.Context, module string) ([]*domain.Permission, error) {
	var poList []*PermissionPO
	if err := r.db.WithContext(ctx).Where("module = ?", module).Find(&poList).Error; err != nil {
		return nil, err
	}
	return toPermissionDomainList(poList), nil
}

func (r *repository) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	po := &RolePermissionPO{RoleID: roleID, PermissionID: permissionID}
	return r.db.WithContext(ctx).FirstOrCreate(po, po).Error
}

func (r *repository) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	return r.db.WithContext(ctx).Where("role_id = ? AND permission_id = ?", roleID, permissionID).Delete(&RolePermissionPO{}).Error
}

func (r *repository) FindPermissionsByRoleID(ctx context.Context, roleID uint) ([]*domain.Permission, error) {
	var poList []*PermissionPO
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&poList).Error
	if err != nil {
		return nil, err
	}
	return toPermissionDomainList(poList), nil
}

func (r *repository) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	po := &UserRolePO{UserID: userID, RoleID: roleID}
	return r.db.WithContext(ctx).FirstOrCreate(po, po).Error
}

func (r *repository) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&UserRolePO{}).Error
}

func (r *repository) FindRolesByUserID(ctx context.Context, userID uint) ([]*domain.Role, error) {
	var poList []*RolePO
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&poList).Error
	if err != nil {
		return nil, err
	}
	return toRoleDomainList(poList), nil
}

func (r *repository) HasPermission(ctx context.Context, userID uint, permissionName string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ? AND permissions.name = ?", userID, permissionName).
		Count(&count).Error
	return count > 0, err
}

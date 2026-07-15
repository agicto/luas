package permission

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
)

type repository struct {
	db *gorm.DB
}

var _ domain.PermissionRepository = (*repository)(nil)

// NewRepository creates the permission persistence adapter.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return infradatabase.ResolveContextDB(ctx, r.db), nil
}

func (r *repository) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if operation == nil {
		return domain.ErrInvalidInput
	}
	if _, active := infradatabase.TransactionFromContext(ctx); active {
		return operation(ctx)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return operation(infradatabase.ContextWithTransaction(ctx, tx))
	})
}

func (r *repository) Effective(ctx context.Context, expected domain.OrganizationContext) (*domain.PermissionContext, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if !expected.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	if _, active := infradatabase.TransactionFromContext(ctx); active {
		return r.effectiveForUpdate(ctx, db, expected)
	}

	type effectiveRow struct {
		MembershipRole string  `gorm:"column:membership_role"`
		AccessRoleID   *uint   `gorm:"column:access_role_id"`
		Permission     *string `gorm:"column:permission"`
	}
	var rows []effectiveRow
	query := db.Table("organization_memberships AS memberships").
		Select("memberships.role AS membership_role, roles.id AS access_role_id, grants.permission AS permission").
		Joins("LEFT JOIN permission_role_assignments AS assignments ON assignments.organization_id = memberships.organization_id AND assignments.membership_id = memberships.id").
		Joins("LEFT JOIN permission_roles AS roles ON roles.organization_id = memberships.organization_id AND roles.id = assignments.access_role_id").
		Joins("LEFT JOIN permission_role_grants AS grants ON grants.access_role_id = roles.id").
		Where(
			"memberships.id = ? AND memberships.organization_id = ? AND memberships.user_id = ?",
			expected.MembershipID,
			expected.OrganizationID,
			expected.UserID,
		).
		Order("roles.id ASC, grants.permission ASC")
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrOrganizationNotFound
	}

	role := domain.OrganizationRole(rows[0].MembershipRole)
	if !role.IsValid() {
		return nil, domain.ErrServiceUnavailable
	}
	roleSet := make(map[uint]struct{})
	permissionSet := make(map[domain.PermissionKey]struct{})
	for _, row := range rows {
		if row.MembershipRole != rows[0].MembershipRole {
			return nil, domain.ErrServiceUnavailable
		}
		if row.AccessRoleID != nil {
			roleSet[*row.AccessRoleID] = struct{}{}
		}
		if row.Permission != nil {
			permissionSet[domain.PermissionKey(*row.Permission)] = struct{}{}
		}
	}

	accessRoleIDs := make([]uint, 0, len(roleSet))
	for roleID := range roleSet {
		accessRoleIDs = append(accessRoleIDs, roleID)
	}
	slices.Sort(accessRoleIDs)
	permissions := make([]domain.PermissionKey, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	slices.Sort(permissions)
	return &domain.PermissionContext{
		OrganizationID:   expected.OrganizationID,
		MembershipID:     expected.MembershipID,
		OrganizationRole: role,
		IsOwner:          role == domain.OrganizationRoleOwner,
		AccessRoleIDs:    accessRoleIDs,
		Permissions:      permissions,
	}, nil
}

func (r *repository) effectiveForUpdate(
	ctx context.Context,
	db *gorm.DB,
	expected domain.OrganizationContext,
) (*domain.PermissionContext, error) {
	type membershipRow struct {
		ID   uint
		Role string
	}
	var membership membershipRow
	query := db.Table("organization_memberships").
		Select("id, role").
		Where(
			"id = ? AND organization_id = ? AND user_id = ?",
			expected.MembershipID,
			expected.OrganizationID,
			expected.UserID,
		)
	query = lockWhenTransactional(ctx, query)
	if err := query.Take(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, err
	}
	role := domain.OrganizationRole(membership.Role)
	if !role.IsValid() {
		return nil, domain.ErrServiceUnavailable
	}

	roleIDs := make([]uint, 0)
	assignmentQuery := db.Model(&AccessRoleAssignmentPO{}).
		Where("organization_id = ? AND membership_id = ?", expected.OrganizationID, expected.MembershipID).
		Order("access_role_id ASC")
	assignmentQuery = lockWhenTransactional(ctx, assignmentQuery)
	if err := assignmentQuery.Pluck("access_role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return &domain.PermissionContext{
			OrganizationID:   expected.OrganizationID,
			MembershipID:     expected.MembershipID,
			OrganizationRole: role,
			IsOwner:          role == domain.OrganizationRoleOwner,
			AccessRoleIDs:    []uint{},
			Permissions:      []domain.PermissionKey{},
		}, nil
	}

	lockedRoleIDs := make([]uint, 0, len(roleIDs))
	roleQuery := db.Model(&AccessRolePO{}).
		Where("organization_id = ? AND id IN ?", expected.OrganizationID, roleIDs).
		Order("id ASC")
	roleQuery = lockWhenTransactional(ctx, roleQuery)
	if err := roleQuery.Pluck("id", &lockedRoleIDs).Error; err != nil {
		return nil, err
	}
	if len(lockedRoleIDs) != len(roleIDs) {
		return nil, domain.ErrServiceUnavailable
	}

	permissionValues := make([]string, 0)
	if err := db.Model(&AccessRolePermissionPO{}).
		Where("access_role_id IN ?", lockedRoleIDs).
		Distinct().
		Order("permission ASC").
		Pluck("permission", &permissionValues).Error; err != nil {
		return nil, err
	}
	permissions := make([]domain.PermissionKey, len(permissionValues))
	for index, permission := range permissionValues {
		permissions[index] = domain.PermissionKey(permission)
	}
	return &domain.PermissionContext{
		OrganizationID:   expected.OrganizationID,
		MembershipID:     expected.MembershipID,
		OrganizationRole: role,
		IsOwner:          role == domain.OrganizationRoleOwner,
		AccessRoleIDs:    lockedRoleIDs,
		Permissions:      permissions,
	}, nil
}

func (r *repository) ListRoles(ctx context.Context, organizationID uint, page, pageSize int) ([]*domain.AccessRole, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || page < 1 || pageSize < 1 {
		return nil, 0, domain.ErrInvalidInput
	}

	query := db.Model(&AccessRolePO{}).Where("organization_id = ?", organizationID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var roles []*AccessRolePO
	if err := query.
		Preload("Permissions", func(preload *gorm.DB) *gorm.DB {
			return preload.Order("permission ASC")
		}).
		Order("name ASC, id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	return accessRoleDomainList(roles), total, nil
}

func (r *repository) FindRole(ctx context.Context, organizationID, roleID uint) (*domain.AccessRole, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 || roleID == 0 {
		return nil, domain.ErrInvalidInput
	}

	var role AccessRolePO
	query := db.Preload("Permissions", func(preload *gorm.DB) *gorm.DB {
		return preload.Order("permission ASC")
	}).Where("organization_id = ? AND id = ?", organizationID, roleID)
	query = lockWhenTransactional(ctx, query)
	if err := query.First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAccessRoleNotFound
		}
		return nil, err
	}
	return role.toDomain(), nil
}

func (r *repository) FindRoles(ctx context.Context, organizationID uint, roleIDs []uint) ([]*domain.AccessRole, error) {
	if len(roleIDs) == 0 {
		return []*domain.AccessRole{}, nil
	}
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 {
		return nil, domain.ErrInvalidInput
	}

	var roles []*AccessRolePO
	query := db.Preload("Permissions", func(preload *gorm.DB) *gorm.DB {
		return preload.Order("permission ASC")
	}).Where("organization_id = ? AND id IN ?", organizationID, roleIDs).Order("id ASC")
	query = lockWhenTransactional(ctx, query)
	if err := query.Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) != len(roleIDs) {
		return nil, domain.ErrAccessRoleNotFound
	}
	return accessRoleDomainList(roles), nil
}

func (r *repository) CreateRole(ctx context.Context, role *domain.AccessRole) error {
	if role == nil || role.ID != 0 || role.OrganizationID == 0 {
		return domain.ErrInvalidInput
	}
	return r.WithinTransaction(ctx, func(transactionContext context.Context) error {
		db, err := r.withContext(transactionContext)
		if err != nil {
			return err
		}
		po := newAccessRolePO(role)
		if err := db.Omit(clause.Associations).Create(po).Error; err != nil {
			if isPermissionUniqueViolation(err) {
				return domain.ErrAccessRoleSlugAlreadyExists
			}
			return err
		}
		if err := createRolePermissionRows(db, po.ID, role.Permissions); err != nil {
			return err
		}
		role.ID = po.ID
		role.CreatedAt = po.CreatedAt
		role.UpdatedAt = po.UpdatedAt
		return nil
	})
}

func (r *repository) UpdateRole(ctx context.Context, role *domain.AccessRole) error {
	if role == nil || role.ID == 0 || role.OrganizationID == 0 {
		return domain.ErrInvalidInput
	}
	return r.WithinTransaction(ctx, func(transactionContext context.Context) error {
		db, err := r.withContext(transactionContext)
		if err != nil {
			return err
		}
		updatedAt := time.Now().UTC()
		result := db.Model(&AccessRolePO{}).
			Where("organization_id = ? AND id = ?", role.OrganizationID, role.ID).
			Updates(map[string]any{"name": role.Name, "updated_at": updatedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrAccessRoleNotFound
		}
		if err := db.Where("access_role_id = ?", role.ID).Delete(&AccessRolePermissionPO{}).Error; err != nil {
			return err
		}
		if err := createRolePermissionRows(db, role.ID, role.Permissions); err != nil {
			return err
		}
		role.UpdatedAt = updatedAt
		return nil
	})
}

func (r *repository) DeleteRole(ctx context.Context, organizationID, roleID uint) error {
	if organizationID == 0 || roleID == 0 {
		return domain.ErrInvalidInput
	}
	return r.WithinTransaction(ctx, func(transactionContext context.Context) error {
		db, err := r.withContext(transactionContext)
		if err != nil {
			return err
		}
		if err := db.Where("access_role_id = ?", roleID).Delete(&AccessRoleAssignmentPO{}).Error; err != nil {
			return err
		}
		if err := db.Where("access_role_id = ?", roleID).Delete(&AccessRolePermissionPO{}).Error; err != nil {
			return err
		}
		result := db.Where("organization_id = ? AND id = ?", organizationID, roleID).Delete(&AccessRolePO{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrAccessRoleNotFound
		}
		return nil
	})
}

func (r *repository) MemberRoleIDs(ctx context.Context, organizationID, memberID uint) ([]uint, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 || memberID == 0 {
		return nil, domain.ErrInvalidInput
	}

	type membershipRow struct{ ID uint }
	var membership membershipRow
	membershipQuery := db.Table("organization_memberships").
		Select("id").
		Where("organization_id = ? AND id = ?", organizationID, memberID)
	membershipQuery = lockWhenTransactional(ctx, membershipQuery)
	if err := membershipQuery.Take(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrganizationMemberNotFound
		}
		return nil, err
	}

	roleIDs := make([]uint, 0)
	query := db.Model(&AccessRoleAssignmentPO{}).
		Where("organization_id = ? AND membership_id = ?", organizationID, memberID).
		Order("access_role_id ASC")
	query = lockWhenTransactional(ctx, query)
	if err := query.Pluck("access_role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	return roleIDs, nil
}

func (r *repository) ReplaceMemberRoleIDs(ctx context.Context, organizationID, memberID uint, roleIDs []uint) error {
	if organizationID == 0 || memberID == 0 {
		return domain.ErrInvalidInput
	}
	return r.WithinTransaction(ctx, func(transactionContext context.Context) error {
		db, err := r.withContext(transactionContext)
		if err != nil {
			return err
		}
		if err := db.Where(
			"organization_id = ? AND membership_id = ?",
			organizationID,
			memberID,
		).Delete(&AccessRoleAssignmentPO{}).Error; err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		rows := make([]AccessRoleAssignmentPO, len(roleIDs))
		for index, roleID := range roleIDs {
			rows[index] = AccessRoleAssignmentPO{
				OrganizationID: organizationID,
				MembershipID:   memberID,
				AccessRoleID:   roleID,
			}
		}
		return db.Omit(clause.Associations).Create(&rows).Error
	})
}

func createRolePermissionRows(db *gorm.DB, roleID uint, permissions []domain.PermissionKey) error {
	if len(permissions) == 0 {
		return nil
	}
	rows := make([]AccessRolePermissionPO, len(permissions))
	for index, permission := range permissions {
		rows[index] = AccessRolePermissionPO{AccessRoleID: roleID, Permission: string(permission)}
	}
	return db.Omit(clause.Associations).Create(&rows).Error
}

func accessRoleDomainList(values []*AccessRolePO) []*domain.AccessRole {
	roles := make([]*domain.AccessRole, len(values))
	for index, value := range values {
		roles[index] = value.toDomain()
	}
	return roles
}

func lockWhenTransactional(ctx context.Context, query *gorm.DB) *gorm.DB {
	if query == nil {
		return nil
	}
	if _, active := infradatabase.TransactionFromContext(ctx); !active {
		return query
	}
	if query.Dialector == nil || query.Name() == "sqlite" {
		return query
	}
	return query.Clauses(clause.Locking{Strength: "UPDATE"})
}

func isPermissionUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var stateError interface{ SQLState() string }
	if errors.As(err, &stateError) && stateError.SQLState() == "23505" {
		return true
	}
	message := strings.ToLower(fmt.Sprint(err))
	return strings.Contains(message, "unique constraint failed") || strings.Contains(message, "duplicate key")
}

package permission

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/zgiai/luas/api/internal/domain"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

// Service defines organization-scoped permission management and checks.
type Service interface {
	domain.PermissionAuthorizer
	Catalog(ctx context.Context, organization domain.OrganizationContext) ([]domain.PermissionKey, error)
	ListRoles(ctx context.Context, organization domain.OrganizationContext, page, pageSize int) ([]*domain.AccessRole, int64, error)
	GetRole(ctx context.Context, organization domain.OrganizationContext, roleID uint) (*domain.AccessRole, error)
	CreateRole(ctx context.Context, organization domain.OrganizationContext, request *CreateAccessRoleRequest) (*domain.AccessRole, error)
	UpdateRole(ctx context.Context, organization domain.OrganizationContext, roleID uint, request *UpdateAccessRoleRequest) (*domain.AccessRole, error)
	DeleteRole(ctx context.Context, organization domain.OrganizationContext, roleID uint) error
	MemberRoles(ctx context.Context, organization domain.OrganizationContext, memberID uint) (*MemberAccessRolesResponse, error)
	ReplaceMemberRoles(ctx context.Context, organization domain.OrganizationContext, memberID uint, request *ReplaceMemberAccessRolesRequest) (*MemberAccessRolesResponse, error)
}

type service struct {
	repo    domain.PermissionRepository
	catalog *Catalog
}

var (
	_ Service                     = (*service)(nil)
	_ domain.PermissionAuthorizer = (*service)(nil)
)

// NewService creates the permission service.
func NewService(repo domain.PermissionRepository, catalog *Catalog) *service {
	return &service{repo: repo, catalog: catalog}
}

// Effective returns current persisted grants and expands owner bypass to the catalog.
func (s *service) Effective(ctx context.Context, organization domain.OrganizationContext) (*domain.PermissionContext, error) {
	if s == nil || s.repo == nil || s.catalog == nil {
		return nil, domain.ErrServiceUnavailable
	}
	if !organization.IsValid() {
		return nil, domain.ErrOrganizationContextRequired
	}
	effective, err := s.repo.Effective(ctx, organization)
	if err != nil {
		return nil, err
	}
	if effective == nil || effective.OrganizationID != organization.OrganizationID || effective.MembershipID != organization.MembershipID {
		return nil, domain.ErrServiceUnavailable
	}
	if effective.IsOwner {
		effective.Permissions = s.catalog.Keys()
		return effective, nil
	}
	for _, permission := range effective.Permissions {
		if !s.catalog.Contains(permission) {
			return nil, fmt.Errorf("persisted permission %q is not registered: %w", permission, domain.ErrServiceUnavailable)
		}
	}
	return effective, nil
}

// Authorize performs one exact, fail-closed permission check.
func (s *service) Authorize(ctx context.Context, organization domain.OrganizationContext, permission domain.PermissionKey) error {
	if s == nil || s.catalog == nil || !s.catalog.Contains(permission) {
		return fmt.Errorf("permission %q is not registered: %w", permission, domain.ErrServiceUnavailable)
	}
	effective, err := s.Effective(ctx, organization)
	if err != nil {
		return err
	}
	if effective.IsOwner || effective.Has(permission) {
		return nil
	}
	return domain.ErrPermissionDenied
}

func (s *service) Catalog(ctx context.Context, organization domain.OrganizationContext) ([]domain.PermissionKey, error) {
	if err := s.Authorize(ctx, organization, PermissionRolesRead); err != nil {
		return nil, err
	}
	return s.catalog.Keys(), nil
}

func (s *service) ListRoles(ctx context.Context, organization domain.OrganizationContext, page, pageSize int) ([]*domain.AccessRole, int64, error) {
	if err := s.Authorize(ctx, organization, PermissionRolesRead); err != nil {
		return nil, 0, err
	}
	roles, total, err := s.repo.ListRoles(ctx, organization.OrganizationID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if err := s.validateStoredRoles(roles); err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func (s *service) GetRole(ctx context.Context, organization domain.OrganizationContext, roleID uint) (*domain.AccessRole, error) {
	if roleID == 0 {
		return nil, domain.ErrInvalidInput
	}
	if err := s.Authorize(ctx, organization, PermissionRolesRead); err != nil {
		return nil, err
	}
	role, err := s.repo.FindRole(ctx, organization.OrganizationID, roleID)
	if err != nil {
		return nil, err
	}
	if err := s.validateStoredRole(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) CreateRole(ctx context.Context, organization domain.OrganizationContext, request *CreateAccessRoleRequest) (*domain.AccessRole, error) {
	role, err := s.roleFromCreateRequest(organization.OrganizationID, request)
	if err != nil {
		return nil, err
	}
	err = s.withAuthorizedTransaction(ctx, organization, PermissionRolesManage, func(transactionContext context.Context, effective *domain.PermissionContext) error {
		if !dominatesRoles(effective, role) {
			return domain.ErrPermissionDenied
		}
		return s.repo.CreateRole(transactionContext, role)
	})
	if err != nil {
		return nil, fmt.Errorf("create access role: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "create",
		Resource:   "access-roles",
		TargetType: "access_role",
		TargetID:   strconv.FormatUint(uint64(role.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"organization_id": organization.OrganizationID,
			"permissions":     role.Permissions,
		},
	})
	return role, nil
}

func (s *service) UpdateRole(ctx context.Context, organization domain.OrganizationContext, roleID uint, request *UpdateAccessRoleRequest) (*domain.AccessRole, error) {
	if roleID == 0 || request == nil || len(request.validationErrors()) > 0 {
		return nil, domain.ErrInvalidInput
	}
	permissions, err := s.normalizePermissions(request.Permissions)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(request.Name)
	var role *domain.AccessRole
	var beforeName string
	var beforePermissions []domain.PermissionKey
	err = s.withAuthorizedTransaction(ctx, organization, PermissionRolesManage, func(transactionContext context.Context, effective *domain.PermissionContext) error {
		current, findErr := s.repo.FindRole(transactionContext, organization.OrganizationID, roleID)
		if findErr != nil {
			return findErr
		}
		if validationErr := s.validateStoredRole(current); validationErr != nil {
			return validationErr
		}
		updated := *current
		updated.Name = name
		updated.Permissions = permissions
		if !dominatesRoles(effective, current, &updated) {
			return domain.ErrPermissionDenied
		}
		beforeName = current.Name
		beforePermissions = slices.Clone(current.Permissions)
		if updateErr := s.repo.UpdateRole(transactionContext, &updated); updateErr != nil {
			return updateErr
		}
		role = &updated
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("update access role: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "access-roles",
		TargetType: "access_role",
		TargetID:   strconv.FormatUint(uint64(role.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata:   map[string]any{"organization_id": organization.OrganizationID},
		Changes: map[string]domain.AuditValueChange{
			"name":        {Before: beforeName, After: role.Name},
			"permissions": {Before: beforePermissions, After: role.Permissions},
		},
	})
	return role, nil
}

func (s *service) DeleteRole(ctx context.Context, organization domain.OrganizationContext, roleID uint) error {
	if roleID == 0 {
		return domain.ErrInvalidInput
	}
	var deleted *domain.AccessRole
	err := s.withAuthorizedTransaction(ctx, organization, PermissionRolesManage, func(transactionContext context.Context, effective *domain.PermissionContext) error {
		role, findErr := s.repo.FindRole(transactionContext, organization.OrganizationID, roleID)
		if findErr != nil {
			return findErr
		}
		if err := s.validateStoredRole(role); err != nil {
			return err
		}
		if !dominatesRoles(effective, role) {
			return domain.ErrPermissionDenied
		}
		if deleteErr := s.repo.DeleteRole(transactionContext, organization.OrganizationID, roleID); deleteErr != nil {
			return deleteErr
		}
		deleted = role
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete access role: %w", err)
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "delete",
		Resource:   "access-roles",
		TargetType: "access_role",
		TargetID:   strconv.FormatUint(uint64(roleID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"organization_id": organization.OrganizationID,
			"permissions":     deleted.Permissions,
		},
	})
	return nil
}

func (s *service) MemberRoles(ctx context.Context, organization domain.OrganizationContext, memberID uint) (*MemberAccessRolesResponse, error) {
	if memberID == 0 {
		return nil, domain.ErrInvalidInput
	}
	if err := s.Authorize(ctx, organization, PermissionAssignmentsRead); err != nil {
		return nil, err
	}
	roleIDs, err := s.repo.MemberRoleIDs(ctx, organization.OrganizationID, memberID)
	if err != nil {
		return nil, err
	}
	return &MemberAccessRolesResponse{MemberID: memberID, AccessRoleIDs: roleIDs}, nil
}

func (s *service) ReplaceMemberRoles(ctx context.Context, organization domain.OrganizationContext, memberID uint, request *ReplaceMemberAccessRolesRequest) (*MemberAccessRolesResponse, error) {
	if memberID == 0 || request == nil || len(request.validationErrors()) > 0 {
		return nil, domain.ErrInvalidInput
	}
	requestedIDs := append([]uint{}, request.AccessRoleIDs...)
	slices.Sort(requestedIDs)
	var before []uint
	err := s.withAuthorizedTransaction(ctx, organization, PermissionAssignmentsManage, func(transactionContext context.Context, effective *domain.PermissionContext) error {
		currentIDs, currentErr := s.repo.MemberRoleIDs(transactionContext, organization.OrganizationID, memberID)
		if currentErr != nil {
			return currentErr
		}
		currentRoles, findErr := s.repo.FindRoles(transactionContext, organization.OrganizationID, currentIDs)
		if findErr != nil {
			return findErr
		}
		requestedRoles, findErr := s.repo.FindRoles(transactionContext, organization.OrganizationID, requestedIDs)
		if findErr != nil {
			return findErr
		}
		if err := s.validateStoredRoles(append(currentRoles, requestedRoles...)); err != nil {
			return err
		}
		if !dominatesRoles(effective, append(currentRoles, requestedRoles...)...) {
			return domain.ErrPermissionDenied
		}
		before = slices.Clone(currentIDs)
		return s.repo.ReplaceMemberRoleIDs(transactionContext, organization.OrganizationID, memberID, requestedIDs)
	})
	if err != nil {
		return nil, fmt.Errorf("replace member access roles: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "member-access-roles",
		TargetType: "organization_membership",
		TargetID:   strconv.FormatUint(uint64(memberID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata:   map[string]any{"organization_id": organization.OrganizationID},
		Changes: map[string]domain.AuditValueChange{
			"access_role_ids": {Before: before, After: requestedIDs},
		},
	})
	return &MemberAccessRolesResponse{MemberID: memberID, AccessRoleIDs: requestedIDs}, nil
}

func (s *service) withAuthorizedTransaction(
	ctx context.Context,
	organization domain.OrganizationContext,
	required domain.PermissionKey,
	operation func(context.Context, *domain.PermissionContext) error,
) error {
	if s == nil || s.repo == nil || operation == nil {
		return domain.ErrServiceUnavailable
	}
	return s.repo.WithinTransaction(ctx, func(transactionContext context.Context) error {
		effective, err := s.Effective(transactionContext, organization)
		if err != nil {
			return err
		}
		if !effective.IsOwner && !effective.Has(required) {
			return domain.ErrPermissionDenied
		}
		return operation(transactionContext, effective)
	})
}

func (s *service) roleFromCreateRequest(organizationID uint, request *CreateAccessRoleRequest) (*domain.AccessRole, error) {
	if organizationID == 0 || request == nil || len(request.validationErrors()) > 0 {
		return nil, domain.ErrInvalidInput
	}
	permissions, err := s.normalizePermissions(request.Permissions)
	if err != nil {
		return nil, err
	}
	return &domain.AccessRole{
		OrganizationID: organizationID,
		Name:           strings.TrimSpace(request.Name),
		Slug:           request.Slug,
		Permissions:    permissions,
	}, nil
}

func (s *service) normalizePermissions(values []domain.PermissionKey) ([]domain.PermissionKey, error) {
	if s == nil || s.catalog == nil || len(values) > maxRoleValues || hasDuplicateOrInvalidPermissions(values) {
		return nil, domain.ErrInvalidInput
	}
	permissions := append([]domain.PermissionKey{}, values...)
	slices.Sort(permissions)
	for _, permission := range permissions {
		if !s.catalog.Contains(permission) {
			return nil, domain.ErrPermissionUnknown
		}
	}
	return permissions, nil
}

func (s *service) validateStoredRoles(roles []*domain.AccessRole) error {
	for _, role := range roles {
		if err := s.validateStoredRole(role); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) validateStoredRole(role *domain.AccessRole) error {
	if role == nil {
		return domain.ErrServiceUnavailable
	}
	for _, permission := range role.Permissions {
		if !s.catalog.Contains(permission) {
			return fmt.Errorf("access role %d contains unregistered permission %q: %w", role.ID, permission, domain.ErrServiceUnavailable)
		}
	}
	return nil
}

func dominatesRoles(effective *domain.PermissionContext, roles ...*domain.AccessRole) bool {
	if effective == nil {
		return false
	}
	if effective.IsOwner {
		return true
	}
	for _, role := range roles {
		if role == nil {
			return false
		}
		for _, permission := range role.Permissions {
			if !effective.Has(permission) {
				return false
			}
		}
	}
	return true
}

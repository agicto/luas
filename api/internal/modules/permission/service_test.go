package permission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestServiceEnforcesExactOrganizationScopedPermissions(t *testing.T) {
	db := newPermissionTestDB(t)
	catalog := newPermissionTestCatalog(t)
	service := NewService(NewRepository(db), catalog)
	owner, manager, target := createPermissionTestOrganization(t, db, "primary")

	managerRole, err := service.CreateRole(context.Background(), owner, &CreateAccessRoleRequest{
		Name: "Access manager",
		Slug: "access-manager",
		Permissions: []domain.PermissionKey{
			PermissionRolesRead,
			PermissionRolesManage,
			PermissionAssignmentsRead,
			PermissionAssignmentsManage,
			"projects.read",
		},
	})
	require.NoError(t, err)
	_, err = service.ReplaceMemberRoles(context.Background(), owner, manager.MembershipID, &ReplaceMemberAccessRolesRequest{
		AccessRoleIDs: []uint{managerRole.ID},
	})
	require.NoError(t, err)

	allowed, err := service.CreateRole(context.Background(), manager, &CreateAccessRoleRequest{
		Name:        "Project viewer",
		Slug:        "project-viewer",
		Permissions: []domain.PermissionKey{"projects.read"},
	})
	require.NoError(t, err)
	assert.Equal(t, []domain.PermissionKey{"projects.read"}, allowed.Permissions)

	_, err = service.CreateRole(context.Background(), manager, &CreateAccessRoleRequest{
		Name:        "Project remover",
		Slug:        "project-remover",
		Permissions: []domain.PermissionKey{"projects.delete"},
	})
	require.ErrorIs(t, err, domain.ErrPermissionDenied)

	require.NoError(t, service.Authorize(context.Background(), manager, "projects.read"))
	require.ErrorIs(t, service.Authorize(context.Background(), manager, "projects.delete"), domain.ErrPermissionDenied)
	require.ErrorIs(t, service.Authorize(context.Background(), manager, "projects"), domain.ErrServiceUnavailable)
	require.NoError(t, service.Authorize(context.Background(), owner, "projects.delete"))

	highRole, err := service.CreateRole(context.Background(), owner, &CreateAccessRoleRequest{
		Name:        "Project remover",
		Slug:        "project-remover-owner",
		Permissions: []domain.PermissionKey{"projects.delete"},
	})
	require.NoError(t, err)
	_, err = service.ReplaceMemberRoles(context.Background(), owner, target.MembershipID, &ReplaceMemberAccessRolesRequest{
		AccessRoleIDs: []uint{highRole.ID},
	})
	require.NoError(t, err)

	_, err = service.ReplaceMemberRoles(context.Background(), manager, target.MembershipID, &ReplaceMemberAccessRolesRequest{})
	require.ErrorIs(t, err, domain.ErrPermissionDenied)
}

func TestServiceRejectsCrossOrganizationRoleAssignments(t *testing.T) {
	db := newPermissionTestDB(t)
	service := NewService(NewRepository(db), newPermissionTestCatalog(t))
	firstOwner, _, firstTarget := createPermissionTestOrganization(t, db, "first")
	secondOwner, _, _ := createPermissionTestOrganization(t, db, "second")

	foreignRole, err := service.CreateRole(context.Background(), secondOwner, &CreateAccessRoleRequest{
		Name:        "Foreign viewer",
		Slug:        "foreign-viewer",
		Permissions: []domain.PermissionKey{"projects.read"},
	})
	require.NoError(t, err)

	_, err = service.ReplaceMemberRoles(context.Background(), firstOwner, firstTarget.MembershipID, &ReplaceMemberAccessRolesRequest{
		AccessRoleIDs: []uint{foreignRole.ID},
	})
	require.ErrorIs(t, err, domain.ErrAccessRoleNotFound)
}

func TestServiceRoleSlugIsUniquePerOrganization(t *testing.T) {
	db := newPermissionTestDB(t)
	service := NewService(NewRepository(db), newPermissionTestCatalog(t))
	owner, _, _ := createPermissionTestOrganization(t, db, "slug")

	request := &CreateAccessRoleRequest{
		Name:        "Project viewer",
		Slug:        "project-viewer",
		Permissions: []domain.PermissionKey{"projects.read"},
	}
	_, err := service.CreateRole(context.Background(), owner, request)
	require.NoError(t, err)
	_, err = service.CreateRole(context.Background(), owner, request)
	require.ErrorIs(t, err, domain.ErrAccessRoleSlugAlreadyExists)
}

func TestServiceRejectsUnregisteredPermissions(t *testing.T) {
	db := newPermissionTestDB(t)
	service := NewService(NewRepository(db), newPermissionTestCatalog(t))
	owner, _, _ := createPermissionTestOrganization(t, db, "unknown")

	_, err := service.CreateRole(context.Background(), owner, &CreateAccessRoleRequest{
		Name:        "Unknown role",
		Slug:        "unknown-role",
		Permissions: []domain.PermissionKey{"invoices.approve"},
	})
	require.ErrorIs(t, err, domain.ErrPermissionUnknown)
}

func newPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
	require.NoError(t, db.AutoMigrate(
		&user.UserPO{},
		&organization.OrganizationPO{},
		&organization.OrganizationMembershipPO{},
		&AccessRolePO{},
		&AccessRolePermissionPO{},
		&AccessRoleAssignmentPO{},
	))
	return db
}

func newPermissionTestCatalog(t *testing.T) *Catalog {
	t.Helper()
	keys := append(DefaultPermissionKeys(), "projects.read", "projects.delete")
	catalog, err := NewCatalog(keys...)
	require.NoError(t, err)
	return catalog
}

func createPermissionTestOrganization(
	t *testing.T,
	db *gorm.DB,
	slug string,
) (domain.OrganizationContext, domain.OrganizationContext, domain.OrganizationContext) {
	t.Helper()
	ownerID := createPermissionTestUser(t, db, slug+"-owner")
	managerID := createPermissionTestUser(t, db, slug+"-manager")
	targetID := createPermissionTestUser(t, db, slug+"-target")
	organizationPO := &organization.OrganizationPO{
		Name:      slug + " organization",
		Slug:      slug + "-organization",
		CreatedBy: ownerID,
	}
	require.NoError(t, db.Omit(clause.Associations).Create(organizationPO).Error)

	owner := createPermissionTestMembership(t, db, organizationPO, ownerID, domain.OrganizationRoleOwner)
	manager := createPermissionTestMembership(t, db, organizationPO, managerID, domain.OrganizationRoleAdmin)
	target := createPermissionTestMembership(t, db, organizationPO, targetID, domain.OrganizationRoleMember)
	return owner, manager, target
}

func createPermissionTestUser(t *testing.T, db *gorm.DB, username string) uint {
	t.Helper()
	po := &user.UserPO{
		Username: username,
		Email:    username + "@example.com",
		Password: "not-used",
		Status:   1,
	}
	require.NoError(t, db.Omit(clause.Associations).Create(po).Error)
	return po.ID
}

func createPermissionTestMembership(
	t *testing.T,
	db *gorm.DB,
	organizationPO *organization.OrganizationPO,
	userID uint,
	role domain.OrganizationRole,
) domain.OrganizationContext {
	t.Helper()
	membership := &organization.OrganizationMembershipPO{
		OrganizationID: organizationPO.ID,
		UserID:         userID,
		Role:           string(role),
	}
	require.NoError(t, db.Omit(clause.Associations).Create(membership).Error)
	return domain.OrganizationContext{
		OrganizationID:   organizationPO.ID,
		OrganizationName: organizationPO.Name,
		OrganizationSlug: organizationPO.Slug,
		MembershipID:     membership.ID,
		UserID:           userID,
		Role:             role,
	}
}

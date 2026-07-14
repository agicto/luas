package organization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func newOrganizationRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.NewTestDB()
	require.NoError(t, err)
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
	require.NoError(t, db.AutoMigrate(
		&user.UserPO{},
		&OrganizationPO{},
		&OrganizationMembershipPO{},
	))
	return db
}

func createOrganizationTestUser(t *testing.T, db *gorm.DB, username string) uint {
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

func TestRepositoryCreateWithOwnerAndMembershipScopedReads(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	ownerID := createOrganizationTestUser(t, db, "owner")
	otherID := createOrganizationTestUser(t, db, "other")

	organization := &domain.Organization{Name: "Acme", Slug: "acme", CreatedBy: ownerID}
	owner := &domain.OrganizationMembership{UserID: ownerID, Role: domain.OrganizationRoleOwner}
	require.NoError(t, repo.CreateWithOwner(context.Background(), organization, owner))

	assert.NotZero(t, organization.ID)
	assert.NotZero(t, owner.ID)
	assert.Equal(t, organization.ID, owner.OrganizationID)
	assert.Same(t, organization, owner.Organization)

	found, err := repo.FindForUser(context.Background(), organization.ID, ownerID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", found.Organization.Name)
	assert.Equal(t, domain.OrganizationRoleOwner, found.Role)

	_, err = repo.FindForUser(context.Background(), organization.ID, otherID)
	require.ErrorIs(t, err, domain.ErrOrganizationNotFound)

	items, total, err := repo.ListForUser(context.Background(), ownerID, 1, 15)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, organization.ID, items[0].OrganizationID)

	owned, err := repo.CountOwnedByUser(context.Background(), ownerID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), owned)
}

func TestRepositoryCreateWithOwnerRollsBackAndMapsDuplicateSlug(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	ownerID := createOrganizationTestUser(t, db, "owner")

	first := &domain.Organization{Name: "First", Slug: "shared", CreatedBy: ownerID}
	require.NoError(t, repo.CreateWithOwner(context.Background(), first, &domain.OrganizationMembership{
		UserID: ownerID,
		Role:   domain.OrganizationRoleOwner,
	}))

	duplicate := &domain.Organization{Name: "Duplicate", Slug: "shared", CreatedBy: ownerID}
	err := repo.CreateWithOwner(context.Background(), duplicate, &domain.OrganizationMembership{
		UserID: ownerID,
		Role:   domain.OrganizationRoleOwner,
	})
	require.ErrorIs(t, err, domain.ErrOrganizationSlugAlreadyExists)
	assert.Zero(t, duplicate.ID)

	var organizations int64
	require.NoError(t, db.Model(&OrganizationPO{}).Count(&organizations).Error)
	assert.Equal(t, int64(1), organizations)
}

func TestRepositoryRejectsInvalidMembershipRole(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	ownerID := createOrganizationTestUser(t, db, "owner")
	organization := &OrganizationPO{Name: "Acme", Slug: "acme", CreatedBy: ownerID}
	require.NoError(t, db.Omit(clause.Associations).Create(organization).Error)

	err := db.Omit(clause.Associations).Create(&OrganizationMembershipPO{
		OrganizationID: organization.ID,
		UserID:         ownerID,
		Role:           "superadmin",
	}).Error
	require.Error(t, err)
}

func TestRepositoryCreateWithOwnerRejectsNonOwnerMembership(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	userID := createOrganizationTestUser(t, db, "member")

	err := repo.CreateWithOwner(
		context.Background(),
		&domain.Organization{Name: "Acme", Slug: "acme", CreatedBy: userID},
		&domain.OrganizationMembership{UserID: userID, Role: domain.OrganizationRoleMember},
	)
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	var organizations int64
	require.NoError(t, db.Model(&OrganizationPO{}).Count(&organizations).Error)
	assert.Zero(t, organizations)
}

func TestRepositoryWithoutDatabaseIsUnavailable(t *testing.T) {
	repo := NewRepository(nil)
	_, err := repo.FindForUser(context.Background(), 1, 1)
	require.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

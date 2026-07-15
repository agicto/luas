package organization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
)

func createOrganizationMembershipFixture(t *testing.T) (*gorm.DB, *repository, uint, map[string]*OrganizationMembershipPO) {
	t.Helper()
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	ownerID := createOrganizationTestUser(t, db, "member-owner")
	adminID := createOrganizationTestUser(t, db, "member-admin")
	memberID := createOrganizationTestUser(t, db, "member-user")
	organization := &OrganizationPO{Name: "Member Org", Slug: "member-org", CreatedBy: ownerID}
	require.NoError(t, db.Omit(clause.Associations).Create(organization).Error)

	memberships := map[string]*OrganizationMembershipPO{
		"owner":  {OrganizationID: organization.ID, UserID: ownerID, Role: string(domain.OrganizationRoleOwner)},
		"admin":  {OrganizationID: organization.ID, UserID: adminID, Role: string(domain.OrganizationRoleAdmin)},
		"member": {OrganizationID: organization.ID, UserID: memberID, Role: string(domain.OrganizationRoleMember)},
	}
	for _, membership := range memberships {
		require.NoError(t, db.Omit(clause.Associations).Create(membership).Error)
	}
	return db, repo, organization.ID, memberships
}

func TestRepositoryListsMembersWithoutPrivateProfileFields(t *testing.T) {
	_, repo, organizationID, memberships := createOrganizationMembershipFixture(t)

	items, total, err := repo.ListMembers(
		context.Background(),
		organizationID,
		memberships["member"].UserID,
		1,
		2,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, items, 2)
	for _, membership := range items {
		require.NotNil(t, membership.User)
		assert.NotZero(t, membership.User.ID)
		assert.NotEmpty(t, membership.User.Username)
		assert.Empty(t, membership.User.Email)
		assert.Empty(t, membership.User.Password)
		assert.Empty(t, membership.User.Phone)
	}

	_, _, err = repo.ListMembers(context.Background(), organizationID, 999999, 1, 10)
	require.ErrorIs(t, err, domain.ErrOrganizationNotFound)
}

func TestRepositoryChangesMemberRoleWithOwnerOnlyPolicy(t *testing.T) {
	_, repo, organizationID, memberships := createOrganizationMembershipFixture(t)
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	change, err := repo.ChangeMemberRole(
		context.Background(),
		organizationID,
		memberships["owner"].UserID,
		memberships["member"].ID,
		domain.OrganizationRoleAdmin,
		now,
	)
	require.NoError(t, err)
	require.NotNil(t, change)
	assert.Equal(t, domain.OrganizationRoleMember, change.BeforeRole)
	assert.Equal(t, domain.OrganizationRoleAdmin, change.Membership.Role)
	assert.Equal(t, now, change.Membership.UpdatedAt)

	_, err = repo.ChangeMemberRole(
		context.Background(),
		organizationID,
		memberships["admin"].UserID,
		memberships["member"].ID,
		domain.OrganizationRoleMember,
		now,
	)
	require.ErrorIs(t, err, domain.ErrPermissionDenied)

	_, err = repo.ChangeMemberRole(
		context.Background(),
		organizationID,
		memberships["owner"].UserID,
		memberships["owner"].ID,
		domain.OrganizationRoleMember,
		now,
	)
	require.ErrorIs(t, err, domain.ErrOrganizationOwnershipTransferRequired)

	_, err = repo.ChangeMemberRole(
		context.Background(),
		organizationID,
		memberships["owner"].UserID,
		999999,
		domain.OrganizationRoleMember,
		now,
	)
	require.ErrorIs(t, err, domain.ErrOrganizationMemberNotFound)
}

func TestRepositoryRemovalPolicySupportsManagerRemovalAndSelfLeave(t *testing.T) {
	db, repo, organizationID, memberships := createOrganizationMembershipFixture(t)

	_, err := repo.RemoveMember(
		context.Background(),
		organizationID,
		memberships["admin"].UserID,
		memberships["owner"].ID,
	)
	require.ErrorIs(t, err, domain.ErrPermissionDenied)

	removed, err := repo.RemoveMember(
		context.Background(),
		organizationID,
		memberships["admin"].UserID,
		memberships["member"].ID,
	)
	require.NoError(t, err)
	assert.Equal(t, memberships["member"].UserID, removed.UserID)
	var remaining int64
	require.NoError(t, db.Model(&OrganizationMembershipPO{}).
		Where("id = ?", memberships["member"].ID).
		Count(&remaining).Error)
	assert.Zero(t, remaining)

	left, err := repo.RemoveMember(
		context.Background(),
		organizationID,
		memberships["admin"].UserID,
		memberships["admin"].ID,
	)
	require.NoError(t, err)
	assert.Equal(t, memberships["admin"].UserID, left.UserID)

	_, err = repo.RemoveMember(
		context.Background(),
		organizationID,
		memberships["owner"].UserID,
		memberships["owner"].ID,
	)
	require.ErrorIs(t, err, domain.ErrOrganizationOwnershipTransferRequired)
}

func TestRepositoryTransfersOwnershipAtomically(t *testing.T) {
	db, repo, organizationID, memberships := createOrganizationMembershipFixture(t)
	now := time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)

	transfer, err := repo.TransferOwnership(
		context.Background(),
		organizationID,
		memberships["owner"].UserID,
		memberships["admin"].ID,
		now,
	)
	require.NoError(t, err)
	require.NotNil(t, transfer)
	assert.Equal(t, domain.OrganizationRoleAdmin, transfer.PreviousOwner.Role)
	assert.Equal(t, domain.OrganizationRoleOwner, transfer.NewOwner.Role)
	assert.Equal(t, now, transfer.PreviousOwner.UpdatedAt)
	assert.Equal(t, now, transfer.NewOwner.UpdatedAt)

	var owners int64
	require.NoError(t, db.Model(&OrganizationMembershipPO{}).
		Where("organization_id = ? AND role = ?", organizationID, domain.OrganizationRoleOwner).
		Count(&owners).Error)
	assert.Equal(t, int64(1), owners)

	_, err = repo.TransferOwnership(
		context.Background(),
		organizationID,
		memberships["owner"].UserID,
		memberships["member"].ID,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, domain.ErrPermissionDenied)

	_, err = repo.TransferOwnership(
		context.Background(),
		organizationID,
		memberships["admin"].UserID,
		memberships["admin"].ID,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, domain.ErrOrganizationOwnershipTransferTargetInvalid)
}

func TestRepositoryCountsOwnedAndNonOwnerMemberships(t *testing.T) {
	_, repo, _, memberships := createOrganizationMembershipFixture(t)

	total, owned, err := repo.CountMembershipsForUser(context.Background(), memberships["owner"].UserID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, int64(1), owned)

	total, owned, err = repo.CountMembershipsForUser(context.Background(), memberships["member"].UserID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Zero(t, owned)
}

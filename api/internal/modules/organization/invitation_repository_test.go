package organization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func createInvitationTestOrganization(t *testing.T, db *gorm.DB, ownerID uint) *domain.Organization {
	t.Helper()
	repo := NewRepository(db)
	organization := &domain.Organization{Name: "Acme", Slug: "acme-invitations", CreatedBy: ownerID}
	owner := &domain.OrganizationMembership{UserID: ownerID, Role: domain.OrganizationRoleOwner}
	require.NoError(t, repo.CreateWithOwner(context.Background(), organization, owner))
	return organization
}

func newPendingInvitation(organizationID, invitedBy uint, email, token string, now time.Time) *domain.OrganizationInvitation {
	return &domain.OrganizationInvitation{
		OrganizationID: organizationID,
		InvitedBy:      invitedBy,
		Email:          email,
		Role:           domain.OrganizationRoleMember,
		TokenHash:      crypto.SHA256Hex(token),
		ExpiresAt:      now.Add(7 * 24 * time.Hour),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestRepositoryInvitationLifecycleIsTransactional(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	ownerID := createOrganizationTestUser(t, db, "invitation-owner")
	inviteeID := createOrganizationTestUser(t, db, "invitation-member")
	require.NoError(t, db.Table("users").Where("id = ?", inviteeID).Update("email", "Member@Example.com").Error)
	organization := createInvitationTestOrganization(t, db, ownerID)
	invitation := newPendingInvitation(organization.ID, ownerID, "member@example.com", "oinv_one.secret", now)

	require.NoError(t, repo.CreateInvitation(context.Background(), invitation, now))
	assert.NotZero(t, invitation.ID)
	assert.Len(t, invitation.TokenHash, 64)

	duplicate := newPendingInvitation(organization.ID, ownerID, "member@example.com", "oinv_two.secret", now)
	err := repo.CreateInvitation(context.Background(), duplicate, now)
	require.ErrorIs(t, err, domain.ErrOrganizationInvitationAlreadyPending)
	assert.Zero(t, duplicate.ID)

	membership, accepted, err := repo.AcceptInvitation(
		context.Background(),
		crypto.SHA256Hex("oinv_one.secret"),
		inviteeID,
		now.Add(time.Hour),
	)
	require.NoError(t, err)
	require.NotNil(t, accepted.AcceptedAt)
	assert.Equal(t, inviteeID, *accepted.AcceptedBy)
	assert.Equal(t, domain.OrganizationRoleMember, membership.Role)
	assert.Equal(t, organization.ID, membership.OrganizationID)
	require.NotNil(t, membership.Organization)

	_, _, err = repo.AcceptInvitation(
		context.Background(),
		crypto.SHA256Hex("oinv_one.secret"),
		inviteeID,
		now.Add(2*time.Hour),
	)
	require.ErrorIs(t, err, domain.ErrOrganizationInvitationInvalid)

	var membershipCount int64
	require.NoError(t, db.Model(&OrganizationMembershipPO{}).
		Where("organization_id = ? AND user_id = ?", organization.ID, inviteeID).
		Count(&membershipCount).Error)
	assert.Equal(t, int64(1), membershipCount)
}

func TestRepositoryInvitationRejectsWrongAccountAndExpiry(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	ownerID := createOrganizationTestUser(t, db, "edge-owner")
	inviteeID := createOrganizationTestUser(t, db, "edge-member")
	otherID := createOrganizationTestUser(t, db, "edge-other")
	organization := createInvitationTestOrganization(t, db, ownerID)

	wrongAccount := newPendingInvitation(organization.ID, ownerID, "edge-member@example.com", "oinv_wrong.secret", now)
	require.NoError(t, repo.CreateInvitation(context.Background(), wrongAccount, now))
	_, _, err := repo.AcceptInvitation(context.Background(), wrongAccount.TokenHash, otherID, now.Add(time.Hour))
	require.ErrorIs(t, err, domain.ErrOrganizationInvitationEmailMismatch)

	expired := newPendingInvitation(organization.ID, ownerID, "edge-other@example.com", "oinv_expired.secret", now)
	expired.ExpiresAt = now.Add(time.Hour)
	require.NoError(t, repo.CreateInvitation(context.Background(), expired, now))
	_, _, err = repo.AcceptInvitation(context.Background(), expired.TokenHash, otherID, expired.ExpiresAt)
	require.ErrorIs(t, err, domain.ErrOrganizationInvitationExpired)

	var memberships int64
	require.NoError(t, db.Model(&OrganizationMembershipPO{}).
		Where("organization_id = ? AND user_id IN ?", organization.ID, []uint{inviteeID, otherID}).
		Count(&memberships).Error)
	assert.Zero(t, memberships)
}

func TestRepositoryInvitationRevokeListAndExpiredReplacement(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	ownerID := createOrganizationTestUser(t, db, "manage-owner")
	organization := createInvitationTestOrganization(t, db, ownerID)

	toRevoke := newPendingInvitation(organization.ID, ownerID, "revoke@example.com", "oinv_revoke.secret", now)
	require.NoError(t, repo.CreateInvitation(context.Background(), toRevoke, now))
	revoked, err := repo.RevokeInvitation(context.Background(), organization.ID, toRevoke.ID, ownerID, now.Add(time.Hour))
	require.NoError(t, err)
	require.NotNil(t, revoked.RevokedAt)
	assert.Equal(t, ownerID, *revoked.RevokedBy)

	_, _, err = repo.AcceptInvitation(context.Background(), toRevoke.TokenHash, ownerID, now.Add(2*time.Hour))
	require.ErrorIs(t, err, domain.ErrOrganizationInvitationInvalid)

	expired := newPendingInvitation(organization.ID, ownerID, "replace@example.com", "oinv_old.secret", now.Add(-8*24*time.Hour))
	expired.ExpiresAt = now.Add(-time.Hour)
	require.NoError(t, repo.CreateInvitation(context.Background(), expired, now.Add(-8*24*time.Hour)))
	expiredUpdatedAt := expired.UpdatedAt
	replacement := newPendingInvitation(organization.ID, ownerID, "replace@example.com", "oinv_new.secret", now)
	require.NoError(t, repo.CreateInvitation(context.Background(), replacement, now))

	items, total, err := repo.ListInvitations(context.Background(), organization.ID, 1, 15)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, items, 3)

	var expiredPO OrganizationInvitationPO
	require.NoError(t, db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&expiredPO, expired.ID).Error)
	assert.Equal(t, domain.OrganizationInvitationStatusExpired, expiredPO.toDomain().Status(now))
	assert.Nil(t, expiredPO.RevokedAt)
	assert.Nil(t, expiredPO.PendingKey)
	assert.Equal(t, expiredUpdatedAt, expiredPO.UpdatedAt)

	_, _, err = repo.AcceptInvitation(context.Background(), expired.TokenHash, ownerID, now)
	require.ErrorIs(t, err, domain.ErrOrganizationInvitationExpired)
}

func TestRepositoryInvitationRejectsExistingMember(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	ownerID := createOrganizationTestUser(t, db, "member-owner")
	memberID := createOrganizationTestUser(t, db, "existing-member")
	organization := createInvitationTestOrganization(t, db, ownerID)
	require.NoError(t, db.Omit(clause.Associations).Create(&OrganizationMembershipPO{
		OrganizationID: organization.ID,
		UserID:         memberID,
		Role:           string(domain.OrganizationRoleMember),
	}).Error)

	invitation := newPendingInvitation(organization.ID, ownerID, "existing-member@example.com", "oinv_member.secret", now)
	err := repo.CreateInvitation(context.Background(), invitation, now)
	require.ErrorIs(t, err, domain.ErrOrganizationMemberAlreadyExists)
	assert.Zero(t, invitation.ID)

	require.NoError(t, db.Delete(&user.UserPO{}, memberID).Error)
	softDeletedMember := newPendingInvitation(
		organization.ID,
		ownerID,
		"existing-member@example.com",
		"oinv_soft_deleted_member.secret",
		now,
	)
	err = repo.CreateInvitation(context.Background(), softDeletedMember, now)
	require.ErrorIs(t, err, domain.ErrOrganizationMemberAlreadyExists)
	assert.Zero(t, softDeletedMember.ID)
}

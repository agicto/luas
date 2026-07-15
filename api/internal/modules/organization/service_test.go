package organization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeRepository struct {
	createFn func(context.Context, *domain.Organization, *domain.OrganizationMembership) error
	findFn   func(context.Context, uint, uint) (*domain.OrganizationMembership, error)
	listFn   func(context.Context, uint, int, int) ([]*domain.OrganizationMembership, int64, error)
	updateFn func(context.Context, *domain.Organization) error
	countFn  func(context.Context, uint) (int64, int64, error)
}

func newOrganizationService(repo domain.OrganizationRepository) *service {
	return NewService(repo, nil, nil, InvitationPolicy{TTL: 7 * 24 * time.Hour})
}

func (r *fakeRepository) CreateWithOwner(ctx context.Context, organization *domain.Organization, owner *domain.OrganizationMembership) error {
	return r.createFn(ctx, organization, owner)
}

func (r *fakeRepository) FindForUser(ctx context.Context, organizationID, userID uint) (*domain.OrganizationMembership, error) {
	return r.findFn(ctx, organizationID, userID)
}

func (r *fakeRepository) ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.OrganizationMembership, int64, error) {
	return r.listFn(ctx, userID, page, pageSize)
}

func (r *fakeRepository) ListMembers(context.Context, uint, uint, int, int) ([]*domain.OrganizationMembership, int64, error) {
	return nil, 0, nil
}

func (r *fakeRepository) ChangeMemberRole(context.Context, uint, uint, uint, domain.OrganizationRole, time.Time) (*domain.OrganizationMembershipRoleChange, error) {
	return nil, nil
}

func (r *fakeRepository) RemoveMember(context.Context, uint, uint, uint) (*domain.OrganizationMembership, error) {
	return nil, nil
}

func (r *fakeRepository) TransferOwnership(context.Context, uint, uint, uint, time.Time) (*domain.OrganizationOwnershipTransfer, error) {
	return nil, nil
}

func (r *fakeRepository) Update(ctx context.Context, organization *domain.Organization) error {
	return r.updateFn(ctx, organization)
}

func (r *fakeRepository) CountMembershipsForUser(ctx context.Context, userID uint) (int64, int64, error) {
	return r.countFn(ctx, userID)
}

func TestServiceCreateBuildsOwnerMembership(t *testing.T) {
	now := time.Date(2026, time.July, 14, 20, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		createFn: func(_ context.Context, organization *domain.Organization, owner *domain.OrganizationMembership) error {
			assert.Equal(t, "Acme Europe", organization.Name)
			assert.Equal(t, "acme-europe", organization.Slug)
			assert.Equal(t, uint(17), organization.CreatedBy)
			assert.Equal(t, uint(17), owner.UserID)
			assert.Equal(t, domain.OrganizationRoleOwner, owner.Role)

			organization.ID = 42
			organization.CreatedAt = now
			organization.UpdatedAt = now
			owner.ID = 91
			owner.OrganizationID = organization.ID
			owner.Organization = organization
			return nil
		},
	}

	membership, err := newOrganizationService(repo).Create(context.Background(), 17, &CreateOrganizationRequest{
		Name: "  Acme Europe  ",
		Slug: "acme-europe",
	})
	require.NoError(t, err)
	require.NotNil(t, membership)
	assert.Equal(t, uint(42), membership.OrganizationID)
	require.NotNil(t, membership.Organization)
	assert.Equal(t, "Acme Europe", membership.Organization.Name)
}

func TestServiceCreateRejectsInvalidSlugBeforePersistence(t *testing.T) {
	repo := &fakeRepository{
		createFn: func(context.Context, *domain.Organization, *domain.OrganizationMembership) error {
			t.Fatal("repository must not run for an invalid slug")
			return nil
		},
	}

	_, err := newOrganizationService(repo).Create(context.Background(), 17, &CreateOrganizationRequest{
		Name: "Acme",
		Slug: "Not Valid",
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestServiceUpdateRequiresOrganizationManagerRole(t *testing.T) {
	organization := &domain.Organization{ID: 42, Name: "Before", Slug: "before"}
	repo := &fakeRepository{
		findFn: func(context.Context, uint, uint) (*domain.OrganizationMembership, error) {
			return &domain.OrganizationMembership{
				OrganizationID: 42,
				UserID:         17,
				Role:           domain.OrganizationRoleMember,
				Organization:   organization,
			}, nil
		},
		updateFn: func(context.Context, *domain.Organization) error {
			t.Fatal("member must not update organization settings")
			return nil
		},
	}

	_, err := newOrganizationService(repo).Update(context.Background(), 17, 42, &UpdateOrganizationRequest{Name: "After"})
	require.ErrorIs(t, err, domain.ErrPermissionDenied)
}

func TestServiceUpdateAllowsOwnerAndReturnsMembershipView(t *testing.T) {
	organization := &domain.Organization{ID: 42, Name: "Before", Slug: "before"}
	membership := &domain.OrganizationMembership{
		OrganizationID: 42,
		UserID:         17,
		Role:           domain.OrganizationRoleOwner,
		Organization:   organization,
	}
	repo := &fakeRepository{
		findFn: func(context.Context, uint, uint) (*domain.OrganizationMembership, error) {
			return membership, nil
		},
		updateFn: func(_ context.Context, updated *domain.Organization) error {
			assert.Equal(t, "After", updated.Name)
			return nil
		},
	}

	updated, err := newOrganizationService(repo).Update(context.Background(), 17, 42, &UpdateOrganizationRequest{Name: " After "})
	require.NoError(t, err)
	assert.Same(t, membership, updated)
	assert.Equal(t, "After", updated.Organization.Name)
}

func TestServiceAccountDeletionGuardProtectsOwnedOrganizations(t *testing.T) {
	repo := &fakeRepository{
		countFn: func(_ context.Context, userID uint) (int64, int64, error) {
			assert.Equal(t, uint(17), userID)
			return 2, 2, nil
		},
	}
	svc := newOrganizationService(repo)

	assert.Equal(t, "organization", svc.AccountDeletionGuardName())
	err := svc.CheckAccountDeletion(context.Background(), 17)
	require.ErrorIs(t, err, domain.ErrOrganizationOwnershipTransferRequired)
}

func TestServiceAccountDeletionGuardRequiresMembershipExit(t *testing.T) {
	repo := &fakeRepository{
		countFn: func(_ context.Context, userID uint) (int64, int64, error) {
			assert.Equal(t, uint(17), userID)
			return 2, 0, nil
		},
	}

	err := newOrganizationService(repo).CheckAccountDeletion(context.Background(), 17)
	require.ErrorIs(t, err, domain.ErrOrganizationMembershipExitRequired)
}

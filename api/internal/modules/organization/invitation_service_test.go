package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
)

type invitationRepositoryStub struct {
	findFn             func(context.Context, uint, uint) (*domain.OrganizationMembership, error)
	createInvitationFn func(context.Context, *domain.OrganizationInvitation, time.Time) error
	listInvitationsFn  func(context.Context, uint, int, int) ([]*domain.OrganizationInvitation, int64, error)
	revokeInvitationFn func(context.Context, uint, uint, uint, time.Time) (*domain.OrganizationInvitation, error)
	acceptInvitationFn func(context.Context, string, uint, time.Time) (*domain.OrganizationMembership, *domain.OrganizationInvitation, error)
}

func (r *invitationRepositoryStub) CreateWithOwner(context.Context, *domain.Organization, *domain.OrganizationMembership) error {
	return nil
}

func (r *invitationRepositoryStub) FindForUser(ctx context.Context, organizationID, userID uint) (*domain.OrganizationMembership, error) {
	return r.findFn(ctx, organizationID, userID)
}

func (r *invitationRepositoryStub) ListForUser(context.Context, uint, int, int) ([]*domain.OrganizationMembership, int64, error) {
	return nil, 0, nil
}

func (r *invitationRepositoryStub) Update(context.Context, *domain.Organization) error {
	return nil
}

func (r *invitationRepositoryStub) CountOwnedByUser(context.Context, uint) (int64, error) {
	return 0, nil
}

func (r *invitationRepositoryStub) CreateInvitation(ctx context.Context, invitation *domain.OrganizationInvitation, now time.Time) error {
	return r.createInvitationFn(ctx, invitation, now)
}

func (r *invitationRepositoryStub) ListInvitations(ctx context.Context, organizationID uint, page, pageSize int) ([]*domain.OrganizationInvitation, int64, error) {
	return r.listInvitationsFn(ctx, organizationID, page, pageSize)
}

func (r *invitationRepositoryStub) RevokeInvitation(ctx context.Context, organizationID, invitationID, revokedBy uint, now time.Time) (*domain.OrganizationInvitation, error) {
	return r.revokeInvitationFn(ctx, organizationID, invitationID, revokedBy, now)
}

func (r *invitationRepositoryStub) AcceptInvitation(ctx context.Context, tokenHash string, userID uint, now time.Time) (*domain.OrganizationMembership, *domain.OrganizationInvitation, error) {
	return r.acceptInvitationFn(ctx, tokenHash, userID, now)
}

type invitationMailerStub struct {
	configured bool
	sendFn     func(context.Context, string, string, domain.OrganizationRole, string, time.Time) error
}

func (m *invitationMailerStub) IsConfigured() bool {
	return m != nil && m.configured
}

func (m *invitationMailerStub) SendInvitation(ctx context.Context, to, organizationName string, role domain.OrganizationRole, token string, expiresAt time.Time) error {
	return m.sendFn(ctx, to, organizationName, role, token, expiresAt)
}

func invitationManager(role domain.OrganizationRole) *domain.OrganizationMembership {
	return &domain.OrganizationMembership{
		OrganizationID: 42,
		UserID:         17,
		Role:           role,
		Organization: &domain.Organization{
			ID:   42,
			Name: "Acme Europe",
			Slug: "acme-europe",
		},
	}
}

func TestServiceInvitePersistsHashedTokenBeforeEmail(t *testing.T) {
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	persisted := false
	var persistedHash string
	repo := &invitationRepositoryStub{
		findFn: func(context.Context, uint, uint) (*domain.OrganizationMembership, error) {
			return invitationManager(domain.OrganizationRoleOwner), nil
		},
		createInvitationFn: func(_ context.Context, invitation *domain.OrganizationInvitation, createdAt time.Time) error {
			persisted = true
			persistedHash = invitation.TokenHash
			assert.Equal(t, now, createdAt)
			assert.Equal(t, "member@example.com", invitation.Email)
			assert.Equal(t, domain.OrganizationRoleMember, invitation.Role)
			assert.Equal(t, now.Add(7*24*time.Hour), invitation.ExpiresAt)
			assert.Len(t, invitation.TokenHash, 64)
			invitation.ID = 73
			return nil
		},
	}
	mailer := &invitationMailerStub{
		configured: true,
		sendFn: func(_ context.Context, to, organizationName string, role domain.OrganizationRole, token string, expiresAt time.Time) error {
			assert.True(t, persisted, "invitation must commit before email")
			assert.Equal(t, "member@example.com", to)
			assert.Equal(t, "Acme Europe", organizationName)
			assert.Equal(t, domain.OrganizationRoleMember, role)
			assert.Equal(t, now.Add(7*24*time.Hour), expiresAt)
			assert.Equal(t, persistedHash, crypto.SHA256Hex(token))
			return nil
		},
	}
	svc := NewService(repo, repo, mailer, InvitationPolicy{TTL: 7 * 24 * time.Hour})
	svc.now = func() time.Time { return now }

	result, err := svc.Invite(context.Background(), 17, 42, &CreateOrganizationInvitationRequest{
		Email: "  MEMBER@Example.com ",
		Role:  domain.OrganizationRoleMember,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, InvitationEmailSendStatusAcceptedByProvider, result.EmailSendStatus)
	assert.Equal(t, uint(73), result.Invitation.ID)
	assert.NotContains(t, result.Invitation.TokenHash, "oinv_")
}

func TestServiceInviteKeepsInvitationWhenEmailFails(t *testing.T) {
	repo := &invitationRepositoryStub{
		findFn: func(context.Context, uint, uint) (*domain.OrganizationMembership, error) {
			return invitationManager(domain.OrganizationRoleAdmin), nil
		},
		createInvitationFn: func(_ context.Context, invitation *domain.OrganizationInvitation, _ time.Time) error {
			invitation.ID = 73
			return nil
		},
	}
	mailer := &invitationMailerStub{
		configured: true,
		sendFn: func(context.Context, string, string, domain.OrganizationRole, string, time.Time) error {
			return errors.New("provider unavailable")
		},
	}
	svc := NewService(repo, repo, mailer, InvitationPolicy{TTL: 7 * 24 * time.Hour})

	result, err := svc.Invite(context.Background(), 17, 42, &CreateOrganizationInvitationRequest{
		Email: "member@example.com",
		Role:  domain.OrganizationRoleMember,
	})
	require.NoError(t, err)
	assert.Equal(t, InvitationEmailSendStatusFailed, result.EmailSendStatus)
	assert.Equal(t, uint(73), result.Invitation.ID)
}

func TestServiceInviteReportsUnconfiguredEmailAfterPersistence(t *testing.T) {
	persisted := false
	repo := &invitationRepositoryStub{
		findFn: func(context.Context, uint, uint) (*domain.OrganizationMembership, error) {
			return invitationManager(domain.OrganizationRoleOwner), nil
		},
		createInvitationFn: func(_ context.Context, invitation *domain.OrganizationInvitation, _ time.Time) error {
			persisted = true
			invitation.ID = 73
			return nil
		},
	}
	svc := NewService(repo, repo, &invitationMailerStub{}, InvitationPolicy{TTL: 7 * 24 * time.Hour})

	result, err := svc.Invite(context.Background(), 17, 42, &CreateOrganizationInvitationRequest{
		Email: "member@example.com",
		Role:  domain.OrganizationRoleMember,
	})
	require.NoError(t, err)
	assert.True(t, persisted)
	assert.Equal(t, InvitationEmailSendStatusNotConfigured, result.EmailSendStatus)
}

func TestServiceInviteRequiresManagerAndNonOwnerRole(t *testing.T) {
	tests := []struct {
		name        string
		managerRole domain.OrganizationRole
		inviteRole  domain.OrganizationRole
		want        error
	}{
		{name: "member cannot invite", managerRole: domain.OrganizationRoleMember, inviteRole: domain.OrganizationRoleMember, want: domain.ErrPermissionDenied},
		{name: "owner role cannot be invited", managerRole: domain.OrganizationRoleOwner, inviteRole: domain.OrganizationRoleOwner, want: domain.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &invitationRepositoryStub{
				findFn: func(context.Context, uint, uint) (*domain.OrganizationMembership, error) {
					return invitationManager(tt.managerRole), nil
				},
				createInvitationFn: func(context.Context, *domain.OrganizationInvitation, time.Time) error {
					t.Fatal("invitation must not persist")
					return nil
				},
			}
			svc := NewService(repo, repo, &invitationMailerStub{}, InvitationPolicy{TTL: 7 * 24 * time.Hour})

			_, err := svc.Invite(context.Background(), 17, 42, &CreateOrganizationInvitationRequest{
				Email: "member@example.com",
				Role:  tt.inviteRole,
			})
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestServiceAcceptInvitationHashesToken(t *testing.T) {
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	repo := &invitationRepositoryStub{
		acceptInvitationFn: func(_ context.Context, tokenHash string, userID uint, acceptedAt time.Time) (*domain.OrganizationMembership, *domain.OrganizationInvitation, error) {
			assert.Equal(t, crypto.SHA256Hex("oinv_test.secret"), tokenHash)
			assert.Equal(t, uint(29), userID)
			assert.Equal(t, now, acceptedAt)
			return &domain.OrganizationMembership{
				OrganizationID: 42,
				UserID:         userID,
				Role:           domain.OrganizationRoleMember,
				Organization:   &domain.Organization{ID: 42, Name: "Acme Europe"},
			}, &domain.OrganizationInvitation{ID: 73, OrganizationID: 42}, nil
		},
	}
	svc := NewService(repo, repo, &invitationMailerStub{}, InvitationPolicy{TTL: 7 * 24 * time.Hour})
	svc.now = func() time.Time { return now }

	membership, err := svc.AcceptInvitation(context.Background(), 29, &AcceptOrganizationInvitationRequest{Token: "oinv_test.secret"})
	require.NoError(t, err)
	assert.Equal(t, uint(42), membership.OrganizationID)
}

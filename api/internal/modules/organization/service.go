package organization

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
	"github.com/zgiai/luas/api/internal/modules/user"
)

// Service defines the organization ownership kernel.
type Service interface {
	Create(ctx context.Context, userID uint, req *CreateOrganizationRequest) (*domain.OrganizationMembership, error)
	List(ctx context.Context, userID uint, page, pageSize int) ([]*domain.OrganizationMembership, int64, error)
	Get(ctx context.Context, userID, organizationID uint) (*domain.OrganizationMembership, error)
	Update(ctx context.Context, userID, organizationID uint, req *UpdateOrganizationRequest) (*domain.OrganizationMembership, error)
	ListMembers(ctx context.Context, userID, organizationID uint, page, pageSize int) ([]*domain.OrganizationMembership, int64, error)
	ChangeMemberRole(ctx context.Context, userID, organizationID, memberID uint, req *UpdateOrganizationMemberRequest) (*domain.OrganizationMembership, error)
	RemoveMember(ctx context.Context, userID, organizationID, memberID uint) error
	TransferOwnership(ctx context.Context, userID, organizationID uint, req *TransferOrganizationOwnershipRequest) (*domain.OrganizationOwnershipTransfer, error)
	Invite(ctx context.Context, userID, organizationID uint, req *CreateOrganizationInvitationRequest) (*OrganizationInvitationResult, error)
	ListInvitations(ctx context.Context, userID, organizationID uint, page, pageSize int) ([]*domain.OrganizationInvitation, int64, error)
	RevokeInvitation(ctx context.Context, userID, organizationID, invitationID uint) error
	AcceptInvitation(ctx context.Context, userID uint, req *AcceptOrganizationInvitationRequest) (*domain.OrganizationMembership, error)
}

type service struct {
	repo             domain.OrganizationRepository
	invitationRepo   domain.OrganizationInvitationRepository
	invitationMailer InvitationMailer
	invitationPolicy InvitationPolicy
	now              func() time.Time
}

var (
	_ Service                   = (*service)(nil)
	_ user.AccountDeletionGuard = (*service)(nil)
)

// NewService creates the organization service and its invitation workflow.
func NewService(
	repo domain.OrganizationRepository,
	invitationRepo domain.OrganizationInvitationRepository,
	invitationMailer InvitationMailer,
	invitationPolicy InvitationPolicy,
) *service {
	if invitationPolicy.TTL <= 0 {
		invitationPolicy = NewInvitationPolicy(nil)
	}
	return &service{
		repo:             repo,
		invitationRepo:   invitationRepo,
		invitationMailer: invitationMailer,
		invitationPolicy: invitationPolicy,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *service) Create(ctx context.Context, userID uint, req *CreateOrganizationRequest) (*domain.OrganizationMembership, error) {
	if userID == 0 || req == nil {
		return nil, domain.ErrInvalidInput
	}
	name := strings.TrimSpace(req.Name)
	if !validOrganizationName(name) {
		return nil, domain.ErrInvalidInput
	}

	slug := req.Slug
	if slug == "" {
		generated, err := crypto.GenerateKeyHex(8)
		if err != nil {
			return nil, fmt.Errorf("generate organization slug: %w", err)
		}
		slug = "org-" + generated
	}
	if !validOrganizationSlug(slug) {
		return nil, domain.ErrInvalidInput
	}

	organization := &domain.Organization{
		Name:      name,
		Slug:      slug,
		CreatedBy: userID,
	}
	owner := &domain.OrganizationMembership{
		UserID: userID,
		Role:   domain.OrganizationRoleOwner,
	}
	if err := s.repo.CreateWithOwner(ctx, organization, owner); err != nil {
		return nil, fmt.Errorf("create organization with owner: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "create",
		Resource:   "organizations",
		TargetType: "organization",
		TargetID:   strconv.FormatUint(uint64(organization.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"role": domain.OrganizationRoleOwner,
		},
	})
	return owner, nil
}

func (s *service) List(ctx context.Context, userID uint, page, pageSize int) ([]*domain.OrganizationMembership, int64, error) {
	if userID == 0 || page < 1 || pageSize < 1 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.repo.ListForUser(ctx, userID, page, pageSize)
}

func (s *service) Get(ctx context.Context, userID, organizationID uint) (*domain.OrganizationMembership, error) {
	if userID == 0 || organizationID == 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.FindForUser(ctx, organizationID, userID)
}

func (s *service) Update(ctx context.Context, userID, organizationID uint, req *UpdateOrganizationRequest) (*domain.OrganizationMembership, error) {
	if userID == 0 || organizationID == 0 || req == nil {
		return nil, domain.ErrInvalidInput
	}
	name := strings.TrimSpace(req.Name)
	if !validOrganizationName(name) {
		return nil, domain.ErrInvalidInput
	}

	membership, err := s.repo.FindForUser(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if membership.Organization == nil {
		return nil, domain.ErrOrganizationNotFound
	}
	if !membership.Role.CanManageOrganization() {
		return nil, domain.ErrPermissionDenied
	}
	if membership.Organization.Name == name {
		return membership, nil
	}

	before := membership.Organization.Name
	membership.Organization.Name = name
	if err := s.repo.Update(ctx, membership.Organization); err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "organizations",
		TargetType: "organization",
		TargetID:   strconv.FormatUint(uint64(organizationID), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"name": {Before: before, After: name},
		},
	})
	return membership, nil
}

// AccountDeletionGuardName identifies the optional starter guard.
func (s *service) AccountDeletionGuardName() string {
	return "organization"
}

// CheckAccountDeletion prevents orphaned or stale memberships after soft deletion.
func (s *service) CheckAccountDeletion(ctx context.Context, userID uint) error {
	if userID == 0 {
		return domain.ErrInvalidInput
	}
	total, owned, err := s.repo.CountMembershipsForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("count organization memberships: %w", err)
	}
	if owned > 0 {
		return domain.ErrOrganizationOwnershipTransferRequired
	}
	if total > 0 {
		return domain.ErrOrganizationMembershipExitRequired
	}
	return nil
}

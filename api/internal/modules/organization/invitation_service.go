package organization

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/capabilities/idgen"
	"github.com/zgiai/luas/api/internal/domain"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

func (s *service) invitationManager(ctx context.Context, userID, organizationID uint) (*domain.OrganizationMembership, error) {
	if s == nil || s.repo == nil {
		return nil, domain.ErrServiceUnavailable
	}
	membership, err := s.repo.FindForUser(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if membership == nil || membership.Organization == nil {
		return nil, domain.ErrOrganizationNotFound
	}
	if !membership.Role.CanManageOrganization() {
		return nil, domain.ErrPermissionDenied
	}
	return membership, nil
}

func generateOrganizationInvitationToken() (string, error) {
	secret, err := crypto.GenerateKeyHex(24)
	if err != nil {
		return "", fmt.Errorf("generate organization invitation token: %w", err)
	}
	return "oinv_" + strings.ToLower(idgen.ShortID()) + "." + secret, nil
}

func (s *service) Invite(
	ctx context.Context,
	userID,
	organizationID uint,
	req *CreateOrganizationInvitationRequest,
) (*OrganizationInvitationResult, error) {
	if userID == 0 || organizationID == 0 || req == nil || len(req.validationErrors()) > 0 {
		return nil, domain.ErrInvalidInput
	}
	if s.invitationRepo == nil {
		return nil, domain.ErrServiceUnavailable
	}
	manager, err := s.invitationManager(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}

	token, err := generateOrganizationInvitationToken()
	if err != nil {
		return nil, err
	}
	now := s.now()
	invitation := &domain.OrganizationInvitation{
		OrganizationID: organizationID,
		InvitedBy:      userID,
		Email:          normalizeInvitationEmail(req.Email),
		Role:           req.Role,
		TokenHash:      crypto.SHA256Hex(token),
		ExpiresAt:      now.Add(s.invitationPolicy.TTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.invitationRepo.CreateInvitation(ctx, invitation, now); err != nil {
		return nil, fmt.Errorf("create organization invitation: %w", err)
	}

	emailStatus := InvitationEmailSendStatusNotConfigured
	if s.invitationMailer != nil && s.invitationMailer.IsConfigured() {
		emailStatus = InvitationEmailSendStatusAcceptedByProvider
		if sendErr := s.invitationMailer.SendInvitation(
			ctx,
			invitation.Email,
			manager.Organization.Name,
			invitation.Role,
			token,
			invitation.ExpiresAt,
		); sendErr != nil {
			emailStatus = InvitationEmailSendStatusFailed
			slog.WarnContext(ctx, "organization.invitation_email_send_failed",
				"organization_id", organizationID,
				"invitation_id", invitation.ID,
				"error_type", fmt.Sprintf("%T", sendErr),
			)
		}
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "invite",
		Resource:   "organization_invitations",
		TargetType: "organization_invitation",
		TargetID:   strconv.FormatUint(uint64(invitation.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"organization_id":   organizationID,
			"role":              invitation.Role,
			"email_send_status": emailStatus,
		},
	})
	return &OrganizationInvitationResult{
		Invitation:      invitation,
		EmailSendStatus: emailStatus,
	}, nil
}

func (s *service) ListInvitations(
	ctx context.Context,
	userID,
	organizationID uint,
	page,
	pageSize int,
) ([]*domain.OrganizationInvitation, int64, error) {
	if userID == 0 || organizationID == 0 || page < 1 || pageSize < 1 {
		return nil, 0, domain.ErrInvalidInput
	}
	if s.invitationRepo == nil {
		return nil, 0, domain.ErrServiceUnavailable
	}
	if _, err := s.invitationManager(ctx, userID, organizationID); err != nil {
		return nil, 0, err
	}
	return s.invitationRepo.ListInvitations(ctx, organizationID, page, pageSize)
}

func (s *service) RevokeInvitation(ctx context.Context, userID, organizationID, invitationID uint) error {
	if userID == 0 || organizationID == 0 || invitationID == 0 {
		return domain.ErrInvalidInput
	}
	if s.invitationRepo == nil {
		return domain.ErrServiceUnavailable
	}
	if _, err := s.invitationManager(ctx, userID, organizationID); err != nil {
		return err
	}
	invitation, err := s.invitationRepo.RevokeInvitation(ctx, organizationID, invitationID, userID, s.now())
	if err != nil {
		return err
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "revoke",
		Resource:   "organization_invitations",
		TargetType: "organization_invitation",
		TargetID:   strconv.FormatUint(uint64(invitation.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"organization_id": organizationID,
			"role":            invitation.Role,
		},
	})
	return nil
}

func (s *service) AcceptInvitation(
	ctx context.Context,
	userID uint,
	req *AcceptOrganizationInvitationRequest,
) (*domain.OrganizationMembership, error) {
	if userID == 0 || req == nil || len(req.validationErrors()) > 0 || len(req.Token) > 256 {
		return nil, domain.ErrInvalidInput
	}
	if s.invitationRepo == nil {
		return nil, domain.ErrServiceUnavailable
	}
	token := strings.TrimSpace(req.Token)
	membership, invitation, err := s.invitationRepo.AcceptInvitation(
		ctx,
		crypto.SHA256Hex(token),
		userID,
		s.now(),
	)
	if err != nil {
		return nil, err
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "accept",
		Resource:   "organization_invitations",
		TargetType: "organization_invitation",
		TargetID:   strconv.FormatUint(uint64(invitation.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"organization_id": invitation.OrganizationID,
			"user_id":         userID,
			"role":            membership.Role,
		},
	})
	return membership, nil
}

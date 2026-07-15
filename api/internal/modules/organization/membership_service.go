package organization

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zgiai/luas/api/internal/domain"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

func (s *service) ListMembers(
	ctx context.Context,
	userID, organizationID uint,
	page, pageSize int,
) ([]*domain.OrganizationMembership, int64, error) {
	if userID == 0 || organizationID == 0 || page < 1 || pageSize < 1 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.repo.ListMembers(ctx, organizationID, userID, page, pageSize)
}

func (s *service) ChangeMemberRole(
	ctx context.Context,
	userID, organizationID, memberID uint,
	req *UpdateOrganizationMemberRequest,
) (*domain.OrganizationMembership, error) {
	if userID == 0 || organizationID == 0 || memberID == 0 || req == nil || len(req.validationErrors()) > 0 {
		return nil, domain.ErrInvalidInput
	}

	change, err := s.repo.ChangeMemberRole(ctx, organizationID, userID, memberID, req.Role, s.now())
	if err != nil {
		return nil, fmt.Errorf("change organization member role: %w", err)
	}
	if change == nil || change.Membership == nil {
		return nil, domain.ErrServiceUnavailable
	}
	if change.BeforeRole != change.Membership.Role {
		auditstarter.RecordChange(ctx, auditstarter.Change{
			Action:     "change_role",
			Resource:   "organization_members",
			TargetType: "organization_membership",
			TargetID:   strconv.FormatUint(uint64(change.Membership.ID), 10),
			Result:     domain.AuditResultSuccess,
			Changes: map[string]domain.AuditValueChange{
				"role": {Before: change.BeforeRole, After: change.Membership.Role},
			},
			Metadata: map[string]any{
				"organization_id": organizationID,
				"user_id":         change.Membership.UserID,
			},
		})
	}
	return change.Membership, nil
}

func (s *service) RemoveMember(ctx context.Context, userID, organizationID, memberID uint) error {
	if userID == 0 || organizationID == 0 || memberID == 0 {
		return domain.ErrInvalidInput
	}

	removed, err := s.repo.RemoveMember(ctx, organizationID, userID, memberID)
	if err != nil {
		return fmt.Errorf("remove organization member: %w", err)
	}
	if removed == nil {
		return domain.ErrServiceUnavailable
	}
	action := "remove"
	if removed.UserID == userID {
		action = "leave"
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     action,
		Resource:   "organization_members",
		TargetType: "organization_membership",
		TargetID:   strconv.FormatUint(uint64(removed.ID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"organization_id": organizationID,
			"user_id":         removed.UserID,
			"role":            removed.Role,
		},
	})
	return nil
}

func (s *service) TransferOwnership(
	ctx context.Context,
	userID, organizationID uint,
	req *TransferOrganizationOwnershipRequest,
) (*domain.OrganizationOwnershipTransfer, error) {
	if userID == 0 || organizationID == 0 || req == nil || len(req.validationErrors()) > 0 {
		return nil, domain.ErrInvalidInput
	}

	transfer, err := s.repo.TransferOwnership(
		ctx,
		organizationID,
		userID,
		req.NewOwnerMemberID,
		s.now(),
	)
	if err != nil {
		return nil, fmt.Errorf("transfer organization ownership: %w", err)
	}
	if transfer == nil || transfer.PreviousOwner == nil || transfer.NewOwner == nil {
		return nil, domain.ErrServiceUnavailable
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "transfer_ownership",
		Resource:   "organizations",
		TargetType: "organization",
		TargetID:   strconv.FormatUint(uint64(organizationID), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"previous_owner_role": {
				Before: domain.OrganizationRoleOwner,
				After:  transfer.PreviousOwner.Role,
			},
			"new_owner_role": {
				Before: transfer.NewOwnerBeforeRole,
				After:  transfer.NewOwner.Role,
			},
		},
		Metadata: map[string]any{
			"previous_owner_member_id": transfer.PreviousOwner.ID,
			"previous_owner_user_id":   transfer.PreviousOwner.UserID,
			"new_owner_member_id":      transfer.NewOwner.ID,
			"new_owner_user_id":        transfer.NewOwner.UserID,
		},
	})
	return transfer, nil
}

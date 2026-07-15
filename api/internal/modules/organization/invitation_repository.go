package organization

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

var _ domain.OrganizationInvitationRepository = (*repository)(nil)

func invitationPendingKey(organizationID uint, email string) string {
	return crypto.SHA256Hex(fmt.Sprintf("%d\x00%s", organizationID, email))
}

func validInvitationEmail(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || value != strings.ToLower(value) {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func validTokenHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validInvitationForCreate(invitation *domain.OrganizationInvitation, now time.Time) bool {
	return invitation != nil &&
		invitation.ID == 0 &&
		invitation.OrganizationID != 0 &&
		invitation.InvitedBy != 0 &&
		invitation.Role.CanBeInvited() &&
		validInvitationEmail(invitation.Email) &&
		validTokenHash(invitation.TokenHash) &&
		invitation.AcceptedAt == nil && invitation.AcceptedBy == nil &&
		invitation.RevokedAt == nil && invitation.RevokedBy == nil &&
		invitation.ExpiresAt.After(now)
}

func (r *repository) CreateInvitation(ctx context.Context, invitation *domain.OrganizationInvitation, now time.Time) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if !validInvitationForCreate(invitation, now) {
		return domain.ErrInvalidInput
	}

	pendingKey := invitationPendingKey(invitation.OrganizationID, invitation.Email)
	invitationPO := newInvitationPO(invitation)
	invitationPO.PendingKey = &pendingKey
	err = db.Transaction(func(tx *gorm.DB) error {
		var memberCount int64
		if countErr := tx.Table("organization_memberships AS memberships").
			Joins("JOIN users ON users.id = memberships.user_id").
			Where("memberships.organization_id = ? AND LOWER(users.email) = ?", invitation.OrganizationID, invitation.Email).
			Count(&memberCount).Error; countErr != nil {
			return countErr
		}
		if memberCount > 0 {
			return domain.ErrOrganizationMemberAlreadyExists
		}

		var existing OrganizationInvitationPO
		findErr := tx.Where("pending_key = ?", pendingKey).First(&existing).Error
		switch {
		case findErr == nil && now.Before(existing.ExpiresAt):
			return domain.ErrOrganizationInvitationAlreadyPending
		case findErr == nil:
			result := tx.Model(&OrganizationInvitationPO{}).
				Where("id = ? AND pending_key = ? AND accepted_at IS NULL AND revoked_at IS NULL", existing.ID, pendingKey).
				UpdateColumn("pending_key", nil)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return domain.ErrOrganizationInvitationAlreadyPending
			}
		case !errors.Is(findErr, gorm.ErrRecordNotFound):
			return findErr
		}

		if createErr := tx.Omit(clause.Associations).Create(invitationPO).Error; createErr != nil {
			if isUniqueViolation(createErr) {
				return domain.ErrOrganizationInvitationAlreadyPending
			}
			return createErr
		}
		return nil
	})
	if err != nil {
		return err
	}

	invitation.ID = invitationPO.ID
	invitation.CreatedAt = invitationPO.CreatedAt
	invitation.UpdatedAt = invitationPO.UpdatedAt
	return nil
}

func (r *repository) ListInvitations(ctx context.Context, organizationID uint, page, pageSize int) ([]*domain.OrganizationInvitation, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || page < 1 || pageSize < 1 {
		return nil, 0, domain.ErrInvalidInput
	}

	query := db.Model(&OrganizationInvitationPO{}).Where("organization_id = ?", organizationID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var invitationPOs []*OrganizationInvitationPO
	if err := query.
		Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&invitationPOs).Error; err != nil {
		return nil, 0, err
	}

	invitations := make([]*domain.OrganizationInvitation, len(invitationPOs))
	for index, invitationPO := range invitationPOs {
		invitations[index] = invitationPO.toDomain()
	}
	return invitations, total, nil
}

func (r *repository) RevokeInvitation(ctx context.Context, organizationID, invitationID, revokedBy uint, now time.Time) (*domain.OrganizationInvitation, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 || invitationID == 0 || revokedBy == 0 {
		return nil, domain.ErrInvalidInput
	}

	var invitation OrganizationInvitationPO
	err = db.Transaction(func(tx *gorm.DB) error {
		if findErr := tx.Where("id = ? AND organization_id = ?", invitationID, organizationID).First(&invitation).Error; findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return domain.ErrOrganizationInvitationNotFound
			}
			return findErr
		}
		if invitation.AcceptedAt != nil || invitation.RevokedAt != nil || !now.Before(invitation.ExpiresAt) {
			return domain.ErrOrganizationInvitationNotFound
		}
		result := tx.Model(&OrganizationInvitationPO{}).
			Where("id = ? AND pending_key IS NOT NULL AND accepted_at IS NULL AND revoked_at IS NULL", invitation.ID).
			Updates(map[string]any{
				"pending_key": nil,
				"revoked_at":  now,
				"revoked_by":  revokedBy,
				"updated_at":  now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrOrganizationInvitationNotFound
		}
		invitation.PendingKey = nil
		invitation.RevokedAt = &now
		invitation.RevokedBy = &revokedBy
		invitation.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return invitation.toDomain(), nil
}

func (r *repository) AcceptInvitation(ctx context.Context, tokenHash string, userID uint, now time.Time) (*domain.OrganizationMembership, *domain.OrganizationInvitation, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	if userID == 0 || !validTokenHash(tokenHash) {
		return nil, nil, domain.ErrOrganizationInvitationInvalid
	}

	var membershipPO OrganizationMembershipPO
	var invitationPO OrganizationInvitationPO
	err = db.Transaction(func(tx *gorm.DB) error {
		findErr := tx.Where("token_hash = ?", tokenHash).First(&invitationPO).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return domain.ErrOrganizationInvitationInvalid
		}
		if findErr != nil {
			return findErr
		}
		if invitationPO.AcceptedAt != nil || invitationPO.RevokedAt != nil {
			return domain.ErrOrganizationInvitationInvalid
		}
		if !now.Before(invitationPO.ExpiresAt) {
			return domain.ErrOrganizationInvitationExpired
		}
		if invitationPO.PendingKey == nil {
			return domain.ErrOrganizationInvitationInvalid
		}

		var invitee user.UserPO
		if userErr := tx.First(&invitee, userID).Error; userErr != nil {
			if errors.Is(userErr, gorm.ErrRecordNotFound) {
				return domain.ErrOrganizationInvitationInvalid
			}
			return userErr
		}
		if !strings.EqualFold(strings.TrimSpace(invitee.Email), invitationPO.Email) {
			return domain.ErrOrganizationInvitationEmailMismatch
		}

		var existingMemberships int64
		if countErr := tx.Model(&OrganizationMembershipPO{}).
			Where("organization_id = ? AND user_id = ?", invitationPO.OrganizationID, userID).
			Count(&existingMemberships).Error; countErr != nil {
			return countErr
		}
		if existingMemberships > 0 {
			return domain.ErrOrganizationMemberAlreadyExists
		}

		membershipPO = OrganizationMembershipPO{
			OrganizationID: invitationPO.OrganizationID,
			UserID:         userID,
			Role:           invitationPO.Role,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if createErr := tx.Omit(clause.Associations).Create(&membershipPO).Error; createErr != nil {
			if isUniqueViolation(createErr) {
				return domain.ErrOrganizationMemberAlreadyExists
			}
			return createErr
		}

		result := tx.Model(&OrganizationInvitationPO{}).
			Where("id = ? AND pending_key IS NOT NULL AND accepted_at IS NULL AND revoked_at IS NULL", invitationPO.ID).
			Updates(map[string]any{
				"pending_key": nil,
				"accepted_at": now,
				"accepted_by": userID,
				"updated_at":  now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrOrganizationInvitationInvalid
		}

		if organizationErr := tx.First(&membershipPO.Organization, invitationPO.OrganizationID).Error; organizationErr != nil {
			return organizationErr
		}
		invitationPO.PendingKey = nil
		invitationPO.AcceptedAt = &now
		invitationPO.AcceptedBy = &userID
		invitationPO.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return membershipPO.toDomain(), invitationPO.toDomain(), nil
}

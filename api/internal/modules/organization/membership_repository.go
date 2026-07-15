package organization

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func (r *repository) ListMembers(
	ctx context.Context,
	organizationID, userID uint,
	page, pageSize int,
) ([]*domain.OrganizationMembership, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || userID == 0 || page < 1 || pageSize < 1 {
		return nil, 0, domain.ErrInvalidInput
	}

	var visible int64
	if err := db.Model(&OrganizationMembershipPO{}).
		Where("organization_id = ? AND user_id = ?", organizationID, userID).
		Count(&visible).Error; err != nil {
		return nil, 0, err
	}
	if visible == 0 {
		return nil, 0, domain.ErrOrganizationNotFound
	}

	query := db.Model(&OrganizationMembershipPO{}).
		Joins("JOIN users AS member_users ON member_users.id = organization_memberships.user_id AND member_users.deleted_at IS NULL").
		Where("organization_memberships.organization_id = ?", organizationID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var memberships []*OrganizationMembershipPO
	if err := query.
		Preload("User").
		Order("organization_memberships.user_id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&memberships).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*domain.OrganizationMembership, len(memberships))
	for index, membership := range memberships {
		items[index] = membership.toDomain()
	}
	return items, total, nil
}

func (r *repository) ChangeMemberRole(
	ctx context.Context,
	organizationID, userID, memberID uint,
	role domain.OrganizationRole,
	now time.Time,
) (*domain.OrganizationMembershipRoleChange, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 || userID == 0 || memberID == 0 || !role.CanBeInvited() || now.IsZero() {
		return nil, domain.ErrInvalidInput
	}

	var change *domain.OrganizationMembershipRoleChange
	err = db.Transaction(func(tx *gorm.DB) error {
		actor, findErr := findMembershipForUserUpdate(tx, organizationID, userID)
		if findErr != nil {
			return findErr
		}
		if !domain.OrganizationRole(actor.Role).CanChangeMemberRoles() {
			return domain.ErrPermissionDenied
		}

		target, findErr := findMembershipByIDUpdate(tx, organizationID, memberID)
		if findErr != nil {
			return findErr
		}
		if target.User.ID == 0 {
			return domain.ErrOrganizationMemberNotFound
		}
		before := domain.OrganizationRole(target.Role)
		if before == domain.OrganizationRoleOwner {
			return domain.ErrOrganizationOwnershipTransferRequired
		}
		if before == role {
			change = &domain.OrganizationMembershipRoleChange{
				Membership: target.toDomain(),
				BeforeRole: before,
			}
			return nil
		}

		result := tx.Model(&OrganizationMembershipPO{}).
			Where("id = ? AND organization_id = ? AND role = ?", target.ID, organizationID, target.Role).
			Updates(map[string]any{"role": role, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrConflict
		}
		target.Role = string(role)
		target.UpdatedAt = now
		change = &domain.OrganizationMembershipRoleChange{
			Membership: target.toDomain(),
			BeforeRole: before,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (r *repository) RemoveMember(
	ctx context.Context,
	organizationID, userID, memberID uint,
) (*domain.OrganizationMembership, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 || userID == 0 || memberID == 0 {
		return nil, domain.ErrInvalidInput
	}

	var removed *domain.OrganizationMembership
	err = db.Transaction(func(tx *gorm.DB) error {
		actor, findErr := findMembershipForUserUpdate(tx, organizationID, userID)
		if findErr != nil {
			return findErr
		}
		target, findErr := findMembershipByIDUpdate(tx, organizationID, memberID)
		if findErr != nil {
			return findErr
		}

		actorRole := domain.OrganizationRole(actor.Role)
		targetRole := domain.OrganizationRole(target.Role)
		if actor.ID == target.ID {
			if actorRole == domain.OrganizationRoleOwner {
				return domain.ErrOrganizationOwnershipTransferRequired
			}
		} else if !actorRole.CanRemoveMember(targetRole) {
			return domain.ErrPermissionDenied
		}

		result := tx.Where("id = ? AND organization_id = ? AND role = ?", target.ID, organizationID, target.Role).
			Delete(&OrganizationMembershipPO{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrOrganizationMemberNotFound
		}
		removed = target.toDomain()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return removed, nil
}

func (r *repository) TransferOwnership(
	ctx context.Context,
	organizationID, userID, newOwnerMemberID uint,
	now time.Time,
) (*domain.OrganizationOwnershipTransfer, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == 0 || userID == 0 || newOwnerMemberID == 0 || now.IsZero() {
		return nil, domain.ErrInvalidInput
	}

	var transfer *domain.OrganizationOwnershipTransfer
	err = db.Transaction(func(tx *gorm.DB) error {
		actor, findErr := findMembershipForUserUpdate(tx, organizationID, userID)
		if findErr != nil {
			return findErr
		}
		if domain.OrganizationRole(actor.Role) != domain.OrganizationRoleOwner {
			return domain.ErrPermissionDenied
		}

		target, findErr := findMembershipByIDUpdate(tx, organizationID, newOwnerMemberID)
		if findErr != nil {
			return findErr
		}
		targetRole := domain.OrganizationRole(target.Role)
		if actor.ID == target.ID || !targetRole.CanBeInvited() || target.User.ID == 0 {
			return domain.ErrOrganizationOwnershipTransferTargetInvalid
		}

		previousOwner := tx.Model(&OrganizationMembershipPO{}).
			Where("id = ? AND organization_id = ? AND role = ?", actor.ID, organizationID, domain.OrganizationRoleOwner).
			Updates(map[string]any{"role": domain.OrganizationRoleAdmin, "updated_at": now})
		if previousOwner.Error != nil {
			return previousOwner.Error
		}
		if previousOwner.RowsAffected != 1 {
			return domain.ErrPermissionDenied
		}

		newOwner := tx.Model(&OrganizationMembershipPO{}).
			Where("id = ? AND organization_id = ? AND role = ?", target.ID, organizationID, target.Role).
			Updates(map[string]any{"role": domain.OrganizationRoleOwner, "updated_at": now})
		if newOwner.Error != nil {
			return newOwner.Error
		}
		if newOwner.RowsAffected != 1 {
			return domain.ErrOrganizationOwnershipTransferTargetInvalid
		}

		actor.Role = string(domain.OrganizationRoleAdmin)
		actor.UpdatedAt = now
		target.Role = string(domain.OrganizationRoleOwner)
		target.UpdatedAt = now
		transfer = &domain.OrganizationOwnershipTransfer{
			PreviousOwner:      actor.toDomain(),
			NewOwner:           target.toDomain(),
			NewOwnerBeforeRole: targetRole,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return transfer, nil
}

func (r *repository) CountMembershipsForUser(ctx context.Context, userID uint) (int64, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return 0, 0, err
	}
	if userID == 0 {
		return 0, 0, domain.ErrInvalidInput
	}

	var counts struct {
		Total int64
		Owned int64
	}
	err = db.Model(&OrganizationMembershipPO{}).
		Select(
			"COUNT(*) AS total, COALESCE(SUM(CASE WHEN role = ? THEN 1 ELSE 0 END), 0) AS owned",
			domain.OrganizationRoleOwner,
		).
		Where("user_id = ?", userID).
		Scan(&counts).Error
	return counts.Total, counts.Owned, err
}

func findMembershipForUserUpdate(tx *gorm.DB, organizationID, userID uint) (*OrganizationMembershipPO, error) {
	var membership OrganizationMembershipPO
	err := forUpdate(tx).
		Preload("User").
		Where("organization_id = ? AND user_id = ?", organizationID, userID).
		First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func findMembershipByIDUpdate(tx *gorm.DB, organizationID, memberID uint) (*OrganizationMembershipPO, error) {
	var membership OrganizationMembershipPO
	err := forUpdate(tx).
		Preload("User").
		Where("organization_id = ? AND id = ?", organizationID, memberID).
		First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrOrganizationMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func findUndeletedUserForUpdate(tx *gorm.DB, userID uint) (*user.UserPO, error) {
	var po user.UserPO
	if err := forUpdate(tx).First(&po, userID).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

func forUpdate(tx *gorm.DB) *gorm.DB {
	if tx == nil || tx.Dialector == nil || tx.Name() == "sqlite" {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}

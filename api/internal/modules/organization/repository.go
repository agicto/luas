package organization

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
)

type repository struct {
	db *gorm.DB
}

var _ domain.OrganizationRepository = (*repository)(nil)

// NewRepository creates the organization repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return r.db.WithContext(ctx), nil
}

func (r *repository) CreateWithOwner(ctx context.Context, organization *domain.Organization, owner *domain.OrganizationMembership) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if organization == nil || owner == nil ||
		organization.ID != 0 || owner.ID != 0 || owner.OrganizationID != 0 ||
		organization.CreatedBy == 0 || owner.UserID != organization.CreatedBy ||
		owner.Role != domain.OrganizationRoleOwner ||
		!validOrganizationName(organization.Name) || !validOrganizationSlug(organization.Slug) {
		return domain.ErrInvalidInput
	}

	organizationPO := newOrganizationPO(organization)
	ownerPO := newMembershipPO(owner)
	err = db.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Omit(clause.Associations).Create(organizationPO).Error; createErr != nil {
			if isUniqueViolation(createErr) {
				return domain.ErrOrganizationSlugAlreadyExists
			}
			return createErr
		}

		ownerPO.OrganizationID = organizationPO.ID
		if createErr := tx.Omit(clause.Associations).Create(ownerPO).Error; createErr != nil {
			return createErr
		}
		return nil
	})
	if err != nil {
		return err
	}

	organization.ID = organizationPO.ID
	organization.CreatedAt = organizationPO.CreatedAt
	organization.UpdatedAt = organizationPO.UpdatedAt
	owner.ID = ownerPO.ID
	owner.OrganizationID = organizationPO.ID
	owner.CreatedAt = ownerPO.CreatedAt
	owner.UpdatedAt = ownerPO.UpdatedAt
	owner.Organization = organization
	return nil
}

func (r *repository) FindForUser(ctx context.Context, organizationID, userID uint) (*domain.OrganizationMembership, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}

	var membership OrganizationMembershipPO
	err = db.
		Preload("Organization").
		Where("organization_id = ? AND user_id = ?", organizationID, userID).
		First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}
	return membership.toDomain(), nil
}

func (r *repository) ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.OrganizationMembership, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := db.Model(&OrganizationMembershipPO{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var memberships []*OrganizationMembershipPO
	offset := (page - 1) * pageSize
	if err := query.
		Preload("Organization").
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&memberships).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*domain.OrganizationMembership, len(memberships))
	for i, membership := range memberships {
		result[i] = membership.toDomain()
	}
	return result, total, nil
}

func (r *repository) Update(ctx context.Context, organization *domain.Organization) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if organization == nil || organization.ID == 0 {
		return domain.ErrInvalidInput
	}

	updatedAt := time.Now().UTC()
	result := db.Model(&OrganizationPO{}).
		Where("id = ?", organization.ID).
		Updates(map[string]any{"name": organization.Name, "updated_at": updatedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrOrganizationNotFound
	}
	organization.UpdatedAt = updatedAt
	return nil
}

func (r *repository) CountOwnedByUser(ctx context.Context, userID uint) (int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	err = db.Model(&OrganizationMembershipPO{}).
		Where("user_id = ? AND role = ?", userID, domain.OrganizationRoleOwner).
		Count(&count).Error
	return count, err
}

func isUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var stateError interface{ SQLState() string }
	if errors.As(err, &stateError) && stateError.SQLState() == "23505" {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") || strings.Contains(message, "duplicate key")
}

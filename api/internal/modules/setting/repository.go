package setting

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type settingStore interface {
	List(context.Context, domain.SettingTarget, []string) (map[string]*settingRecord, error)
	Set(context.Context, domain.SettingTarget, string, string, uint64) (*settingRecord, bool, error)
	Reset(context.Context, domain.SettingTarget, string, uint64) (uint64, bool, error)
	DeleteForUser(context.Context, uint) error
}

type repository struct {
	db  *gorm.DB
	now func() time.Time
}

var _ settingStore = (*repository)(nil)

// NewRepository creates the setting persistence adapter.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db, now: time.Now}
}

func (r *repository) List(
	ctx context.Context,
	target domain.SettingTarget,
	keys []string,
) (map[string]*settingRecord, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	if !target.IsValid() || len(keys) == 0 || len(keys) > maxSettingDefinitions {
		return nil, domain.ErrInvalidInput
	}

	var rows []SettingPO
	if err := db.
		Where("scope = ? AND subject_id = ? AND key IN ?", target.Scope, target.SubjectID, keys).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list setting overrides: %w", err)
	}
	result := make(map[string]*settingRecord, len(rows))
	for index := range rows {
		result[rows[index].Key] = recordFromPO(&rows[index])
	}
	return result, nil
}

func (r *repository) Set(
	ctx context.Context,
	target domain.SettingTarget,
	key string,
	valueJSON string,
	expectedVersion uint64,
) (*settingRecord, bool, error) {
	if !target.IsValid() || key == "" || valueJSON == "" {
		return nil, false, domain.ErrInvalidInput
	}
	var result *settingRecord
	changed := false
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		if err := lockSettingOwner(tx, target); err != nil {
			return err
		}

		po, err := findSettingForUpdate(tx, target, key)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedVersion != 0 {
				return domain.ErrSettingVersionConflict
			}
			po = newSettingPO(target, key, valueJSON, true, 1)
			create := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(po)
			if create.Error != nil {
				return fmt.Errorf("create setting override: %w", create.Error)
			}
			if create.RowsAffected != 1 {
				return domain.ErrSettingVersionConflict
			}
			result = recordFromPO(po)
			changed = true
			return nil
		}
		if err != nil {
			return fmt.Errorf("find setting override: %w", err)
		}
		if po.Version != expectedVersion {
			return domain.ErrSettingVersionConflict
		}
		if po.IsOverridden && po.ValueJSON == valueJSON {
			result = recordFromPO(po)
			return nil
		}

		now := r.now().UTC()
		update := tx.Model(&SettingPO{}).
			Where("id = ? AND version = ?", po.ID, expectedVersion).
			Updates(map[string]any{
				"value_json":    valueJSON,
				"is_overridden": true,
				"version":       expectedVersion + 1,
				"updated_at":    now,
			})
		if update.Error != nil {
			return fmt.Errorf("update setting override: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrSettingVersionConflict
		}
		po.ValueJSON = valueJSON
		po.IsOverridden = true
		po.Version = expectedVersion + 1
		po.UpdatedAt = now
		result = recordFromPO(po)
		changed = true
		return nil
	})
	return result, changed, err
}

func (r *repository) Reset(
	ctx context.Context,
	target domain.SettingTarget,
	key string,
	expectedVersion uint64,
) (uint64, bool, error) {
	if !target.IsValid() || key == "" {
		return 0, false, domain.ErrInvalidInput
	}
	version := uint64(0)
	changed := false
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		if err := lockSettingOwner(tx, target); err != nil {
			return err
		}
		po, err := findSettingForUpdate(tx, target, key)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedVersion != 0 {
				return domain.ErrSettingVersionConflict
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("find setting override: %w", err)
		}
		if po.Version != expectedVersion {
			return domain.ErrSettingVersionConflict
		}
		version = po.Version
		if !po.IsOverridden {
			return nil
		}

		version = expectedVersion + 1
		now := r.now().UTC()
		update := tx.Model(&SettingPO{}).
			Where("id = ? AND version = ?", po.ID, expectedVersion).
			Updates(map[string]any{
				"value_json":    "",
				"is_overridden": false,
				"version":       version,
				"updated_at":    now,
			})
		if update.Error != nil {
			return fmt.Errorf("reset setting override: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrSettingVersionConflict
		}
		changed = true
		return nil
	})
	return version, changed, err
}

func (r *repository) DeleteForUser(ctx context.Context, userID uint) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if userID == 0 {
		return domain.ErrInvalidInput
	}
	if err := db.Where("scope = ? AND user_id = ?", domain.SettingScopeUser, userID).
		Delete(&SettingPO{}).Error; err != nil {
		return fmt.Errorf("delete user settings: %w", err)
	}
	return nil
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	db := infradatabase.ResolveContextDB(ctx, r.db)
	if db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return db, nil
}

func (r *repository) inTransaction(ctx context.Context, operation func(*gorm.DB) error) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if _, bound := infradatabase.TransactionFromContext(ctx); bound {
		return operation(db)
	}
	return db.Transaction(operation)
}

func lockSettingOwner(tx *gorm.DB, target domain.SettingTarget) error {
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"})
	switch target.Scope {
	case domain.SettingScopeApp:
		return nil
	case domain.SettingScopeUser:
		var owner user.UserPO
		if err := query.First(&owner, target.SubjectID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("lock setting user: %w", err)
		}
		return nil
	case domain.SettingScopeOrganization:
		var owner organization.OrganizationPO
		if err := query.First(&owner, target.SubjectID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrOrganizationNotFound
			}
			return fmt.Errorf("lock setting organization: %w", err)
		}
		return nil
	default:
		return domain.ErrInvalidInput
	}
}

func findSettingForUpdate(
	tx *gorm.DB,
	target domain.SettingTarget,
	key string,
) (*SettingPO, error) {
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"})
	var po SettingPO
	if err := query.
		Where("scope = ? AND subject_id = ? AND key = ?", target.Scope, target.SubjectID, key).
		First(&po).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

func newSettingPO(
	target domain.SettingTarget,
	key string,
	valueJSON string,
	isOverridden bool,
	version uint64,
) *SettingPO {
	po := &SettingPO{
		Scope:        string(target.Scope),
		SubjectID:    target.SubjectID,
		Key:          key,
		ValueJSON:    valueJSON,
		IsOverridden: isOverridden,
		Version:      version,
	}
	switch target.Scope {
	case domain.SettingScopeApp:
		// App settings use subject 0 and have no owner foreign key.
	case domain.SettingScopeUser:
		id := target.SubjectID
		po.UserID = &id
	case domain.SettingScopeOrganization:
		id := target.SubjectID
		po.OrganizationID = &id
	}
	return po
}

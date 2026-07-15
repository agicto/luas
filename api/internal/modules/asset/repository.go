package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type createIntentResult struct {
	Asset   *domain.Asset
	Created bool
}

type claimedAsset struct {
	Asset   *domain.Asset
	Claimed bool
}

type assetStore interface {
	CreateIntent(context.Context, *domain.Asset) (*createIntentResult, error)
	ListForUser(context.Context, uint, string, int, int) ([]*domain.Asset, int64, error)
	FindForUser(context.Context, uint, string) (*domain.Asset, error)
	FindByID(context.Context, string) (*domain.Asset, error)
	ClaimCompletion(context.Context, uint, string, string, time.Time, time.Time) (*claimedAsset, error)
	ClaimDeletion(context.Context, uint, string, string, time.Time, time.Time) (*claimedAsset, error)
	ClaimPrune(context.Context, string, string, time.Time, time.Time) (*claimedAsset, error)
	ReleaseOperation(context.Context, string, string, time.Time) error
	MarkReady(context.Context, string, string, string, time.Time) (*domain.Asset, error)
	MarkRejected(context.Context, string, string, string, time.Time) (*domain.Asset, error)
	MarkDeleted(context.Context, string, string, time.Time) (*domain.Asset, error)
	FinishPrune(context.Context, string, string, string, bool, time.Time) error
	CountActiveForUser(context.Context, uint) (int64, error)
	ListCleanupCandidates(context.Context, time.Time, int) ([]*domain.Asset, error)
}

type repository struct {
	db *gorm.DB
}

var _ assetStore = (*repository)(nil)

// NewRepository creates the asset persistence adapter.
func NewRepository(db *gorm.DB) *repository { return &repository{db: db} }

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return infradatabase.ResolveContextDB(ctx, r.db), nil
}

func (r *repository) CreateIntent(ctx context.Context, value *domain.Asset) (*createIntentResult, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	po := assetToPO(value)
	if po == nil {
		return nil, domain.ErrInvalidInput
	}

	result := &createIntentResult{}
	err = db.Transaction(func(tx *gorm.DB) error {
		var owner user.UserPO
		ownerQuery := tx.Select("id")
		if tx.Dialector != nil && tx.Name() != "sqlite" {
			ownerQuery = ownerQuery.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if ownerErr := ownerQuery.First(&owner, po.UserID).Error; ownerErr != nil {
			if errors.Is(ownerErr, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("find asset owner: %w", ownerErr)
		}

		create := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "idempotency_key"}},
			DoNothing: true,
		}).Create(po)
		if create.Error != nil {
			return fmt.Errorf("create asset intent: %w", create.Error)
		}
		if create.RowsAffected > 0 {
			result.Asset = assetFromPO(po)
			result.Created = true
			return nil
		}

		var existing AssetPO
		if existingErr := tx.Where("user_id = ? AND idempotency_key = ?", po.UserID, po.IdempotencyKey).
			First(&existing).Error; existingErr != nil {
			return fmt.Errorf("find idempotent asset: %w", existingErr)
		}
		if existing.RequestHash != po.RequestHash {
			return domain.ErrAssetIdempotencyConflict
		}
		result.Asset = assetFromPO(&existing)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *repository) ListForUser(
	ctx context.Context,
	userID uint,
	status string,
	page int,
	pageSize int,
) ([]*domain.Asset, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	query := db.Model(&AssetPO{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)
	if status != "all" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count assets: %w", err)
	}
	var rows []AssetPO
	if err := query.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list assets: %w", err)
	}
	items := make([]*domain.Asset, len(rows))
	for index := range rows {
		items[index] = assetFromPO(&rows[index])
	}
	return items, total, nil
}

func (r *repository) FindForUser(ctx context.Context, userID uint, id string) (*domain.Asset, error) {
	db, dbErr := r.withContext(ctx)
	if dbErr != nil {
		return nil, dbErr
	}
	var po AssetPO
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAssetNotFound
		}
		return nil, fmt.Errorf("find owned asset: %w", err)
	}
	return assetFromPO(&po), nil
}

func (r *repository) FindByID(ctx context.Context, id string) (*domain.Asset, error) {
	db, dbErr := r.withContext(ctx)
	if dbErr != nil {
		return nil, dbErr
	}
	var po AssetPO
	if err := db.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAssetNotFound
		}
		return nil, fmt.Errorf("find asset transfer: %w", err)
	}
	return assetFromPO(&po), nil
}

func (r *repository) ClaimCompletion(
	ctx context.Context,
	userID uint,
	id string,
	token string,
	now time.Time,
	until time.Time,
) (*claimedAsset, error) {
	value, err := r.FindForUser(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if value.DeletedAt != nil || value.Status == domain.AssetStatusDeleted {
		return nil, domain.ErrAssetNotFound
	}
	if value.Status == domain.AssetStatusReady {
		return &claimedAsset{Asset: value}, nil
	}
	if value.Status != domain.AssetStatusPending {
		return nil, domain.ErrAssetNotReady
	}
	if !value.PendingExpiresAt.After(now) {
		return nil, domain.ErrAssetUploadExpired
	}
	return r.claim(ctx, id, userID, operationComplete, token, now, until, "status = ?", domain.AssetStatusPending)
}

func (r *repository) ClaimDeletion(
	ctx context.Context,
	userID uint,
	id string,
	token string,
	now time.Time,
	until time.Time,
) (*claimedAsset, error) {
	value, err := r.FindForUser(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if value.DeletedAt != nil || value.Status == domain.AssetStatusDeleted {
		return &claimedAsset{Asset: value}, nil
	}
	return r.claim(ctx, id, userID, operationDelete, token, now, until, "deleted_at IS NULL")
}

func (r *repository) ClaimPrune(
	ctx context.Context,
	id string,
	token string,
	now time.Time,
	until time.Time,
) (*claimedAsset, error) {
	return r.claim(ctx, id, 0, operationPrune, token, now, until,
		"(staging_key <> '' OR object_key <> '') AND pending_expires_at <= ? AND (status = ? OR status = ? OR status = ? OR (status = ? AND staging_key <> ''))",
		now, domain.AssetStatusPending, domain.AssetStatusRejected, domain.AssetStatusDeleted, domain.AssetStatusReady,
	)
}

func (r *repository) claim(
	ctx context.Context,
	id string,
	userID uint,
	operation string,
	token string,
	now time.Time,
	until time.Time,
	extraWhere string,
	extraArgs ...any,
) (*claimedAsset, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	query := db.Model(&AssetPO{}).
		Where("id = ?", id).
		Where("operation_token = '' OR operation_until IS NULL OR operation_until <= ?", now).
		Where(extraWhere, extraArgs...)
	if userID != 0 {
		query = query.Where("user_id = ?", userID)
	}
	update := query.Updates(map[string]any{
		"operation_kind":  operation,
		"operation_token": token,
		"operation_until": until,
		"updated_at":      now,
	})
	if update.Error != nil {
		return nil, fmt.Errorf("claim asset operation: %w", update.Error)
	}
	if update.RowsAffected == 0 {
		return nil, domain.ErrAssetNotReady
	}
	var po AssetPO
	if err := db.Where("id = ? AND operation_token = ?", id, token).First(&po).Error; err != nil {
		return nil, fmt.Errorf("reload claimed asset: %w", err)
	}
	return &claimedAsset{Asset: assetFromPO(&po), Claimed: true}, nil
}

func (r *repository) ReleaseOperation(ctx context.Context, id string, token string, now time.Time) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	result := db.Model(&AssetPO{}).
		Where("id = ? AND operation_token = ?", id, token).
		Updates(map[string]any{
			"operation_kind":  "",
			"operation_token": "",
			"operation_until": nil,
			"updated_at":      now,
		})
	if result.Error != nil {
		return fmt.Errorf("release asset operation: %w", result.Error)
	}
	return nil
}

func (r *repository) MarkReady(
	ctx context.Context,
	id string,
	token string,
	checksum string,
	now time.Time,
) (*domain.Asset, error) {
	return r.finish(ctx, id, token, map[string]any{
		"status":          domain.AssetStatusReady,
		"checksum_sha256": checksum,
		"rejection_code":  "",
		"ready_at":        now,
		"operation_kind":  "",
		"operation_token": "",
		"operation_until": nil,
		"updated_at":      now,
	})
}

func (r *repository) MarkRejected(
	ctx context.Context,
	id string,
	token string,
	code string,
	now time.Time,
) (*domain.Asset, error) {
	values := map[string]any{
		"status":          domain.AssetStatusRejected,
		"rejection_code":  code,
		"checksum_sha256": "",
		"operation_kind":  "",
		"operation_token": "",
		"operation_until": nil,
		"updated_at":      now,
	}
	return r.finish(ctx, id, token, values)
}

func (r *repository) MarkDeleted(
	ctx context.Context,
	id string,
	token string,
	now time.Time,
) (*domain.Asset, error) {
	return r.finish(ctx, id, token, map[string]any{
		"status":          domain.AssetStatusDeleted,
		"checksum_sha256": "",
		"deleted_at":      now,
		"operation_kind":  "",
		"operation_token": "",
		"operation_until": nil,
		"updated_at":      now,
	})
}

func (r *repository) finish(
	ctx context.Context,
	id string,
	token string,
	values map[string]any,
) (*domain.Asset, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	update := db.Model(&AssetPO{}).
		Where("id = ? AND operation_token = ?", id, token).
		Updates(values)
	if update.Error != nil {
		return nil, fmt.Errorf("finish asset operation: %w", update.Error)
	}
	if update.RowsAffected == 0 {
		return nil, domain.ErrAssetNotReady
	}
	var po AssetPO
	if err := db.Where("id = ?", id).First(&po).Error; err != nil {
		return nil, fmt.Errorf("reload completed asset: %w", err)
	}
	return assetFromPO(&po), nil
}

func (r *repository) FinishPrune(
	ctx context.Context,
	id string,
	token string,
	rejectionCode string,
	preserveFinal bool,
	now time.Time,
) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	values := map[string]any{
		"status":          gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", domain.AssetStatusPending, domain.AssetStatusRejected),
		"rejection_code":  gorm.Expr("CASE WHEN status = ? THEN ? ELSE rejection_code END", domain.AssetStatusPending, rejectionCode),
		"staging_key":     "",
		"operation_kind":  "",
		"operation_token": "",
		"operation_until": nil,
		"updated_at":      now,
	}
	if !preserveFinal {
		values["object_key"] = ""
	}
	update := db.Model(&AssetPO{}).
		Where("id = ? AND operation_token = ?", id, token).
		Updates(values)
	if update.Error != nil {
		return fmt.Errorf("finish asset prune: %w", update.Error)
	}
	if update.RowsAffected == 0 {
		return domain.ErrAssetNotReady
	}
	return nil
}

func (r *repository) CountActiveForUser(ctx context.Context, userID uint) (int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := db.Model(&AssetPO{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count active assets: %w", err)
	}
	return count, nil
}

func (r *repository) ListCleanupCandidates(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]*domain.Asset, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var rows []AssetPO
	if err := db.
		Where("staging_key <> '' OR object_key <> ''").
		Where("pending_expires_at <= ? AND (status = ? OR status = ? OR status = ? OR (status = ? AND staging_key <> ''))", now, domain.AssetStatusPending, domain.AssetStatusRejected, domain.AssetStatusDeleted, domain.AssetStatusReady).
		Where("operation_token = '' OR operation_until IS NULL OR operation_until <= ?", now).
		Order("pending_expires_at ASC, id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list asset cleanup candidates: %w", err)
	}
	items := make([]*domain.Asset, len(rows))
	for index := range rows {
		items[index] = assetFromPO(&rows[index])
	}
	return items, nil
}

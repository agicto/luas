package usage

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

type usageMutation struct {
	Operation      string
	Source         string
	EventID        string
	Fingerprint    string
	Target         domain.UsageTarget
	Metric         string
	Quantity       int64
	DimensionsJSON string
	OccurredAt     time.Time
	PeriodStart    time.Time
	PeriodEnd      time.Time
	DefaultLimit   *int64
}

type usageStore interface {
	ListCurrent(
		context.Context,
		domain.UsageTarget,
		time.Time,
		[]string,
	) (map[string]*usageCounterRecord, map[string]*usageQuotaRecord, error)
	Apply(context.Context, usageMutation) (*domain.UsageReceipt, error)
	SetQuota(
		context.Context,
		domain.UsageTarget,
		string,
		int64,
		uint64,
	) (*usageQuotaRecord, bool, error)
	ResetQuota(
		context.Context,
		domain.UsageTarget,
		string,
		uint64,
	) (*usageQuotaRecord, bool, error)
	PruneReceipts(context.Context, time.Time) (int64, error)
	DeleteForUser(context.Context, uint) error
}

type repository struct {
	db  *gorm.DB
	now func() time.Time
}

var _ usageStore = (*repository)(nil)

// NewRepository creates the relational usage persistence adapter.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db, now: time.Now}
}

func (r *repository) ListCurrent(
	ctx context.Context,
	target domain.UsageTarget,
	at time.Time,
	metrics []string,
) (map[string]*usageCounterRecord, map[string]*usageQuotaRecord, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !target.IsValid() || at.IsZero() || len(metrics) == 0 || len(metrics) > maxUsageDefinitions {
		return nil, nil, domain.ErrInvalidInput
	}

	var counters []UsageCounterPO
	if err := db.
		Where(
			"scope = ? AND subject_id = ? AND metric IN ? AND period_start <= ? AND period_end > ?",
			target.Scope,
			target.SubjectID,
			metrics,
			at,
			at,
		).
		Find(&counters).Error; err != nil {
		return nil, nil, fmt.Errorf("list current usage counters: %w", err)
	}
	var quotas []UsageQuotaPO
	if err := db.
		Where("scope = ? AND subject_id = ? AND metric IN ?", target.Scope, target.SubjectID, metrics).
		Find(&quotas).Error; err != nil {
		return nil, nil, fmt.Errorf("list usage quotas: %w", err)
	}

	counterResult := make(map[string]*usageCounterRecord, len(counters))
	for index := range counters {
		row := &counters[index]
		counterResult[row.Metric] = &usageCounterRecord{
			Value:     row.Value,
			Version:   row.Version,
			UpdatedAt: row.UpdatedAt,
		}
	}
	quotaResult := make(map[string]*usageQuotaRecord, len(quotas))
	for index := range quotas {
		row := &quotas[index]
		quotaResult[row.Metric] = quotaRecordFromPO(row)
	}
	return counterResult, quotaResult, nil
}

func (r *repository) Apply(
	ctx context.Context,
	mutation usageMutation,
) (*domain.UsageReceipt, error) {
	if err := validateUsageMutation(mutation); err != nil {
		return nil, err
	}
	var result *domain.UsageReceipt
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		po := newUsageEventPO(mutation)
		create := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(po)
		if create.Error != nil {
			return fmt.Errorf("reserve usage event receipt: %w", create.Error)
		}
		if create.RowsAffected == 0 {
			existing, err := findUsageEvent(tx, mutation.Source, mutation.EventID)
			if err != nil {
				return err
			}
			if existing.Fingerprint != mutation.Fingerprint {
				return domain.ErrUsageIdempotencyConflict
			}
			result, err = receiptFromPO(existing, true)
			return err
		}

		if err := lockUsageOwner(tx, mutation.Target); err != nil {
			return err
		}
		quota, err := findUsageQuotaForUpdate(tx, mutation.Target, mutation.Metric)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find usage quota: %w", err)
		}
		limit := cloneInt64(mutation.DefaultLimit)
		if err == nil && quota.IsOverridden {
			limit = cloneInt64(quota.LimitValue)
		}

		counter, err := findUsageCounterForUpdate(
			tx,
			mutation.Target,
			mutation.Metric,
			mutation.PeriodStart,
		)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find usage counter: %w", err)
		}
		before := int64(0)
		if err == nil {
			before = counter.Value
		}
		after, valid := addSafeUsageQuantity(before, mutation.Quantity)
		if !valid {
			return domain.ErrUsageInvalidEvent
		}
		decision := domain.UsageDecisionAccepted
		if mutation.Operation == usageOperationConsume && limit != nil && after > *limit {
			decision = domain.UsageDecisionDenied
			after = before
		}
		if decision == domain.UsageDecisionAccepted {
			if counter == nil {
				counter = newUsageCounterPO(mutation, after)
				if createErr := tx.Create(counter).Error; createErr != nil {
					return fmt.Errorf("create usage counter: %w", createErr)
				}
			} else {
				now := r.now().UTC()
				update := tx.Model(&UsageCounterPO{}).
					Where("id = ? AND version = ?", counter.ID, counter.Version).
					Updates(map[string]any{
						"value":      after,
						"version":    counter.Version + 1,
						"updated_at": now,
					})
				if update.Error != nil {
					return fmt.Errorf("update usage counter: %w", update.Error)
				}
				if update.RowsAffected != 1 {
					return domain.ErrServiceUnavailable
				}
			}
		}

		finalize := tx.Model(&UsageEventPO{}).
			Where("id = ? AND decision = ?", po.ID, usageDecisionPending).
			Updates(map[string]any{
				"decision":       decision,
				"counter_before": before,
				"counter_after":  after,
				"limit_value":    limit,
				"updated_at":     r.now().UTC(),
			})
		if finalize.Error != nil {
			return fmt.Errorf("finalize usage event receipt: %w", finalize.Error)
		}
		if finalize.RowsAffected != 1 {
			return domain.ErrServiceUnavailable
		}
		po.Decision = string(decision)
		po.CounterBefore = before
		po.CounterAfter = after
		po.LimitValue = cloneInt64(limit)
		result, err = receiptFromPO(po, false)
		return err
	})
	return result, err
}

func (r *repository) SetQuota(
	ctx context.Context,
	target domain.UsageTarget,
	metric string,
	limit int64,
	expectedVersion uint64,
) (*usageQuotaRecord, bool, error) {
	if !target.IsValid() || metric == "" || limit < 0 || limit > maxSafeUsageInteger {
		return nil, false, domain.ErrInvalidInput
	}
	var result *usageQuotaRecord
	changed := false
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		if err := lockUsageOwner(tx, target); err != nil {
			return err
		}
		po, err := findUsageQuotaForUpdate(tx, target, metric)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedVersion != 0 {
				return domain.ErrUsageQuotaVersionConflict
			}
			po = newUsageQuotaPO(target, metric, &limit, true, 1)
			create := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(po)
			if create.Error != nil {
				return fmt.Errorf("create usage quota override: %w", create.Error)
			}
			if create.RowsAffected != 1 {
				return domain.ErrUsageQuotaVersionConflict
			}
			result = quotaRecordFromPO(po)
			changed = true
			return nil
		}
		if err != nil {
			return fmt.Errorf("find usage quota override: %w", err)
		}
		if po.Version != expectedVersion {
			return domain.ErrUsageQuotaVersionConflict
		}
		if po.IsOverridden && po.LimitValue != nil && *po.LimitValue == limit {
			result = quotaRecordFromPO(po)
			return nil
		}
		now := r.now().UTC()
		update := tx.Model(&UsageQuotaPO{}).
			Where("id = ? AND version = ?", po.ID, expectedVersion).
			Updates(map[string]any{
				"limit_value":   limit,
				"is_overridden": true,
				"version":       expectedVersion + 1,
				"updated_at":    now,
			})
		if update.Error != nil {
			return fmt.Errorf("update usage quota override: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrUsageQuotaVersionConflict
		}
		po.LimitValue = cloneInt64(&limit)
		po.IsOverridden = true
		po.Version = expectedVersion + 1
		po.UpdatedAt = now
		result = quotaRecordFromPO(po)
		changed = true
		return nil
	})
	return result, changed, err
}

func (r *repository) ResetQuota(
	ctx context.Context,
	target domain.UsageTarget,
	metric string,
	expectedVersion uint64,
) (*usageQuotaRecord, bool, error) {
	if !target.IsValid() || metric == "" {
		return nil, false, domain.ErrInvalidInput
	}
	var result *usageQuotaRecord
	changed := false
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		if err := lockUsageOwner(tx, target); err != nil {
			return err
		}
		po, err := findUsageQuotaForUpdate(tx, target, metric)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedVersion != 0 {
				return domain.ErrUsageQuotaVersionConflict
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("find usage quota override: %w", err)
		}
		if po.Version != expectedVersion {
			return domain.ErrUsageQuotaVersionConflict
		}
		result = quotaRecordFromPO(po)
		if !po.IsOverridden {
			return nil
		}
		now := r.now().UTC()
		update := tx.Model(&UsageQuotaPO{}).
			Where("id = ? AND version = ?", po.ID, expectedVersion).
			Updates(map[string]any{
				"limit_value":   nil,
				"is_overridden": false,
				"version":       expectedVersion + 1,
				"updated_at":    now,
			})
		if update.Error != nil {
			return fmt.Errorf("reset usage quota override: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrUsageQuotaVersionConflict
		}
		result = &usageQuotaRecord{
			Version:      expectedVersion + 1,
			UpdatedAt:    now,
			LimitValue:   nil,
			IsOverridden: false,
		}
		changed = true
		return nil
	})
	return result, changed, err
}

func (r *repository) PruneReceipts(ctx context.Context, before time.Time) (int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return 0, err
	}
	if before.IsZero() {
		return 0, domain.ErrInvalidInput
	}
	result := db.
		Where("created_at < ? AND decision <> ?", before.UTC(), usageDecisionPending).
		Delete(&UsageEventPO{})
	if result.Error != nil {
		return 0, fmt.Errorf("prune usage receipts: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *repository) DeleteForUser(ctx context.Context, userID uint) error {
	if userID == 0 {
		return domain.ErrInvalidInput
	}
	return r.inTransaction(ctx, func(tx *gorm.DB) error {
		for _, model := range []any{&UsageEventPO{}, &UsageCounterPO{}, &UsageQuotaPO{}} {
			if err := tx.Where("scope = ? AND user_id = ?", domain.UsageScopeUser, userID).
				Delete(model).Error; err != nil {
				return fmt.Errorf("delete user usage data: %w", err)
			}
		}
		return nil
	})
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

func validateUsageMutation(mutation usageMutation) error {
	if mutation.Operation != usageOperationRecord && mutation.Operation != usageOperationConsume {
		return domain.ErrUsageInvalidEvent
	}
	if !validUsageSource(mutation.Source) || !validUsageEventID(mutation.EventID) ||
		!mutation.Target.IsValid() || mutation.Metric == "" || mutation.Fingerprint == "" ||
		mutation.Quantity == 0 || mutation.Quantity < -maxSafeUsageInteger || mutation.Quantity > maxSafeUsageInteger ||
		mutation.OccurredAt.IsZero() || mutation.PeriodStart.IsZero() || !mutation.PeriodEnd.After(mutation.PeriodStart) {
		return domain.ErrUsageInvalidEvent
	}
	if mutation.Operation == usageOperationConsume && mutation.Quantity < 0 {
		return domain.ErrUsageInvalidEvent
	}
	return nil
}

func findUsageEvent(tx *gorm.DB, source string, eventID string) (*UsageEventPO, error) {
	var po UsageEventPO
	if err := tx.Where("source = ? AND event_id = ?", source, eventID).First(&po).Error; err != nil {
		return nil, fmt.Errorf("find usage event receipt: %w", err)
	}
	return &po, nil
}

func findUsageCounterForUpdate(
	tx *gorm.DB,
	target domain.UsageTarget,
	metric string,
	periodStart time.Time,
) (*UsageCounterPO, error) {
	query := lockQuery(tx)
	var po UsageCounterPO
	if err := query.Where(
		"scope = ? AND subject_id = ? AND metric = ? AND period_start = ?",
		target.Scope,
		target.SubjectID,
		metric,
		periodStart,
	).First(&po).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

func findUsageQuotaForUpdate(
	tx *gorm.DB,
	target domain.UsageTarget,
	metric string,
) (*UsageQuotaPO, error) {
	query := lockQuery(tx)
	var po UsageQuotaPO
	if err := query.Where(
		"scope = ? AND subject_id = ? AND metric = ?",
		target.Scope,
		target.SubjectID,
		metric,
	).First(&po).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

func lockUsageOwner(tx *gorm.DB, target domain.UsageTarget) error {
	query := ownerLockQuery(tx)
	switch target.Scope {
	case domain.UsageScopeUser:
		var owner user.UserPO
		if err := query.First(&owner, target.SubjectID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("lock usage user: %w", err)
		}
		return nil
	case domain.UsageScopeOrganization:
		var owner organization.OrganizationPO
		if err := query.First(&owner, target.SubjectID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrOrganizationNotFound
			}
			return fmt.Errorf("lock usage organization: %w", err)
		}
		return nil
	default:
		return domain.ErrInvalidInput
	}
}

func ownerLockQuery(tx *gorm.DB) *gorm.DB {
	if tx.Dialector != nil && tx.Name() != "sqlite" {
		// Receipt foreign-key checks hold KEY SHARE before owner serialization.
		// NO KEY UPDATE blocks owner mutation/deletion without a lock-upgrade deadlock.
		return tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"})
	}
	return tx
}

func lockQuery(tx *gorm.DB) *gorm.DB {
	if tx.Dialector != nil && tx.Name() != "sqlite" {
		return tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return tx
}

func newUsageEventPO(mutation usageMutation) *UsageEventPO {
	po := &UsageEventPO{
		Source:         mutation.Source,
		EventID:        mutation.EventID,
		Fingerprint:    mutation.Fingerprint,
		Operation:      mutation.Operation,
		Scope:          string(mutation.Target.Scope),
		SubjectID:      mutation.Target.SubjectID,
		Metric:         mutation.Metric,
		Quantity:       mutation.Quantity,
		DimensionsJSON: mutation.DimensionsJSON,
		OccurredAt:     mutation.OccurredAt,
		PeriodStart:    mutation.PeriodStart,
		PeriodEnd:      mutation.PeriodEnd,
		Decision:       usageDecisionPending,
	}
	setUsageOwner(&po.UserID, &po.OrganizationID, mutation.Target)
	return po
}

func newUsageCounterPO(mutation usageMutation, value int64) *UsageCounterPO {
	po := &UsageCounterPO{
		Scope:       string(mutation.Target.Scope),
		SubjectID:   mutation.Target.SubjectID,
		Metric:      mutation.Metric,
		PeriodStart: mutation.PeriodStart,
		PeriodEnd:   mutation.PeriodEnd,
		Value:       value,
		Version:     1,
	}
	setUsageOwner(&po.UserID, &po.OrganizationID, mutation.Target)
	return po
}

func newUsageQuotaPO(
	target domain.UsageTarget,
	metric string,
	limit *int64,
	isOverridden bool,
	version uint64,
) *UsageQuotaPO {
	po := &UsageQuotaPO{
		Scope:        string(target.Scope),
		SubjectID:    target.SubjectID,
		Metric:       metric,
		LimitValue:   cloneInt64(limit),
		IsOverridden: isOverridden,
		Version:      version,
	}
	setUsageOwner(&po.UserID, &po.OrganizationID, target)
	return po
}

func setUsageOwner(userID **uint, organizationID **uint, target domain.UsageTarget) {
	switch target.Scope {
	case domain.UsageScopeUser:
		id := target.SubjectID
		*userID = &id
	case domain.UsageScopeOrganization:
		id := target.SubjectID
		*organizationID = &id
	}
}

func quotaRecordFromPO(po *UsageQuotaPO) *usageQuotaRecord {
	if po == nil {
		return nil
	}
	return &usageQuotaRecord{
		LimitValue:   cloneInt64(po.LimitValue),
		IsOverridden: po.IsOverridden,
		Version:      po.Version,
		UpdatedAt:    po.UpdatedAt,
	}
}

func receiptFromPO(po *UsageEventPO, replayed bool) (*domain.UsageReceipt, error) {
	if po == nil {
		return nil, domain.ErrServiceUnavailable
	}
	decision := domain.UsageDecision(po.Decision)
	if decision != domain.UsageDecisionAccepted && decision != domain.UsageDecisionDenied {
		return nil, domain.ErrServiceUnavailable
	}
	target := domain.UsageTarget{Scope: domain.UsageScope(po.Scope), SubjectID: po.SubjectID}
	if !target.IsValid() {
		return nil, domain.ErrServiceUnavailable
	}
	return &domain.UsageReceipt{
		Source:        po.Source,
		EventID:       po.EventID,
		Target:        target,
		Metric:        po.Metric,
		Quantity:      po.Quantity,
		OccurredAt:    po.OccurredAt,
		PeriodStart:   po.PeriodStart,
		PeriodEnd:     po.PeriodEnd,
		Decision:      decision,
		CounterBefore: po.CounterBefore,
		CounterAfter:  po.CounterAfter,
		Limit:         cloneInt64(po.LimitValue),
		Replayed:      replayed,
	}, nil
}

func addSafeUsageQuantity(current int64, quantity int64) (int64, bool) {
	if current < 0 || current > maxSafeUsageInteger || quantity == 0 ||
		quantity < -maxSafeUsageInteger || quantity > maxSafeUsageInteger {
		return 0, false
	}
	if quantity > 0 && current > maxSafeUsageInteger-quantity {
		return 0, false
	}
	if quantity < 0 && current < -quantity {
		return 0, false
	}
	return current + quantity, true
}

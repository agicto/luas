package usage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

const (
	usageMaxEventAge      = 24 * time.Hour
	usageMaxFutureSkew    = 5 * time.Minute
	usageReceiptRetention = 90 * 24 * time.Hour
)

// Service owns metric validation, period resolution, idempotent accounting, and quota decisions.
type Service interface {
	domain.UsageReader
	domain.UsageRecorder
	domain.UsageConsumer
	domain.UsageQuotaWriter
	domain.UsageMaintainer
}

type service struct {
	catalog *Catalog
	store   usageStore
	enabled bool
	now     func() time.Time
}

var (
	_ Service                 = (*service)(nil)
	_ domain.UsageReader      = (*service)(nil)
	_ domain.UsageRecorder    = (*service)(nil)
	_ domain.UsageConsumer    = (*service)(nil)
	_ domain.UsageQuotaWriter = (*service)(nil)
	_ domain.UsageMaintainer  = (*service)(nil)
)

// NewService creates the optional usage service from one immutable catalog snapshot.
func NewService(catalog *Catalog, store usageStore, cfg *config.Config) *service {
	value := &service{catalog: catalog, store: store, now: time.Now}
	if cfg != nil {
		value.enabled = slices.Contains(cfg.Starters.Optional, "usage")
	}
	return value
}

func (s *service) ListUsage(
	ctx context.Context,
	target domain.UsageTarget,
) ([]*domain.UsageSummary, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if !target.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	definitions := s.catalog.Definitions(target.Scope)
	if len(definitions) == 0 {
		return []*domain.UsageSummary{}, nil
	}
	keys := make([]string, len(definitions))
	for index := range definitions {
		keys[index] = definitions[index].Key
	}
	now := s.now().UTC()
	counters, quotas, err := s.store.ListCurrent(ctx, target, now, keys)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.UsageSummary, 0, len(definitions))
	for _, definition := range definitions {
		start, end := usagePeriodWindow(definition.Period, now)
		counter := counters[definition.Key]
		quota, err := effectiveUsageQuota(target, definition, quotas[definition.Key])
		if err != nil {
			return nil, err
		}
		summary, err := buildUsageSummary(target, definition, quota, counter, start, end)
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
	}
	return result, nil
}

func (s *service) RecordUsage(
	ctx context.Context,
	event domain.UsageEvent,
) (*domain.UsageReceipt, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	definition, dimensionsJSON, occurredAt, err := s.normalizeRecord(event)
	if err != nil {
		return nil, err
	}
	start, end := usagePeriodWindow(definition.Period, occurredAt)
	fingerprint, err := usageEventFingerprint(
		usageOperationRecord,
		event.Target,
		event.Metric,
		event.Quantity,
		dimensionsJSON,
		occurredAt,
	)
	if err != nil {
		return nil, err
	}
	return s.store.Apply(ctx, usageMutation{
		Operation:      usageOperationRecord,
		Source:         event.Source,
		EventID:        event.EventID,
		Fingerprint:    fingerprint,
		Target:         event.Target,
		Metric:         event.Metric,
		Quantity:       event.Quantity,
		DimensionsJSON: dimensionsJSON,
		OccurredAt:     occurredAt,
		PeriodStart:    start,
		PeriodEnd:      end,
		DefaultLimit:   definition.DefaultLimit,
	})
}

func (s *service) ConsumeUsage(
	ctx context.Context,
	consumption domain.UsageConsumption,
) (*domain.UsageReceipt, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	definition, dimensionsJSON, err := s.normalizeConsumption(consumption)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	start, end := usagePeriodWindow(definition.Period, now)
	fingerprint, err := usageEventFingerprint(
		usageOperationConsume,
		consumption.Target,
		consumption.Metric,
		consumption.Quantity,
		dimensionsJSON,
		time.Time{},
	)
	if err != nil {
		return nil, err
	}
	receipt, err := s.store.Apply(ctx, usageMutation{
		Operation:      usageOperationConsume,
		Source:         consumption.Source,
		EventID:        consumption.EventID,
		Fingerprint:    fingerprint,
		Target:         consumption.Target,
		Metric:         consumption.Metric,
		Quantity:       consumption.Quantity,
		DimensionsJSON: dimensionsJSON,
		OccurredAt:     now,
		PeriodStart:    start,
		PeriodEnd:      end,
		DefaultLimit:   definition.DefaultLimit,
	})
	if err != nil {
		return nil, err
	}
	if receipt.Decision == domain.UsageDecisionDenied {
		return receipt, domain.ErrUsageQuotaExceeded
	}
	return receipt, nil
}

func (s *service) SetUsageQuota(
	ctx context.Context,
	target domain.UsageTarget,
	metric string,
	limit int64,
	expectedVersion uint64,
) (*domain.UsageQuota, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if !target.IsValid() || limit < 0 || limit > maxSafeUsageInteger {
		return nil, domain.ErrUsageInvalidEvent
	}
	definition, ok := s.catalog.Definition(target.Scope, metric)
	if !ok {
		return nil, domain.ErrUsageMetricNotFound
	}
	record, changed, err := s.store.SetQuota(ctx, target, metric, limit, expectedVersion)
	if err != nil {
		return nil, err
	}
	quota, err := effectiveUsageQuota(target, definition, record)
	if err != nil {
		return nil, err
	}
	if changed {
		recordUsageQuotaAudit(ctx, "set", quota, expectedVersion)
	}
	return quota, nil
}

func (s *service) ResetUsageQuota(
	ctx context.Context,
	target domain.UsageTarget,
	metric string,
	expectedVersion uint64,
) (*domain.UsageQuota, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if !target.IsValid() {
		return nil, domain.ErrUsageInvalidEvent
	}
	definition, ok := s.catalog.Definition(target.Scope, metric)
	if !ok {
		return nil, domain.ErrUsageMetricNotFound
	}
	record, changed, err := s.store.ResetQuota(ctx, target, metric, expectedVersion)
	if err != nil {
		return nil, err
	}
	quota, err := effectiveUsageQuota(target, definition, record)
	if err != nil {
		return nil, err
	}
	if changed {
		recordUsageQuotaAudit(ctx, "reset", quota, expectedVersion)
	}
	return quota, nil
}

func (s *service) PruneUsageReceipts(ctx context.Context, before time.Time) (int64, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	if before.IsZero() {
		return 0, domain.ErrUsageInvalidEvent
	}
	cutoff := before.UTC()
	if cutoff.After(s.now().UTC().Add(-usageReceiptRetention)) {
		return 0, domain.ErrUsageInvalidEvent
	}
	return s.store.PruneReceipts(ctx, cutoff)
}

func (s *service) AccountDeletionCleanerName() string { return "usage" }

func (s *service) CleanAccountDeletion(ctx context.Context, userID uint) error {
	if err := s.available(); err != nil {
		return err
	}
	return s.store.DeleteForUser(ctx, userID)
}

func (s *service) available() error {
	if s == nil || !s.enabled || s.catalog == nil || s.store == nil || s.now == nil {
		return domain.ErrServiceUnavailable
	}
	return nil
}

func (s *service) normalizeRecord(
	event domain.UsageEvent,
) (Definition, string, time.Time, error) {
	if !event.Target.IsValid() || !validUsageSource(event.Source) || !validUsageEventID(event.EventID) ||
		event.Quantity == 0 || event.Quantity < -maxSafeUsageInteger || event.Quantity > maxSafeUsageInteger ||
		event.OccurredAt.IsZero() {
		return Definition{}, "", time.Time{}, domain.ErrUsageInvalidEvent
	}
	definition, ok := s.catalog.Definition(event.Target.Scope, event.Metric)
	if !ok {
		return Definition{}, "", time.Time{}, domain.ErrUsageMetricNotFound
	}
	_, encoded, err := normalizeUsageDimensions(definition, event.Dimensions)
	if err != nil {
		return Definition{}, "", time.Time{}, err
	}
	now := s.now().UTC()
	occurredAt := event.OccurredAt.UTC()
	if occurredAt.Before(now.Add(-usageMaxEventAge)) || occurredAt.After(now.Add(usageMaxFutureSkew)) {
		return Definition{}, "", time.Time{}, domain.ErrUsageEventOutsideWindow
	}
	return definition, encoded, occurredAt, nil
}

func (s *service) normalizeConsumption(
	consumption domain.UsageConsumption,
) (Definition, string, error) {
	if !consumption.Target.IsValid() || !validUsageSource(consumption.Source) ||
		!validUsageEventID(consumption.EventID) || consumption.Quantity <= 0 ||
		consumption.Quantity > maxSafeUsageInteger {
		return Definition{}, "", domain.ErrUsageInvalidEvent
	}
	definition, ok := s.catalog.Definition(consumption.Target.Scope, consumption.Metric)
	if !ok {
		return Definition{}, "", domain.ErrUsageMetricNotFound
	}
	_, encoded, err := normalizeUsageDimensions(definition, consumption.Dimensions)
	if err != nil {
		return Definition{}, "", err
	}
	return definition, encoded, nil
}

func usagePeriodWindow(period domain.UsagePeriod, at time.Time) (time.Time, time.Time) {
	value := at.UTC()
	switch period {
	case domain.UsagePeriodDay:
		start := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 1)
	case domain.UsagePeriodMonth:
		start := time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	default:
		return time.Time{}, time.Time{}
	}
}

func effectiveUsageQuota(
	target domain.UsageTarget,
	definition Definition,
	record *usageQuotaRecord,
) (*domain.UsageQuota, error) {
	quota := &domain.UsageQuota{
		Target: target,
		Metric: definition.Key,
		Limit:  cloneInt64(definition.DefaultLimit),
		Source: domain.UsageQuotaSourceDefault,
	}
	if record == nil {
		return quota, nil
	}
	if record.Version == 0 || record.UpdatedAt.IsZero() ||
		record.IsOverridden != (record.LimitValue != nil) ||
		(record.LimitValue != nil && (*record.LimitValue < 0 || *record.LimitValue > maxSafeUsageInteger)) {
		return nil, domain.ErrServiceUnavailable
	}
	quota.Version = record.Version
	updatedAt := record.UpdatedAt
	quota.UpdatedAt = &updatedAt
	if record.IsOverridden {
		quota.Limit = cloneInt64(record.LimitValue)
		quota.Source = domain.UsageQuotaSourceOverride
	}
	return quota, nil
}

func buildUsageSummary(
	target domain.UsageTarget,
	definition Definition,
	quota *domain.UsageQuota,
	counter *usageCounterRecord,
	periodStart time.Time,
	periodEnd time.Time,
) (*domain.UsageSummary, error) {
	if quota == nil || periodStart.IsZero() || !periodEnd.After(periodStart) {
		return nil, domain.ErrServiceUnavailable
	}
	used := int64(0)
	var updatedAt *time.Time
	if counter != nil {
		if counter.Value < 0 || counter.Value > maxSafeUsageInteger ||
			counter.Version == 0 || counter.UpdatedAt.IsZero() {
			return nil, domain.ErrServiceUnavailable
		}
		used = counter.Value
		updated := counter.UpdatedAt
		updatedAt = &updated
	}
	summary := &domain.UsageSummary{
		Target:       target,
		Metric:       definition.Key,
		Unit:         definition.Unit,
		Period:       definition.Period,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		Used:         used,
		Limit:        cloneInt64(quota.Limit),
		QuotaSource:  quota.Source,
		QuotaVersion: quota.Version,
		UpdatedAt:    updatedAt,
	}
	if quota.Limit == nil {
		return summary, nil
	}
	remaining := *quota.Limit - used
	if remaining < 0 {
		summary.OverLimit = true
		summary.Overage = -remaining
		remaining = 0
	}
	summary.Remaining = &remaining
	return summary, nil
}

func usageEventFingerprint(
	operation string,
	target domain.UsageTarget,
	metric string,
	quantity int64,
	dimensionsJSON string,
	occurredAt time.Time,
) (string, error) {
	payload := struct {
		Version        int               `json:"version"`
		Operation      string            `json:"operation"`
		Scope          domain.UsageScope `json:"scope"`
		SubjectID      uint              `json:"subject_id"`
		Metric         string            `json:"metric"`
		Quantity       int64             `json:"quantity"`
		DimensionsJSON string            `json:"dimensions_json"`
		OccurredAt     string            `json:"occurred_at,omitempty"`
	}{
		Version:        1,
		Operation:      operation,
		Scope:          target.Scope,
		SubjectID:      target.SubjectID,
		Metric:         metric,
		Quantity:       quantity,
		DimensionsJSON: dimensionsJSON,
	}
	if !occurredAt.IsZero() {
		payload.OccurredAt = occurredAt.UTC().Format(time.RFC3339Nano)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode usage event fingerprint: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func recordUsageQuotaAudit(
	ctx context.Context,
	operation string,
	quota *domain.UsageQuota,
	beforeVersion uint64,
) {
	if quota == nil {
		return
	}
	metadata := map[string]any{
		"operation":      operation,
		"scope":          quota.Target.Scope,
		"metric":         quota.Metric,
		"before_version": beforeVersion,
		"after_version":  quota.Version,
		"source":         quota.Source,
	}
	if quota.Target.Scope == domain.UsageScopeUser {
		metadata["user_id"] = quota.Target.SubjectID
	} else {
		metadata["organization_id"] = quota.Target.SubjectID
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     operation,
		Resource:   "usage_quotas",
		TargetType: "usage_quota",
		TargetID: string(quota.Target.Scope) + ":" +
			strconv.FormatUint(uint64(quota.Target.SubjectID), 10) + ":" + quota.Metric,
		Result:   domain.AuditResultSuccess,
		Metadata: metadata,
	})
}

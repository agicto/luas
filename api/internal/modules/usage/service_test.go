package usage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestUsageRecordIsIdempotentAndSupportsBoundedCorrections(t *testing.T) {
	fixture := newUsageTestFixture(t)
	target := domain.UsageTarget{Scope: domain.UsageScopeUser, SubjectID: fixture.user.ID}
	event := domain.UsageEvent{
		Source:     "api.gateway",
		EventID:    "request-001",
		Target:     target,
		Metric:     "api.requests",
		Quantity:   10,
		OccurredAt: fixture.now.Add(-time.Hour),
	}

	first, err := fixture.service.RecordUsage(context.Background(), event)
	require.NoError(t, err)
	assert.False(t, first.Replayed)
	assert.Equal(t, int64(10), first.CounterAfter)

	replayed, err := fixture.service.RecordUsage(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, replayed.Replayed)
	assert.Equal(t, first.CounterAfter, replayed.CounterAfter)

	conflict := event
	conflict.Quantity = 11
	_, err = fixture.service.RecordUsage(context.Background(), conflict)
	assert.ErrorIs(t, err, domain.ErrUsageIdempotencyConflict)

	correction := event
	correction.EventID = "request-001-correction"
	correction.Quantity = -3
	corrected, err := fixture.service.RecordUsage(context.Background(), correction)
	require.NoError(t, err)
	assert.Equal(t, int64(7), corrected.CounterAfter)

	invalid := correction
	invalid.EventID = "request-001-invalid-correction"
	invalid.Quantity = -8
	_, err = fixture.service.RecordUsage(context.Background(), invalid)
	assert.ErrorIs(t, err, domain.ErrUsageInvalidEvent)

	summary := usageSummaryByMetric(t, fixture.service, target, "api.requests")
	assert.Equal(t, int64(7), summary.Used)
	assert.Nil(t, summary.Limit)
	assert.Nil(t, summary.Remaining)
	assert.Equal(t, domain.UsageQuotaSourceDefault, summary.QuotaSource)
	assert.Equal(t, uint64(0), summary.QuotaVersion)

	var receipts int64
	require.NoError(t, fixture.db.Model(&UsageEventPO{}).Count(&receipts).Error)
	assert.Equal(t, int64(2), receipts)
}

func TestUsageRecordEnforcesOccurrenceWindowAndSafeIntegers(t *testing.T) {
	fixture := newUsageTestFixture(t)
	target := domain.UsageTarget{Scope: domain.UsageScopeUser, SubjectID: fixture.user.ID}
	base := domain.UsageEvent{
		Source:     "worker.jobs",
		EventID:    "old-event",
		Target:     target,
		Metric:     "workflow.runs",
		Quantity:   1,
		OccurredAt: fixture.now.Add(-usageMaxEventAge - time.Nanosecond),
	}
	_, err := fixture.service.RecordUsage(context.Background(), base)
	assert.ErrorIs(t, err, domain.ErrUsageEventOutsideWindow)

	base.EventID = "future-event"
	base.OccurredAt = fixture.now.Add(usageMaxFutureSkew + time.Nanosecond)
	_, err = fixture.service.RecordUsage(context.Background(), base)
	assert.ErrorIs(t, err, domain.ErrUsageEventOutsideWindow)

	base.EventID = "unsafe-event"
	base.OccurredAt = fixture.now
	base.Quantity = maxSafeUsageInteger + 1
	_, err = fixture.service.RecordUsage(context.Background(), base)
	assert.ErrorIs(t, err, domain.ErrUsageInvalidEvent)
}

func TestUsageConsumeAtomicallyPersistsAcceptedAndDeniedDecisions(t *testing.T) {
	fixture := newUsageTestFixture(t)
	target := domain.UsageTarget{Scope: domain.UsageScopeOrganization, SubjectID: fixture.organization.ID}
	quota, err := fixture.service.SetUsageQuota(context.Background(), target, "ai.input_tokens", 5, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), quota.Version)
	assert.Equal(t, domain.UsageQuotaSourceOverride, quota.Source)

	accepted := domain.UsageConsumption{
		Source:   "ai.runtime",
		EventID:  "completion-001",
		Target:   target,
		Metric:   "ai.input_tokens",
		Quantity: 3,
	}
	receipt, err := fixture.service.ConsumeUsage(context.Background(), accepted)
	require.NoError(t, err)
	assert.Equal(t, domain.UsageDecisionAccepted, receipt.Decision)
	assert.Equal(t, int64(3), receipt.CounterAfter)

	replayed, err := fixture.service.ConsumeUsage(context.Background(), accepted)
	require.NoError(t, err)
	assert.True(t, replayed.Replayed)

	denied := accepted
	denied.EventID = "completion-002"
	denied.Quantity = 3
	deniedReceipt, err := fixture.service.ConsumeUsage(context.Background(), denied)
	assert.ErrorIs(t, err, domain.ErrUsageQuotaExceeded)
	require.NotNil(t, deniedReceipt)
	assert.Equal(t, domain.UsageDecisionDenied, deniedReceipt.Decision)
	assert.Equal(t, int64(3), deniedReceipt.CounterAfter)

	replayedDenied, err := fixture.service.ConsumeUsage(context.Background(), denied)
	assert.ErrorIs(t, err, domain.ErrUsageQuotaExceeded)
	require.NotNil(t, replayedDenied)
	assert.True(t, replayedDenied.Replayed)

	summary := usageSummaryByMetric(t, fixture.service, target, "ai.input_tokens")
	assert.Equal(t, int64(3), summary.Used)
	require.NotNil(t, summary.Remaining)
	assert.Equal(t, int64(2), *summary.Remaining)
}

func TestUsageQuotaCASRetainsResetHistoryAndRecordCanExposeOverage(t *testing.T) {
	fixture := newUsageTestFixture(t)
	target := domain.UsageTarget{Scope: domain.UsageScopeUser, SubjectID: fixture.user.ID}

	reset, err := fixture.service.ResetUsageQuota(context.Background(), target, "api.requests", 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), reset.Version)
	assert.Nil(t, reset.Limit)

	quota, err := fixture.service.SetUsageQuota(context.Background(), target, "api.requests", 5, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), quota.Version)
	unchanged, err := fixture.service.SetUsageQuota(context.Background(), target, "api.requests", 5, 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), unchanged.Version)
	_, err = fixture.service.SetUsageQuota(context.Background(), target, "api.requests", 6, 0)
	assert.ErrorIs(t, err, domain.ErrUsageQuotaVersionConflict)

	_, err = fixture.service.RecordUsage(context.Background(), domain.UsageEvent{
		Source: "api.gateway", EventID: "overage", Target: target,
		Metric: "api.requests", Quantity: 8, OccurredAt: fixture.now,
	})
	require.NoError(t, err)
	summary := usageSummaryByMetric(t, fixture.service, target, "api.requests")
	assert.True(t, summary.OverLimit)
	assert.Equal(t, int64(3), summary.Overage)
	require.NotNil(t, summary.Remaining)
	assert.Zero(t, *summary.Remaining)

	reset, err = fixture.service.ResetUsageQuota(context.Background(), target, "api.requests", 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), reset.Version)
	assert.Equal(t, domain.UsageQuotaSourceDefault, reset.Source)
}

func TestUsageFailsClosedForCorruptQuotaAndCounterState(t *testing.T) {
	target := domain.UsageTarget{Scope: domain.UsageScopeUser, SubjectID: 1}
	definition := NewDefinition(target.Scope, "api.requests", "request", domain.UsagePeriodMonth, nil, nil)
	now := time.Now().UTC()

	_, err := effectiveUsageQuota(target, definition, &usageQuotaRecord{
		IsOverridden: true,
		Version:      1,
		UpdatedAt:    now,
	})
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)

	quota, err := effectiveUsageQuota(target, definition, nil)
	require.NoError(t, err)
	_, err = buildUsageSummary(target, definition, quota, &usageCounterRecord{
		Value:     1,
		Version:   0,
		UpdatedAt: now,
	}, now, now.AddDate(0, 1, 0))
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

func TestUsageRetentionAndAccountDeletionPreserveOwnedBoundaries(t *testing.T) {
	fixture := newUsageTestFixture(t)
	target := domain.UsageTarget{Scope: domain.UsageScopeUser, SubjectID: fixture.user.ID}
	_, err := fixture.service.RecordUsage(context.Background(), domain.UsageEvent{
		Source: "api.gateway", EventID: "delete-me", Target: target,
		Metric: "api.requests", Quantity: 1, OccurredAt: fixture.now,
	})
	require.NoError(t, err)
	require.NoError(t, fixture.db.Model(&UsageEventPO{}).
		Where("event_id = ?", "delete-me").
		Update("created_at", fixture.now.Add(-100*24*time.Hour)).Error)

	_, err = fixture.service.PruneUsageReceipts(context.Background(), fixture.now.Add(-89*24*time.Hour))
	assert.ErrorIs(t, err, domain.ErrUsageInvalidEvent)
	count, err := fixture.service.PruneUsageReceipts(context.Background(), fixture.now.Add(-90*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	_, err = fixture.service.RecordUsage(context.Background(), domain.UsageEvent{
		Source: "api.gateway", EventID: "account-delete", Target: target,
		Metric: "api.requests", Quantity: 1, OccurredAt: fixture.now,
	})
	require.NoError(t, err)
	policy := user.NewAccountDeletionPolicy()
	require.NoError(t, policy.RegisterCleaner(fixture.service))
	err = user.NewRepository(fixture.db).DeleteAccount(context.Background(), fixture.user.ID, func(ctx context.Context) error {
		return policy.Prepare(ctx, fixture.user.ID)
	})
	require.NoError(t, err)

	for _, model := range []any{&UsageEventPO{}, &UsageCounterPO{}, &UsageQuotaPO{}} {
		var rows int64
		require.NoError(t, fixture.db.Model(model).Count(&rows).Error)
		assert.Zero(t, rows)
	}
}

func usageSummaryByMetric(
	t *testing.T,
	service *service,
	target domain.UsageTarget,
	metric string,
) *domain.UsageSummary {
	t.Helper()
	values, err := service.ListUsage(context.Background(), target)
	require.NoError(t, err)
	for _, value := range values {
		if value.Metric == metric {
			return value
		}
	}
	require.FailNow(t, "usage metric missing", metric)
	return nil
}

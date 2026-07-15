package domain

import (
	"context"
	"time"
)

// UsageScope identifies a durable business usage owner.
type UsageScope string

const (
	UsageScopeUser         UsageScope = "user"
	UsageScopeOrganization UsageScope = "organization"
)

// IsValid reports whether the scope belongs to the stable usage vocabulary.
func (s UsageScope) IsValid() bool {
	return s == UsageScopeUser || s == UsageScopeOrganization
}

// UsagePeriod identifies a UTC calendar aggregation window.
type UsagePeriod string

const (
	UsagePeriodDay   UsagePeriod = "day"
	UsagePeriodMonth UsagePeriod = "month"
)

// IsValid reports whether the period belongs to the stable usage vocabulary.
func (p UsagePeriod) IsValid() bool {
	return p == UsagePeriodDay || p == UsagePeriodMonth
}

// UsageTarget identifies one user or organization usage subject.
type UsageTarget struct {
	Scope     UsageScope
	SubjectID uint
}

// IsValid enforces a real owner identity.
func (t UsageTarget) IsValid() bool {
	return t.Scope.IsValid() && t.SubjectID > 0
}

// UsageEvent is one trusted additive fact or correction with caller-owned idempotency.
type UsageEvent struct {
	Source     string
	EventID    string
	Target     UsageTarget
	Metric     string
	Quantity   int64
	Dimensions map[string]string
	OccurredAt time.Time
}

// UsageConsumption requests one server-timed atomic quota decision.
type UsageConsumption struct {
	Source     string
	EventID    string
	Target     UsageTarget
	Metric     string
	Quantity   int64
	Dimensions map[string]string
}

// UsageDecision records whether an event affected its period counter.
type UsageDecision string

const (
	UsageDecisionAccepted UsageDecision = "accepted"
	UsageDecisionDenied   UsageDecision = "denied"
)

// UsageReceipt is the immutable result of recording or consuming one event.
type UsageReceipt struct {
	Source        string
	EventID       string
	Target        UsageTarget
	Metric        string
	Quantity      int64
	OccurredAt    time.Time
	PeriodStart   time.Time
	PeriodEnd     time.Time
	Decision      UsageDecision
	CounterBefore int64
	CounterAfter  int64
	Limit         *int64
	Replayed      bool
}

// UsageQuotaSource says whether a hard limit comes from code or a durable subject override.
type UsageQuotaSource string

const (
	UsageQuotaSourceDefault  UsageQuotaSource = "default"
	UsageQuotaSourceOverride UsageQuotaSource = "override"
)

// UsageQuota is one effective hard limit and its durable override version.
type UsageQuota struct {
	Target    UsageTarget
	Metric    string
	Limit     *int64
	Source    UsageQuotaSource
	Version   uint64
	UpdatedAt *time.Time
}

// UsageSummary is the current effective counter and quota for one metric.
type UsageSummary struct {
	Target       UsageTarget
	Metric       string
	Unit         string
	Period       UsagePeriod
	PeriodStart  time.Time
	PeriodEnd    time.Time
	Used         int64
	Limit        *int64
	Remaining    *int64
	Overage      int64
	OverLimit    bool
	QuotaSource  UsageQuotaSource
	QuotaVersion uint64
	UpdatedAt    *time.Time
}

// UsageReader exposes bounded current-period business usage.
type UsageReader interface {
	ListUsage(ctx context.Context, target UsageTarget) ([]*UsageSummary, error)
}

// UsageRecorder stores trusted occurred usage facts and corrections.
type UsageRecorder interface {
	RecordUsage(ctx context.Context, event UsageEvent) (*UsageReceipt, error)
}

// UsageConsumer performs an atomic server-timed hard-quota decision.
type UsageConsumer interface {
	ConsumeUsage(ctx context.Context, consumption UsageConsumption) (*UsageReceipt, error)
}

// UsageQuotaWriter owns monotonic subject hard-limit overrides.
type UsageQuotaWriter interface {
	SetUsageQuota(
		ctx context.Context,
		target UsageTarget,
		metric string,
		limit int64,
		expectedVersion uint64,
	) (*UsageQuota, error)
	ResetUsageQuota(
		ctx context.Context,
		target UsageTarget,
		metric string,
		expectedVersion uint64,
	) (*UsageQuota, error)
}

// UsageMaintainer owns bounded receipt retention without deleting counters or quotas.
type UsageMaintainer interface {
	PruneUsageReceipts(ctx context.Context, before time.Time) (int64, error)
}

package usage

import (
	"time"

	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

const (
	usageOperationRecord  = "record"
	usageOperationConsume = "consume"
	usageDecisionPending  = "pending"
)

// UsageEventPO stores one minimized idempotency receipt and atomic counter decision.
type UsageEventPO struct {
	ID             uint64    `gorm:"primaryKey"`
	Source         string    `gorm:"size:64;not null;uniqueIndex:idx_usage_events_source_event,priority:1"`
	EventID        string    `gorm:"size:128;not null;uniqueIndex:idx_usage_events_source_event,priority:2"`
	Fingerprint    string    `gorm:"size:64;not null;check:usage_events_fingerprint_check,length(fingerprint) = 64"`
	Operation      string    `gorm:"size:16;not null;check:usage_events_operation_check,operation IN ('record','consume')"`
	Scope          string    `gorm:"size:24;not null;index:idx_usage_events_subject,priority:1;check:usage_events_scope_check,scope IN ('user','organization')"`
	SubjectID      uint      `gorm:"not null;index:idx_usage_events_subject,priority:2;check:usage_events_subject_check,(scope = 'user' AND subject_id = user_id AND user_id IS NOT NULL AND organization_id IS NULL) OR (scope = 'organization' AND subject_id = organization_id AND organization_id IS NOT NULL AND user_id IS NULL)"`
	UserID         *uint     `gorm:"index:idx_usage_events_user"`
	OrganizationID *uint     `gorm:"index:idx_usage_events_organization"`
	Metric         string    `gorm:"size:96;not null;index:idx_usage_events_subject,priority:3"`
	Quantity       int64     `gorm:"not null;check:usage_events_quantity_check,quantity <> 0 AND quantity BETWEEN -9007199254740991 AND 9007199254740991 AND (operation = 'record' OR quantity > 0)"`
	DimensionsJSON string    `gorm:"type:text;not null;default:'{}';check:usage_events_dimensions_check,length(dimensions_json) BETWEEN 2 AND 4096"`
	OccurredAt     time.Time `gorm:"not null"`
	PeriodStart    time.Time `gorm:"not null;index:idx_usage_events_subject,priority:4"`
	PeriodEnd      time.Time `gorm:"not null;check:usage_events_period_check,period_end > period_start"`
	Decision       string    `gorm:"size:16;not null;check:usage_events_decision_check,decision IN ('pending','accepted','denied')"`
	CounterBefore  int64     `gorm:"not null;check:usage_events_counter_before_check,counter_before BETWEEN 0 AND 9007199254740991"`
	CounterAfter   int64     `gorm:"not null;check:usage_events_counter_after_check,counter_after BETWEEN 0 AND 9007199254740991 AND (decision <> 'denied' OR counter_after = counter_before)"`
	LimitValue     *int64    `gorm:"check:usage_events_limit_check,limit_value IS NULL OR limit_value BETWEEN 0 AND 9007199254740991"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	User         *user.UserPO                 `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Organization *organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (UsageEventPO) TableName() string { return "usage_events" }

// UsageCounterPO stores one non-negative scoped UTC period aggregate.
type UsageCounterPO struct {
	ID             uint64    `gorm:"primaryKey"`
	Scope          string    `gorm:"size:24;not null;uniqueIndex:idx_usage_counters_identity,priority:1;index:idx_usage_counters_current,priority:1;check:usage_counters_scope_check,scope IN ('user','organization')"`
	SubjectID      uint      `gorm:"not null;uniqueIndex:idx_usage_counters_identity,priority:2;index:idx_usage_counters_current,priority:2;check:usage_counters_subject_check,(scope = 'user' AND subject_id = user_id AND user_id IS NOT NULL AND organization_id IS NULL) OR (scope = 'organization' AND subject_id = organization_id AND organization_id IS NOT NULL AND user_id IS NULL)"`
	UserID         *uint     `gorm:"index:idx_usage_counters_user"`
	OrganizationID *uint     `gorm:"index:idx_usage_counters_organization"`
	Metric         string    `gorm:"size:96;not null;uniqueIndex:idx_usage_counters_identity,priority:3;index:idx_usage_counters_current,priority:3"`
	PeriodStart    time.Time `gorm:"not null;uniqueIndex:idx_usage_counters_identity,priority:4"`
	PeriodEnd      time.Time `gorm:"not null;index:idx_usage_counters_current,priority:4;check:usage_counters_period_check,period_end > period_start"`
	Value          int64     `gorm:"not null;check:usage_counters_value_check,value BETWEEN 0 AND 9007199254740991"`
	Version        uint64    `gorm:"not null;check:usage_counters_version_check,version > 0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	User         *user.UserPO                 `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Organization *organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (UsageCounterPO) TableName() string { return "usage_counters" }

// UsageQuotaPO stores one subject override or reset tombstone for a code-owned metric.
type UsageQuotaPO struct {
	ID             uint64 `gorm:"primaryKey"`
	Scope          string `gorm:"size:24;not null;uniqueIndex:idx_usage_quotas_identity,priority:1;check:usage_quotas_scope_check,scope IN ('user','organization')"`
	SubjectID      uint   `gorm:"not null;uniqueIndex:idx_usage_quotas_identity,priority:2;check:usage_quotas_subject_check,(scope = 'user' AND subject_id = user_id AND user_id IS NOT NULL AND organization_id IS NULL) OR (scope = 'organization' AND subject_id = organization_id AND organization_id IS NOT NULL AND user_id IS NULL)"`
	UserID         *uint  `gorm:"index:idx_usage_quotas_user"`
	OrganizationID *uint  `gorm:"index:idx_usage_quotas_organization"`
	Metric         string `gorm:"size:96;not null;uniqueIndex:idx_usage_quotas_identity,priority:3"`
	LimitValue     *int64 `gorm:"check:usage_quotas_limit_check,limit_value IS NULL OR limit_value BETWEEN 0 AND 9007199254740991"`
	IsOverridden   bool   `gorm:"not null;default:false;check:usage_quotas_state_check,(is_overridden = TRUE AND limit_value IS NOT NULL) OR (is_overridden = FALSE AND limit_value IS NULL)"`
	Version        uint64 `gorm:"not null;check:usage_quotas_version_check,version > 0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	User         *user.UserPO                 `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Organization *organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (UsageQuotaPO) TableName() string { return "usage_quotas" }

type usageCounterRecord struct {
	Value     int64
	Version   uint64
	UpdatedAt time.Time
}

type usageQuotaRecord struct {
	LimitValue   *int64
	IsOverridden bool
	Version      uint64
	UpdatedAt    time.Time
}

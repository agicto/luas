package usage

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

// UsageSummaryResponse is one bounded current-period browser summary.
type UsageSummaryResponse struct {
	Scope        domain.UsageScope       `json:"scope"`
	Metric       string                  `json:"metric"`
	Unit         string                  `json:"unit"`
	Period       domain.UsagePeriod      `json:"period"`
	PeriodStart  time.Time               `json:"period_start"`
	PeriodEnd    time.Time               `json:"period_end"`
	Used         int64                   `json:"used"`
	Limit        *int64                  `json:"limit"`
	Remaining    *int64                  `json:"remaining"`
	Overage      int64                   `json:"overage"`
	OverLimit    bool                    `json:"over_limit"`
	QuotaSource  domain.UsageQuotaSource `json:"quota_source"`
	QuotaVersion uint64                  `json:"quota_version"`
	UpdatedAt    *time.Time              `json:"updated_at"`
}

func toUsageSummaryResponse(summary *domain.UsageSummary) *UsageSummaryResponse {
	if summary == nil {
		return nil
	}
	return &UsageSummaryResponse{
		Scope:        summary.Target.Scope,
		Metric:       summary.Metric,
		Unit:         summary.Unit,
		Period:       summary.Period,
		PeriodStart:  summary.PeriodStart,
		PeriodEnd:    summary.PeriodEnd,
		Used:         summary.Used,
		Limit:        cloneInt64(summary.Limit),
		Remaining:    cloneInt64(summary.Remaining),
		Overage:      summary.Overage,
		OverLimit:    summary.OverLimit,
		QuotaSource:  summary.QuotaSource,
		QuotaVersion: summary.QuotaVersion,
		UpdatedAt:    summary.UpdatedAt,
	}
}

func toUsageSummaryResponses(values []*domain.UsageSummary) []*UsageSummaryResponse {
	result := make([]*UsageSummaryResponse, len(values))
	for index := range values {
		result[index] = toUsageSummaryResponse(values[index])
	}
	return result
}

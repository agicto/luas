package trend

import (
	"time"
)

type TrendListRequest struct {
	Status      string `form:"status" binding:"omitempty,oneof=new candidate selected rejected archived"`
	Channel     string `form:"channel" binding:"omitempty,max=80"`
	Search      string `form:"search" binding:"omitempty,max=160"`
	Recommended *bool  `form:"recommended"`
}

type TrendSourceResponse struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
}

type TrendScoreResponse struct {
	HScore        *int  `json:"h_score,omitempty"`
	KScore        *int  `json:"k_score,omitempty"`
	RScore        *int  `json:"r_score,omitempty"`
	BrandFitScore *int  `json:"brand_fit_score,omitempty"`
	RiskScore     *int  `json:"risk_score,omitempty"`
	TotalScore    *int  `json:"total_score,omitempty"`
	Recommended   *bool `json:"recommended,omitempty"`
}

type TrendResponse struct {
	ID               string              `json:"id"`
	Source           TrendSourceResponse `json:"source"`
	CanonicalURL     string              `json:"canonical_url"`
	Title            string              `json:"title"`
	Summary          string              `json:"summary,omitempty"`
	Channel          string              `json:"channel,omitempty"`
	Language         string              `json:"language"`
	HighlightedAt    *time.Time          `json:"highlighted_at,omitempty"`
	Significance     string              `json:"significance,omitempty"`
	Status           string              `json:"status"`
	Scores           TrendScoreResponse  `json:"scores"`
	TargetAudience   string              `json:"target_audience,omitempty"`
	RecommendedAngle string              `json:"recommended_angle,omitempty"`
	RiskNotes        string              `json:"risk_notes,omitempty"`
	EvaluatedAt      *time.Time          `json:"evaluated_at,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type TrendStatsResponse struct {
	Total             int64      `json:"total"`
	New               int64      `json:"new"`
	Candidate         int64      `json:"candidate"`
	Selected          int64      `json:"selected"`
	Rejected          int64      `json:"rejected"`
	QueuedJobs        int64      `json:"queued_jobs"`
	SucceededJobs     int64      `json:"succeeded_jobs"`
	LatestTrendAt     *time.Time `json:"latest_trend_at,omitempty"`
	LatestPolledAt    *time.Time `json:"latest_polled_at,omitempty"`
	LatestEvaluatedAt *time.Time `json:"latest_evaluated_at,omitempty"`
}

type SourceStatusResponse struct {
	ID                  string     `json:"id,omitempty"`
	Provider            string     `json:"provider,omitempty"`
	Name                string     `json:"name,omitempty"`
	SourceURL           string     `json:"source_url,omitempty"`
	PollIntervalMinutes int        `json:"poll_interval_minutes,omitempty"`
	Status              string     `json:"status,omitempty"`
	LastPolledAt        *time.Time `json:"last_polled_at,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
}

type TrendListSummaryResponse struct {
	Stats  *TrendStatsResponse   `json:"stats"`
	Source *SourceStatusResponse `json:"source,omitempty"`
}

type TrendSyncRunResponse struct {
	SourceID         string `json:"source_id"`
	Fetched          int    `json:"fetched"`
	Upserted         int    `json:"upserted"`
	Inserted         int    `json:"inserted"`
	Evaluated        int    `json:"evaluated"`
	Candidates       int    `json:"candidates"`
	EnqueuedScoreJob int    `json:"enqueued_score_job"`
}

func (r *TrendListRequest) toFilter(page, perPage int) TrendListFilter {
	if r == nil {
		return TrendListFilter{Page: page, PerPage: perPage}
	}

	return TrendListFilter{
		Status:      r.Status,
		Channel:     r.Channel,
		Search:      r.Search,
		Recommended: r.Recommended,
		Page:        page,
		PerPage:     perPage,
	}
}

func toTrendResponse(item TrendListItem) TrendResponse {
	return TrendResponse{
		ID: item.ID,
		Source: TrendSourceResponse{
			Provider: item.SourceProvider,
			Name:     item.SourceName,
		},
		CanonicalURL:  item.CanonicalURL,
		Title:         item.Title,
		Summary:       item.Summary,
		Channel:       item.Channel,
		Language:      item.Language,
		HighlightedAt: item.HighlightedAt,
		Significance:  item.Significance,
		Status:        item.Status,
		Scores: TrendScoreResponse{
			HScore:        item.HScore,
			KScore:        item.KScore,
			RScore:        item.RScore,
			BrandFitScore: item.BrandFitScore,
			RiskScore:     item.RiskScore,
			TotalScore:    item.TotalScore,
			Recommended:   item.Recommended,
		},
		TargetAudience:   item.TargetAudience,
		RecommendedAngle: item.RecommendedAngle,
		RiskNotes:        item.RiskNotes,
		EvaluatedAt:      item.EvaluatedAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func toTrendResponses(items []TrendListItem) []TrendResponse {
	result := make([]TrendResponse, len(items))
	for i, item := range items {
		result[i] = toTrendResponse(item)
	}
	return result
}

func toTrendStatsResponse(stats *TrendStats) *TrendStatsResponse {
	if stats == nil {
		return &TrendStatsResponse{}
	}

	return &TrendStatsResponse{
		Total:             stats.Total,
		New:               stats.New,
		Candidate:         stats.Candidate,
		Selected:          stats.Selected,
		Rejected:          stats.Rejected,
		QueuedJobs:        stats.QueuedJobs,
		SucceededJobs:     stats.SucceededJobs,
		LatestTrendAt:     stats.LatestTrendAt,
		LatestPolledAt:    stats.LatestPolledAt,
		LatestEvaluatedAt: stats.LatestEvaluatedAt,
	}
}

func toSourceStatusResponse(status *SourceStatus) *SourceStatusResponse {
	if status == nil {
		return nil
	}

	return &SourceStatusResponse{
		ID:                  status.ID,
		Provider:            status.Provider,
		Name:                status.Name,
		SourceURL:           status.SourceURL,
		PollIntervalMinutes: status.PollIntervalMinutes,
		Status:              status.Status,
		LastPolledAt:        status.LastPolledAt,
		LastError:           status.LastError,
	}
}

func toSyncRunResponse(result *SyncResult) *TrendSyncRunResponse {
	if result == nil {
		return &TrendSyncRunResponse{}
	}

	return &TrendSyncRunResponse{
		SourceID:         result.SourceID,
		Fetched:          result.Fetched,
		Upserted:         result.Upserted,
		Inserted:         result.Inserted,
		Evaluated:        result.Evaluated,
		Candidates:       result.Candidates,
		EnqueuedScoreJob: result.EnqueuedScoreJob,
	}
}

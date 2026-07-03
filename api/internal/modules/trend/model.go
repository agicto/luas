package trend

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const (
	ProviderDailyDevHighlights = "daily_dev_highlights"

	TrendStatusNew       = "new"
	TrendStatusCandidate = "candidate"
	TrendStatusSelected  = "selected"
	TrendStatusRejected  = "rejected"
	TrendStatusArchived  = "archived"
)

type FetchedTrend struct {
	SourceURL     string
	ExternalID    string
	CanonicalURL  string
	Title         string
	Summary       string
	Channel       string
	Language      string
	HighlightedAt *time.Time
	Significance  string
	Tags          []string
	Metrics       map[string]any
	RawPayload    json.RawMessage
}

func (t FetchedTrend) DedupeHash(sourceID string) string {
	parts := []string{
		strings.TrimSpace(sourceID),
		strings.TrimSpace(t.ExternalID),
		strings.TrimSpace(t.CanonicalURL),
		strings.TrimSpace(t.Title),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

type SyncResult struct {
	SourceID         string
	Fetched          int
	Upserted         int
	Inserted         int
	Evaluated        int
	Candidates       int
	EnqueuedScoreJob int
}

type EvaluationResult struct {
	TrendID          string
	HScore           int
	KScore           int
	RScore           int
	BrandFitScore    int
	RiskScore        int
	Recommended      bool
	TargetAudience   string
	RecommendedAngle string
	RiskNotes        string
}

type TrendListFilter struct {
	Status      string
	Channel     string
	Search      string
	Recommended *bool
	Page        int
	PerPage     int
}

type TrendListItem struct {
	ID               string     `gorm:"column:id"`
	SourceProvider   string     `gorm:"column:source_provider"`
	SourceName       string     `gorm:"column:source_name"`
	CanonicalURL     string     `gorm:"column:canonical_url"`
	Title            string     `gorm:"column:title"`
	Summary          string     `gorm:"column:summary"`
	Channel          string     `gorm:"column:channel"`
	Language         string     `gorm:"column:language"`
	HighlightedAt    *time.Time `gorm:"column:highlighted_at"`
	Significance     string     `gorm:"column:significance"`
	Status           string     `gorm:"column:status"`
	HScore           *int       `gorm:"column:h_score"`
	KScore           *int       `gorm:"column:k_score"`
	RScore           *int       `gorm:"column:r_score"`
	BrandFitScore    *int       `gorm:"column:brand_fit_score"`
	RiskScore        *int       `gorm:"column:risk_score"`
	TotalScore       *int       `gorm:"column:total_score"`
	Recommended      *bool      `gorm:"column:recommended"`
	TargetAudience   string     `gorm:"column:target_audience"`
	RecommendedAngle string     `gorm:"column:recommended_angle"`
	RiskNotes        string     `gorm:"column:risk_notes"`
	EvaluatedAt      *time.Time `gorm:"column:evaluated_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

type TrendStats struct {
	Total             int64      `gorm:"column:total"`
	New               int64      `gorm:"column:new_count"`
	Candidate         int64      `gorm:"column:candidate_count"`
	Selected          int64      `gorm:"column:selected_count"`
	Rejected          int64      `gorm:"column:rejected_count"`
	QueuedJobs        int64      `gorm:"column:queued_jobs"`
	SucceededJobs     int64      `gorm:"column:succeeded_jobs"`
	LatestTrendAt     *time.Time `gorm:"column:latest_trend_at"`
	LatestPolledAt    *time.Time `gorm:"column:latest_polled_at"`
	LatestEvaluatedAt *time.Time `gorm:"column:latest_evaluated_at"`
}

type SourceStatus struct {
	ID                  string     `gorm:"column:id"`
	Provider            string     `gorm:"column:provider"`
	Name                string     `gorm:"column:name"`
	SourceURL           string     `gorm:"column:source_url"`
	PollIntervalMinutes int        `gorm:"column:poll_interval_minutes"`
	Status              string     `gorm:"column:status"`
	LastPolledAt        *time.Time `gorm:"column:last_polled_at"`
	LastError           string     `gorm:"column:last_error"`
}

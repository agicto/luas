package trend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureDailyDevSource(ctx context.Context, sourceURL string) (string, error) {
	if r == nil || r.db == nil {
		return "", errors.New("trend repository requires a database")
	}

	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		sourceURL = DailyDevHighlightsURL
	}

	var sourceID string
	err := r.db.WithContext(ctx).Raw(`
		insert into content_sources (provider, name, source_url, poll_interval_minutes, status, config)
		values (
			?,
			?,
			?,
			10,
			'active',
			'{"adapter":"next_data_highlights","preferred_future_adapter":"daily_dev_public_api"}'::jsonb
		)
		on conflict (provider, source_url) do update
		set
			poll_interval_minutes = excluded.poll_interval_minutes,
			status = 'active',
			updated_at = now()
		returning id
	`, ProviderDailyDevHighlights, "daily.dev Highlights", sourceURL).Scan(&sourceID).Error
	if err != nil {
		return "", fmt.Errorf("ensure daily.dev content source: %w", err)
	}
	if sourceID == "" {
		return "", errors.New("ensure daily.dev content source returned empty id")
	}
	return sourceID, nil
}

func (r *Repository) MarkSourcePolled(ctx context.Context, sourceID string, syncErr error) error {
	if r == nil || r.db == nil || strings.TrimSpace(sourceID) == "" {
		return nil
	}

	var lastErr any
	if syncErr != nil {
		lastErr = syncErr.Error()
	}

	if err := r.db.WithContext(ctx).Exec(`
		update content_sources
		set last_polled_at = now(), last_error = ?, updated_at = now()
		where id = ?
	`, lastErr, sourceID).Error; err != nil {
		return fmt.Errorf("mark content source polled: %w", err)
	}
	return nil
}

func (r *Repository) UpsertFetchedTrend(ctx context.Context, sourceID string, item FetchedTrend) (string, bool, error) {
	if r == nil || r.db == nil {
		return "", false, errors.New("trend repository requires a database")
	}

	metrics, err := marshalJSONObject(item.Metrics)
	if err != nil {
		return "", false, fmt.Errorf("encode trend metrics: %w", err)
	}
	rawPayload := item.RawPayload
	if len(rawPayload) == 0 {
		rawPayload = json.RawMessage(`{}`)
	}
	if !json.Valid(rawPayload) {
		return "", false, errors.New("trend raw payload must be valid JSON")
	}

	var highlightedAt any
	if item.HighlightedAt != nil {
		highlightedAt = *item.HighlightedAt
	}

	var row struct {
		ID       string
		Inserted bool
	}
	err = r.db.WithContext(ctx).Raw(`
		with upserted as (
			insert into trend_items (
				source_id,
				external_id,
				canonical_url,
				title,
				summary,
				channel,
				language,
				highlighted_at,
				significance,
				tags,
				metrics,
				raw_payload,
				dedupe_hash,
				status
			)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, '{}'::text[], ?::jsonb, ?::jsonb, ?, 'new')
			on conflict (source_id, external_id) do update
			set
				canonical_url = excluded.canonical_url,
				title = excluded.title,
				summary = excluded.summary,
				channel = excluded.channel,
				language = excluded.language,
				highlighted_at = excluded.highlighted_at,
				significance = excluded.significance,
				metrics = excluded.metrics,
				raw_payload = excluded.raw_payload,
				updated_at = now()
			returning id, (xmax = 0) as inserted
		)
		select id, inserted from upserted
	`,
		sourceID,
		strings.TrimSpace(item.ExternalID),
		strings.TrimSpace(item.CanonicalURL),
		strings.TrimSpace(item.Title),
		strings.TrimSpace(item.Summary),
		strings.TrimSpace(item.Channel),
		firstNonEmpty(item.Language, "en"),
		highlightedAt,
		strings.TrimSpace(item.Significance),
		string(metrics),
		string(rawPayload),
		item.DedupeHash(sourceID),
	).Scan(&row).Error
	if err != nil {
		return "", false, fmt.Errorf("upsert trend item %q: %w", item.ExternalID, err)
	}
	if row.ID == "" {
		return "", false, errors.New("upsert trend item returned empty id")
	}
	return row.ID, row.Inserted, nil
}

func (r *Repository) EnqueueScoreJob(ctx context.Context, trendID string) error {
	if strings.TrimSpace(trendID) == "" {
		return nil
	}
	return r.db.WithContext(ctx).Exec(`
		insert into automation_jobs (job_type, status, scheduled_for, payload)
		values ('score_trend', 'queued', now(), jsonb_build_object('trend_id', ?::text))
	`, trendID).Error
}

func (r *Repository) ListUnevaluated(ctx context.Context, limit int) ([]TrendForEvaluation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows []TrendForEvaluation
	if err := r.db.WithContext(ctx).Raw(`
		select
			t.id,
			t.title,
			coalesce(t.summary, '') as summary,
			coalesce(t.channel, '') as channel,
			coalesce(t.significance, '') as significance,
			t.highlighted_at
		from trend_items t
		where not exists (
			select 1 from trend_evaluations e where e.trend_id = t.id
		)
		order by t.highlighted_at desc nulls last, t.created_at desc
		limit ?
	`, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list unevaluated trends: %w", err)
	}
	return rows, nil
}

func (r *Repository) SaveEvaluation(ctx context.Context, result EvaluationResult) error {
	output, err := marshalJSONObject(map[string]any{
		"rules": []string{"daily.dev significance", "channel fit", "summary density", "AGI01 builder fit"},
	})
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			insert into trend_evaluations (
				trend_id,
				model_provider,
				model,
				prompt_version,
				h_score,
				k_score,
				r_score,
				brand_fit_score,
				risk_score,
				confidence,
				recommended,
				target_audience,
				recommended_angle,
				risk_notes,
				output
			)
			values (?, 'rules', 'trend-sync-v1', 'rule-v1', ?, ?, ?, ?, ?, 0.650, ?, ?, ?, ?, ?::jsonb)
		`,
			result.TrendID,
			result.HScore,
			result.KScore,
			result.RScore,
			result.BrandFitScore,
			result.RiskScore,
			result.Recommended,
			result.TargetAudience,
			result.RecommendedAngle,
			result.RiskNotes,
			string(output),
		).Error; err != nil {
			return fmt.Errorf("insert trend evaluation: %w", err)
		}

		status := TrendStatusNew
		if result.Recommended {
			status = TrendStatusCandidate
		}
		if err := tx.Exec(`
			update trend_items
			set status = ?, updated_at = ?
			where id = ?
		`, status, time.Now().UTC(), result.TrendID).Error; err != nil {
			return fmt.Errorf("update trend status: %w", err)
		}

		return nil
	})
}

func (r *Repository) ListTrends(ctx context.Context, filter TrendListFilter) ([]TrendListItem, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, errors.New("trend repository requires a database")
	}

	filter = normalizeTrendListFilter(filter)
	whereSQL, args := buildTrendWhere(filter)
	countSQL := fmt.Sprintf(`
		select count(*)
		from trend_items t
		left join lateral (
			select recommended
			from trend_evaluations e
			where e.trend_id = t.id
			order by e.created_at desc
			limit 1
		) e on true
		where %s
	`, whereSQL)

	var total int64
	if err := r.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count trends: %w", err)
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, filter.PerPage, (filter.Page-1)*filter.PerPage)
	querySQL := fmt.Sprintf(`
		select
			t.id::text as id,
			s.provider as source_provider,
			s.name as source_name,
			t.canonical_url,
			t.title,
			coalesce(t.summary, '') as summary,
			coalesce(t.channel, '') as channel,
			t.language,
			t.highlighted_at,
			coalesce(t.significance, '') as significance,
			t.status::text as status,
			e.h_score,
			e.k_score,
			e.r_score,
			e.brand_fit_score,
			e.risk_score,
			e.total_score,
			e.recommended,
			coalesce(e.target_audience, '') as target_audience,
			coalesce(e.recommended_angle, '') as recommended_angle,
			coalesce(e.risk_notes, '') as risk_notes,
			e.created_at as evaluated_at,
			t.created_at,
			t.updated_at
		from trend_items t
		join content_sources s on s.id = t.source_id
		left join lateral (
			select
				h_score,
				k_score,
				r_score,
				brand_fit_score,
				risk_score,
				total_score,
				recommended,
				target_audience,
				recommended_angle,
				risk_notes,
				created_at
			from trend_evaluations e
			where e.trend_id = t.id
			order by e.created_at desc
			limit 1
		) e on true
		where %s
		order by e.total_score desc nulls last, t.highlighted_at desc nulls last, t.created_at desc
		limit ? offset ?
	`, whereSQL)

	var rows []TrendListItem
	if err := r.db.WithContext(ctx).Raw(querySQL, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list trends: %w", err)
	}

	return rows, total, nil
}

func (r *Repository) GetStats(ctx context.Context) (*TrendStats, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("trend repository requires a database")
	}

	var stats TrendStats
	if err := r.db.WithContext(ctx).Raw(`
		select
			count(t.id) as total,
			count(t.id) filter (where t.status::text = 'new') as new_count,
			count(t.id) filter (where t.status::text = 'candidate') as candidate_count,
			count(t.id) filter (where t.status::text = 'selected') as selected_count,
			count(t.id) filter (where t.status::text = 'rejected') as rejected_count,
			coalesce((select count(*) from automation_jobs where status::text = 'queued'), 0) as queued_jobs,
			coalesce((select count(*) from automation_jobs where status::text = 'succeeded'), 0) as succeeded_jobs,
			max(t.highlighted_at) as latest_trend_at,
			(select max(last_polled_at) from content_sources) as latest_polled_at,
			(select max(created_at) from trend_evaluations) as latest_evaluated_at
		from trend_items t
	`).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("load trend stats: %w", err)
	}

	return &stats, nil
}

func (r *Repository) GetDailyDevSourceStatus(ctx context.Context) (*SourceStatus, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("trend repository requires a database")
	}

	var status SourceStatus
	tx := r.db.WithContext(ctx).Raw(`
		select
			id::text as id,
			provider,
			name,
			source_url,
			poll_interval_minutes,
			status::text as status,
			last_polled_at,
			coalesce(last_error, '') as last_error
		from content_sources
		where provider = ?
		order by created_at asc
		limit 1
	`, ProviderDailyDevHighlights).Scan(&status)
	if tx.Error != nil {
		return nil, fmt.Errorf("load daily.dev source status: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}
	return &status, nil
}

type TrendForEvaluation struct {
	ID            string
	Title         string
	Summary       string
	Channel       string
	Significance  string
	HighlightedAt *time.Time
}

func marshalJSONObject(value any) ([]byte, error) {
	if value == nil {
		return []byte(`{}`), nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 || string(encoded) == "null" {
		return []byte(`{}`), nil
	}
	return encoded, nil
}

func normalizeTrendListFilter(filter TrendListFilter) TrendListFilter {
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Channel = strings.TrimSpace(filter.Channel)
	filter.Search = strings.TrimSpace(filter.Search)

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 15
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	return filter
}

func buildTrendWhere(filter TrendListFilter) (string, []any) {
	clauses := []string{"1 = 1"}
	args := []any{}

	if filter.Status != "" {
		clauses = append(clauses, "t.status::text = ?")
		args = append(args, filter.Status)
	}
	if filter.Channel != "" {
		clauses = append(clauses, "t.channel = ?")
		args = append(args, filter.Channel)
	}
	if filter.Search != "" {
		like := "%" + escapeLike(filter.Search) + "%"
		clauses = append(clauses, `(t.title ilike ? escape '\' or coalesce(t.summary, '') ilike ? escape '\' or coalesce(t.channel, '') ilike ? escape '\')`)
		args = append(args, like, like, like)
	}
	if filter.Recommended != nil {
		clauses = append(clauses, "coalesce(e.recommended, false) = ?")
		args = append(args, *filter.Recommended)
	}

	return strings.Join(clauses, " and "), args
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

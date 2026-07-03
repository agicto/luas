package trend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const candidateScoreThreshold = 18

type Service struct {
	repo    *Repository
	fetcher *DailyDevFetcher
}

func NewService(repo *Repository, fetcher *DailyDevFetcher) *Service {
	return &Service{repo: repo, fetcher: fetcher}
}

func (s *Service) SyncDailyDevHighlights(ctx context.Context, sourceURL string) (*SyncResult, error) {
	if s == nil || s.repo == nil || s.fetcher == nil {
		return nil, fmt.Errorf("trend service is not configured")
	}

	sourceID, err := s.repo.EnsureDailyDevSource(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{SourceID: sourceID}
	items, fetchErr := s.fetcher.FetchHighlights(ctx, sourceURL)
	if fetchErr != nil {
		return result, errors.Join(fetchErr, s.repo.MarkSourcePolled(ctx, sourceID, fetchErr))
	}
	result.Fetched = len(items)

	for _, item := range items {
		if strings.TrimSpace(item.Title) == "" {
			continue
		}
		trendID, inserted, upsertErr := s.repo.UpsertFetchedTrend(ctx, sourceID, item)
		if upsertErr != nil {
			return result, errors.Join(upsertErr, s.repo.MarkSourcePolled(ctx, sourceID, upsertErr))
		}
		result.Upserted++
		if inserted {
			result.Inserted++
			if enqueueErr := s.repo.EnqueueScoreJob(ctx, trendID); enqueueErr == nil {
				result.EnqueuedScoreJob++
			}
		}
	}

	evaluated, candidates, err := s.EvaluatePending(ctx, 100)
	if err != nil {
		return result, errors.Join(err, s.repo.MarkSourcePolled(ctx, sourceID, err))
	}
	result.Evaluated = evaluated
	result.Candidates = candidates

	if err := s.repo.MarkSourcePolled(ctx, sourceID, nil); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) EvaluatePending(ctx context.Context, limit int) (int, int, error) {
	items, err := s.repo.ListUnevaluated(ctx, limit)
	if err != nil {
		return 0, 0, err
	}

	candidates := 0
	for _, item := range items {
		evaluation := evaluateTrend(item)
		if evaluation.Recommended {
			candidates++
		}
		if err := s.repo.SaveEvaluation(ctx, evaluation); err != nil {
			return 0, candidates, err
		}
	}
	return len(items), candidates, nil
}

func (s *Service) ListTrends(ctx context.Context, filter TrendListFilter) ([]TrendListItem, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, fmt.Errorf("trend service is not configured")
	}

	return s.repo.ListTrends(ctx, filter)
}

func (s *Service) GetStats(ctx context.Context) (*TrendStats, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("trend service is not configured")
	}

	return s.repo.GetStats(ctx)
}

func (s *Service) GetDailyDevSourceStatus(ctx context.Context) (*SourceStatus, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("trend service is not configured")
	}

	return s.repo.GetDailyDevSourceStatus(ctx)
}

func evaluateTrend(item TrendForEvaluation) EvaluationResult {
	title := strings.ToLower(item.Title)
	summary := strings.ToLower(item.Summary)
	channel := strings.ToLower(item.Channel)
	significance := strings.ToLower(item.Significance)
	text := title + " " + summary + " " + channel

	hScore := 1
	switch significance {
	case "breaking":
		hScore = 5
	case "major":
		hScore = 3
	}
	if containsAny(title, "cve", "rce", "retires", "shutdown", "raises", "launches", "ships", "open-sources") {
		hScore++
	}
	hScore = clampScore(hScore)

	kScore := 2
	if len(item.Summary) > 240 {
		kScore = 4
	} else if len(item.Summary) > 120 {
		kScore = 3
	}
	if containsAny(text, "api", "model", "security", "database", "developer", "open source", "github", "android", "postgres", "linux") {
		kScore++
	}
	kScore = clampScore(kScore)

	rScore := 1
	if containsAny(text, "ai", "agent", "coding", "developer", "github", "claude", "openai", "cursor", "security", "supply chain") {
		rScore = 3
	}
	if containsAny(text, "shutdown", "breaking", "exploited", "malicious", "raises") {
		rScore++
	}
	rScore = clampScore(rScore)

	brandFit := 1
	if containsAny(text, "ai", "agent", "llm", "model", "coding", "claude", "openai", "cursor") {
		brandFit = 5
	} else if containsAny(text, "developer", "github", "open source", "api", "postgres", "linux", "security", "supply chain") {
		brandFit = 4
	} else if containsAny(channel, "backend", "webdev", "databases", "opensource", "security", "vibes") {
		brandFit = 3
	}

	riskScore := 0
	if containsAny(text, "supreme court", "trump", "ftc", "lawsuit", "politic") {
		riskScore = 2
	}
	if containsAny(text, "rumor", "alleged") {
		riskScore++
	}

	total := hScore + kScore + rScore + brandFit - riskScore
	recommended := total >= candidateScoreThreshold && brandFit >= 4

	return EvaluationResult{
		TrendID:          item.ID,
		HScore:           hScore,
		KScore:           kScore,
		RScore:           rScore,
		BrandFitScore:    brandFit,
		RiskScore:        riskScore,
		Recommended:      recommended,
		TargetAudience:   inferAudience(text, channel),
		RecommendedAngle: recommendedAngle(item),
		RiskNotes:        inferRiskNotes(riskScore),
	}
}

func recommendedAngle(item TrendForEvaluation) string {
	channel := strings.TrimSpace(item.Channel)
	if channel == "" {
		channel = "developer"
	}
	return fmt.Sprintf("从 %s 热点切入，写给 builder：这件事对 AI 产品、工程效率或安全边界有什么实际影响。", channel)
}

func inferAudience(text, channel string) string {
	switch {
	case containsAny(text, "security", "cve", "rce", "supply chain", "malicious"):
		return "programmers"
	case containsAny(text, "raises", "funding", "enterprise", "market"):
		return "ai_founders"
	case containsAny(text, "agent", "llm", "model", "coding", "github"):
		return "ai_builders"
	case containsAny(channel, "webdev", "backend", "databases", "opensource"):
		return "programmers"
	default:
		return "mixed_builders"
	}
}

func inferRiskNotes(score int) string {
	if score == 0 {
		return "未发现明显高风险信号；仍需在研究 brief 阶段核验一手来源。"
	}
	return "存在政策、法律、传闻或过热表达风险；写作前必须做交叉验证并软化结论。"
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 5 {
		return 5
	}
	return score
}

func NextRunAfter(interval time.Duration) time.Time {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return time.Now().Add(interval)
}

package trend

import (
	"os"
	"testing"
)

func TestParseDailyDevHighlights(t *testing.T) {
	body, err := os.ReadFile("../../../testdata/daily_dev_highlights.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	items, err := ParseDailyDevHighlights(body)
	if err != nil {
		t.Fatalf("ParseDailyDevHighlights() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	if got := items[0].Title; got != "GitHub Models fully retires on July 30 with playground and inference API shutdown" {
		t.Fatalf("first title = %q", got)
	}
	if got := items[0].Summary; got == "" {
		t.Fatalf("first summary is empty")
	}
	if got := items[1].Summary; got != "Shared summary text" {
		t.Fatalf("shared summary = %q", got)
	}
}

func TestEvaluateTrend(t *testing.T) {
	result := evaluateTrend(TrendForEvaluation{
		ID:           "trend-1",
		Title:        "Multiple Backstage npm plugins backdoored with supply chain worm and credential harvester",
		Summary:      "A malicious payload targets developer machines, GitHub Actions, cloud credentials, package registries, password managers, SSH keys, Cursor, Claude Code and VS Code. Teams need to rotate secrets from a clean machine and inspect CI activity logs before trusting visible Git history.",
		Channel:      "backend",
		Significance: "breaking",
	})

	if !result.Recommended {
		t.Fatalf("expected security AI trend to be recommended: %+v", result)
	}
	if result.BrandFitScore < 4 {
		t.Fatalf("brand fit = %d, want >= 4", result.BrandFitScore)
	}
}

package trend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const DailyDevHighlightsURL = "https://daily.dev/highlights"

var nextDataPattern = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

// DailyDevFetcher fetches and parses the daily.dev highlights page.
type DailyDevFetcher struct {
	client *http.Client
}

func NewDailyDevFetcher(client *http.Client) *DailyDevFetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &DailyDevFetcher{client: client}
}

func (f *DailyDevFetcher) FetchHighlights(ctx context.Context, sourceURL string) ([]FetchedTrend, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		sourceURL = DailyDevHighlightsURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create daily.dev request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "AGI01 trend-sync/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch daily.dev highlights: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("daily.dev highlights returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read daily.dev highlights: %w", err)
	}

	items, err := ParseDailyDevHighlights(body)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].SourceURL = sourceURL
	}
	return items, nil
}

func ParseDailyDevHighlights(body []byte) ([]FetchedTrend, error) {
	match := nextDataPattern.FindSubmatch(body)
	if len(match) < 2 {
		return nil, errors.New("daily.dev highlights __NEXT_DATA__ not found")
	}

	payload := html.UnescapeString(string(match[1]))
	var page nextDataPayload
	if err := json.Unmarshal([]byte(payload), &page); err != nil {
		return nil, fmt.Errorf("decode daily.dev highlights __NEXT_DATA__: %w", err)
	}

	for _, query := range page.Props.PageProps.DehydratedState.Queries {
		if !query.IsHighlightsPage() {
			continue
		}
		return query.State.Data.ToFetchedTrends()
	}

	return nil, errors.New("daily.dev highlights-page query not found")
}

type nextDataPayload struct {
	Props struct {
		PageProps struct {
			DehydratedState struct {
				Queries []highlightQuery `json:"queries"`
			} `json:"dehydratedState"`
		} `json:"pageProps"`
	} `json:"props"`
}

type highlightQuery struct {
	QueryKey []string `json:"queryKey"`
	State    struct {
		Data highlightData `json:"data"`
	} `json:"state"`
}

func (q highlightQuery) IsHighlightsPage() bool {
	return len(q.QueryKey) > 0 && q.QueryKey[0] == "highlights-page"
}

type highlightData struct {
	MajorHeadlines struct {
		Edges []struct {
			Node highlightNode `json:"node"`
		} `json:"edges"`
	} `json:"majorHeadlines"`
}

func (d highlightData) ToFetchedTrends() ([]FetchedTrend, error) {
	items := make([]FetchedTrend, 0, len(d.MajorHeadlines.Edges))
	for _, edge := range d.MajorHeadlines.Edges {
		node := edge.Node
		if strings.TrimSpace(node.ID) == "" || strings.TrimSpace(node.Headline) == "" {
			continue
		}

		raw, err := json.Marshal(node)
		if err != nil {
			return nil, fmt.Errorf("encode daily.dev highlight raw payload: %w", err)
		}

		items = append(items, FetchedTrend{
			ExternalID:    node.ID,
			CanonicalURL:  firstNonEmpty(node.Post.CommentsPermalink, DailyDevHighlightsURL),
			Title:         node.Headline,
			Summary:       node.Summary(),
			Channel:       node.Channel,
			Language:      "en",
			HighlightedAt: node.HighlightedAt,
			Significance:  node.Significance,
			Tags:          []string{},
			Metrics: map[string]any{
				"post_id":   node.Post.ID,
				"post_type": node.Post.Type,
			},
			RawPayload: json.RawMessage(raw),
		})
	}
	return items, nil
}

type highlightNode struct {
	ID            string     `json:"id"`
	Channel       string     `json:"channel"`
	Headline      string     `json:"headline"`
	HighlightedAt *time.Time `json:"highlightedAt"`
	Significance  string     `json:"significance"`
	Post          struct {
		ID                string `json:"id"`
		Type              string `json:"type"`
		CommentsPermalink string `json:"commentsPermalink"`
		Summary           string `json:"summary"`
		SharedPost        *struct {
			Summary string `json:"summary"`
		} `json:"sharedPost"`
	} `json:"post"`
}

func (n highlightNode) Summary() string {
	if strings.TrimSpace(n.Post.Summary) != "" {
		return strings.TrimSpace(n.Post.Summary)
	}
	if n.Post.SharedPost != nil {
		return strings.TrimSpace(n.Post.SharedPost.Summary)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

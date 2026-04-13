package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zgiai/zgo/pkg/env"
)

type GitHubClient interface {
	GetViewer(ctx context.Context, token string) (*GitHubViewer, error)
	ListRepositories(ctx context.Context, token, query string, limit int) ([]GitHubRepository, error)
}

type githubClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewGitHubClient() GitHubClient {
	baseURL := env.Get("GITHUB_API_BASE_URL", "https://api.github.com")
	return &githubClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *githubClient) GetViewer(ctx context.Context, token string) (*GitHubViewer, error) {
	var payload struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := c.get(ctx, token, "/user", &payload); err != nil {
		return nil, err
	}

	return &GitHubViewer{
		Login:       payload.Login,
		DisplayName: firstNonEmpty(payload.Name, payload.Login),
		AvatarURL:   payload.AvatarURL,
	}, nil
}

func (c *githubClient) ListRepositories(ctx context.Context, token, query string, limit int) ([]GitHubRepository, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	params := url.Values{}
	params.Set("sort", "updated")
	params.Set("direction", "desc")
	params.Set("per_page", fmt.Sprintf("%d", limit))
	params.Set("affiliation", "owner,collaborator,organization_member")

	var payload []struct {
		ID            int64     `json:"id"`
		Name          string    `json:"name"`
		FullName      string    `json:"full_name"`
		CloneURL      string    `json:"clone_url"`
		HTMLURL       string    `json:"html_url"`
		DefaultBranch string    `json:"default_branch"`
		Private       bool      `json:"private"`
		Description   string    `json:"description"`
		Language      string    `json:"language"`
		UpdatedAt     time.Time `json:"updated_at"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}

	path := "/user/repos"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	if err := c.get(ctx, token, path, &payload); err != nil {
		return nil, err
	}

	filter := strings.ToLower(strings.TrimSpace(query))
	repositories := make([]GitHubRepository, 0, len(payload))
	for _, repo := range payload {
		if filter != "" {
			matchTarget := strings.ToLower(strings.Join([]string{repo.Name, repo.FullName, repo.Description, repo.Language}, " "))
			if !strings.Contains(matchTarget, filter) {
				continue
			}
		}

		repositories = append(repositories, GitHubRepository{
			ID:            repo.ID,
			Name:          repo.Name,
			FullName:      repo.FullName,
			Owner:         repo.Owner.Login,
			DefaultBranch: repo.DefaultBranch,
			CloneURL:      repo.CloneURL,
			HTMLURL:       repo.HTMLURL,
			Private:       repo.Private,
			Description:   repo.Description,
			Language:      repo.Language,
			UpdatedAt:     repo.UpdatedAt,
		})
	}

	return repositories, nil
}

func (c *githubClient) get(ctx context.Context, token, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return fmt.Errorf("github api returned %d: %s", resp.StatusCode, payload.Message)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	infrahttp "github.com/zgiai/zgo/internal/infra/http"
)

// OpenAIProvider implements text generation with the OpenAI Responses API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	timeout time.Duration
}

// NewOpenAIProvider creates a new OpenAI-backed provider.
func NewOpenAIProvider(cfg ProviderConfig, timeout time.Duration) *OpenAIProvider {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	return &OpenAIProvider{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		baseURL: strings.TrimRight(baseURL, "/"),
		timeout: timeout,
	}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return ProviderOpenAI
}

// GenerateText calls the OpenAI Responses API and aggregates output_text items.
func (p *OpenAIProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	body := map[string]any{
		"model": req.Model,
		"input": req.Input,
	}
	if req.Instructions != "" {
		body["instructions"] = req.Instructions
	}
	if req.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{
			"effort": req.ReasoningEffort,
		}
	}

	resp, err := infrahttp.New().
		BaseURL(p.baseURL).
		Timeout(p.timeout).
		WithToken(p.apiKey).
		AcceptJSON().
		AsJSON().
		PostContext(ctx, "/responses", body)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}

	var payload openAIResponse
	if err := resp.JSON(&payload); err != nil {
		return nil, fmt.Errorf("openai: failed to decode response: %w", err)
	}

	if resp.Failed() {
		message := strings.TrimSpace(payload.Error.Message)
		if message == "" {
			message = strings.TrimSpace(resp.String())
		}
		return nil, fmt.Errorf("openai: %s", message)
	}

	text := payload.outputText()
	if text == "" {
		return nil, ErrEmptyResponseText
	}

	return &TextResponse{
		ID:       payload.ID,
		Provider: ProviderOpenAI,
		Model:    payload.Model,
		Text:     text,
	}, nil
}

type openAIResponse struct {
	ID     string `json:"id"`
	Model  string `json:"model"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r *openAIResponse) outputText() string {
	parts := make([]string, 0, len(r.Output))
	for _, item := range r.Output {
		for _, content := range item.Content {
			if content.Type != "output_text" {
				continue
			}
			text := strings.TrimSpace(content.Text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	infrahttp "github.com/zgiai/luas/api/internal/infra/http"
)

// OpenAIProvider implements text generation with the OpenAI Responses API.
//
// Implements both [Provider] (one-shot) and [StreamingProvider] (SSE).
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
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
		// Streaming uses a separate http.Client without timeout — the SSE
		// connection is expected to be long-lived; we propagate ctx instead.
		client: &http.Client{},
	}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return ProviderOpenAI
}

// GenerateText calls the OpenAI Responses API and aggregates output_text items.
func (p *OpenAIProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	body := p.requestBody(req, false)

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

// GenerateTextStream calls the OpenAI Responses API with stream=true and
// returns a channel of [StreamChunk]. The channel is closed when the
// stream ends or fails.
//
// SSE wire format: each event is a `data: {...}` line. We forward
// `response.output_text.delta` events as Delta chunks and treat
// `response.completed` / `response.error` / `[DONE]` as terminal.
func (p *OpenAIProvider) GenerateTextStream(ctx context.Context, req *TextRequest) (<-chan StreamChunk, error) {
	body := p.requestBody(req, true)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: encode stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai: build stream request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: stream request failed: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		defer httpResp.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 4096))
		return nil, fmt.Errorf("openai: stream HTTP %d: %s", httpResp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	out := make(chan StreamChunk, 16)
	go func() {
		defer httpResp.Body.Close()
		defer close(out)

		scanner := bufio.NewScanner(httpResp.Body)
		// SSE events can be large with structured JSON — give scanner room.
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				if data == "[DONE]" {
					return
				}
				continue
			}

			var ev openAIStreamEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				// Tolerate one bad line — but report it.
				sendOrAbort(ctx, out, StreamChunk{Err: fmt.Errorf("openai: bad SSE payload: %w", err)})
				return
			}

			switch ev.Type {
			case "response.output_text.delta":
				if ev.Delta != "" {
					if !sendOrAbort(ctx, out, StreamChunk{Delta: ev.Delta}) {
						return
					}
				}
			case "response.completed":
				return
			case "response.error", "error":
				message := strings.TrimSpace(ev.Error.Message)
				if message == "" {
					message = "openai: stream error"
				}
				sendOrAbort(ctx, out, StreamChunk{Err: fmt.Errorf("openai: %s", message)})
				return
			}
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
			sendOrAbort(ctx, out, StreamChunk{Err: fmt.Errorf("openai: stream read: %w", err)})
		}
	}()

	return out, nil
}

// requestBody assembles the JSON body shared by streaming and non-streaming
// calls. `stream` toggles the SSE behavior on the Responses API.
func (p *OpenAIProvider) requestBody(req *TextRequest, stream bool) map[string]any {
	body := map[string]any{
		"model": req.Model,
		"input": req.Input,
	}
	if stream {
		body["stream"] = true
	}
	if req.Instructions != "" {
		body["instructions"] = req.Instructions
	}
	if req.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{
			"effort": req.ReasoningEffort,
		}
	}
	return body
}

func sendOrAbort(ctx context.Context, out chan<- StreamChunk, chunk StreamChunk) bool {
	select {
	case out <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
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

// openAIStreamEvent is the minimal SSE event shape we care about.
// The Responses API emits a richer schema; we ignore fields we don't use.
type openAIStreamEvent struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIProviderGenerateText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %s, want /responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %s, want Bearer test-key", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["model"] != "gpt-5" {
			t.Fatalf("model = %v, want gpt-5", body["model"])
		}
		if body["input"] != "ping" {
			t.Fatalf("input = %v, want ping", body["input"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "resp_1",
			"model": "gpt-5",
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": "pong",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}, 5*time.Second)

	resp, err := provider.GenerateText(context.Background(), &TextRequest{
		Model:        "gpt-5",
		Input:        "ping",
		Instructions: "Be terse.",
	})
	if err != nil {
		t.Fatalf("GenerateText() error = %v", err)
	}

	if resp.Text != "pong" {
		t.Fatalf("text = %q, want pong", resp.Text)
	}
	if resp.Provider != ProviderOpenAI {
		t.Fatalf("provider = %q, want %q", resp.Provider, ProviderOpenAI)
	}
}

func TestManagerUsesDefaults(t *testing.T) {
	manager := NewManager(Config{
		Enabled:         true,
		DefaultProvider: ProviderOpenAI,
		DefaultModel:    "gpt-5",
		OpenAI: ProviderConfig{
			APIKey:  "test-key",
			BaseURL: "http://127.0.0.1:1",
		},
	})

	if _, err := manager.GenerateText(context.Background(), &TextRequest{}); err != ErrInputRequired {
		t.Fatalf("GenerateText() error = %v, want %v", err, ErrInputRequired)
	}
}

func TestOpenAIProviderGenerateTextStream(t *testing.T) {
	// Minimal SSE event stream: two deltas then completed.
	events := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"hello "}`,
		``,
		`data: {"type":"response.output_text.delta","delta":"world"}`,
		``,
		`data: {"type":"response.completed"}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["stream"] != true {
			t.Fatalf("stream field = %v, want true", body["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(events))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}, 5*time.Second)

	ch, err := provider.GenerateTextStream(context.Background(), &TextRequest{
		Model: "gpt-5",
		Input: "ping",
	})
	if err != nil {
		t.Fatalf("GenerateTextStream() error = %v", err)
	}

	var got strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		got.WriteString(chunk.Delta)
	}

	if want := "hello world"; got.String() != want {
		t.Fatalf("stream output = %q, want %q", got.String(), want)
	}
}

func TestManagerStreamingUnsupported(t *testing.T) {
	manager := NewManager(Config{
		Enabled:         true,
		DefaultProvider: "fake",
		DefaultModel:    "any",
	})
	// Register a Provider that does NOT implement StreamingProvider.
	manager.providers["fake"] = &nonStreamingFake{}

	_, err := manager.GenerateTextStream(context.Background(), &TextRequest{Input: "x"})
	if err == nil || !strings.Contains(err.Error(), "does not support streaming") {
		t.Fatalf("err = %v, want ErrStreamingUnsupported", err)
	}
}

type nonStreamingFake struct{}

func (f *nonStreamingFake) Name() string { return "fake" }
func (f *nonStreamingFake) GenerateText(context.Context, *TextRequest) (*TextResponse, error) {
	return &TextResponse{Text: "ok"}, nil
}

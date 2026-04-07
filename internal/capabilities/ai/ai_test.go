package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		if body["model"] != "gpt-5.4" {
			t.Fatalf("model = %v, want gpt-5.4", body["model"])
		}
		if body["input"] != "ping" {
			t.Fatalf("input = %v, want ping", body["input"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "resp_1",
			"model": "gpt-5.4",
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
		Model:        "gpt-5.4",
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
		DefaultModel:    "gpt-5.4",
		OpenAI: ProviderConfig{
			APIKey:  "test-key",
			BaseURL: "http://127.0.0.1:1",
		},
	})

	if _, err := manager.GenerateText(context.Background(), &TextRequest{}); err != ErrInputRequired {
		t.Fatalf("GenerateText() error = %v, want %v", err, ErrInputRequired)
	}
}

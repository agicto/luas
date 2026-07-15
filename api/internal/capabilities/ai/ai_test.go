package ai

import (
	"context"
	"encoding/json"
	"errors"
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
	}, 5*time.Second, RequestLimits{})

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
	if resp.ProviderResponseID != "resp_1" {
		t.Fatalf("provider response ID = %q, want resp_1", resp.ProviderResponseID)
	}
}

func TestOpenAIProviderGenerateTextReturnsProviderError(t *testing.T) {
	const sensitiveProviderMessage = "provider-secret-prompt-fragment"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": sensitiveProviderMessage,
			},
		})
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}, 5*time.Second, RequestLimits{})

	_, err := provider.GenerateText(context.Background(), &TextRequest{
		Model: "gpt-5",
		Input: "ping",
	})
	if err == nil {
		t.Fatal("GenerateText() error = nil, want provider status error")
	}
	if strings.Contains(err.Error(), sensitiveProviderMessage) {
		t.Fatalf("GenerateText() error = %q, provider response body must remain private", err)
	}
	if !strings.Contains(err.Error(), "HTTP status 429") {
		t.Fatalf("GenerateText() error = %q, want stable HTTP status", err)
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.StatusCode != http.StatusTooManyRequests || !providerErr.Retryable {
		t.Fatalf("GenerateText() error = %#v, want retryable ProviderError(429)", err)
	}
	if !errors.Is(err, ErrProviderRequestFailed) {
		t.Fatalf("GenerateText() error = %v, want ErrProviderRequestFailed category", err)
	}
}

func TestOpenAIProviderGenerateTextRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat("x", 5*1024*1024)))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}, 5*time.Second, RequestLimits{})

	_, err := provider.GenerateText(context.Background(), &TextRequest{
		Model: "gpt-5",
		Input: "ping",
	})
	if !errors.Is(err, ErrProviderResponseTooLarge) {
		t.Fatalf("GenerateText() error = %v, want bounded response error", err)
	}
}

func TestOpenAIProviderGenerateTextBlocksRedirects(t *testing.T) {
	redirected := make(chan struct{}, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected <- struct{}{}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "resp_redirected",
			"model": "gpt-5",
			"output": []map[string]any{{
				"type": "message",
				"content": []map[string]any{{
					"type": "output_text",
					"text": "redirected",
				}},
			}},
		})
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: origin.URL,
	}, 5*time.Second, RequestLimits{})

	_, err := provider.GenerateText(context.Background(), &TextRequest{
		Model: "gpt-5",
		Input: "ping",
	})
	if err == nil {
		t.Fatal("GenerateText() error = nil, want redirect rejection")
	}
	if !errors.Is(err, ErrProviderRequestFailed) {
		t.Fatalf("GenerateText() error = %v, want ErrProviderRequestFailed category", err)
	}
	select {
	case <-redirected:
		t.Fatal("provider client followed a redirect carrying an authenticated request")
	default:
	}
}

func TestOpenAIProviderRejectsMalformedBaseURL(t *testing.T) {
	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "https://user:pass@provider.example/v1?secret=value",
	}, 5*time.Second, RequestLimits{})

	_, err := provider.GenerateText(context.Background(), &TextRequest{Model: "provider-model", Input: "ping"})
	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("GenerateText() error = %v, want invalid provider configuration rejection", err)
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
	}, 5*time.Second, RequestLimits{})

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

func TestOpenAIProviderGenerateTextStreamHonorsRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("test server does not support flushing")
		}
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"))
		flusher.Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(250 * time.Millisecond):
		}
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}, 25*time.Millisecond, RequestLimits{})

	started := time.Now()
	ch, err := provider.GenerateTextStream(context.Background(), &TextRequest{
		Model: "gpt-5",
		Input: "ping",
	})
	if err != nil {
		t.Fatalf("GenerateTextStream() error = %v", err)
	}

	var streamErr error
	for chunk := range ch {
		if chunk.Err != nil {
			streamErr = chunk.Err
		}
	}
	if elapsed := time.Since(started); elapsed >= 150*time.Millisecond {
		t.Fatalf("stream elapsed = %s, want configured timeout to stop it", elapsed)
	}
	if streamErr == nil {
		t.Fatal("stream error = nil, want timeout error")
	}
	if !errors.Is(streamErr, context.DeadlineExceeded) || !errors.Is(streamErr, ErrProviderRequestFailed) {
		t.Fatalf("stream error = %v, want deadline and provider request categories", streamErr)
	}
}

func TestOpenAIProviderGenerateTextStreamHidesProviderErrorBody(t *testing.T) {
	const sensitiveProviderMessage = "private-provider-diagnostic"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"response.failed","error":{"message":"` + sensitiveProviderMessage + `"}}` + "\n\n"))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}, 5*time.Second, RequestLimits{})

	ch, err := provider.GenerateTextStream(context.Background(), &TextRequest{Model: "gpt-5", Input: "ping"})
	if err != nil {
		t.Fatalf("GenerateTextStream() error = %v", err)
	}
	var streamErr error
	for chunk := range ch {
		streamErr = chunk.Err
	}
	if streamErr == nil || strings.Contains(streamErr.Error(), sensitiveProviderMessage) {
		t.Fatalf("stream error = %v, want private stable provider error", streamErr)
	}
	var providerErr *ProviderError
	if !errors.As(streamErr, &providerErr) || !errors.Is(streamErr, ErrProviderRequestFailed) {
		t.Fatalf("stream error = %#v, want ProviderError category", streamErr)
	}
}

func TestManagerProviderNamesAreDeterministic(t *testing.T) {
	manager := NewManager(Config{Enabled: true})
	manager.providers["zeta"] = &namedFake{name: "zeta"}
	manager.providers["alpha"] = &namedFake{name: "alpha"}

	got := strings.Join(manager.ProviderNames(), ",")
	if got != "alpha,zeta" {
		t.Fatalf("ProviderNames() = %q, want alpha,zeta", got)
	}
}

func TestManagerRejectsInvalidConfiguredDefault(t *testing.T) {
	manager := NewManager(Config{
		Enabled:         true,
		DefaultProvider: "openai\nforged-log-line",
		DefaultModel:    "provider-model",
		OpenAI: ProviderConfig{
			APIKey: "test-key",
		},
	})

	_, err := manager.GenerateText(context.Background(), &TextRequest{Input: "ping"})
	if !errors.Is(err, ErrProviderRequired) {
		t.Fatalf("GenerateText() error = %v, want invalid default to fail closed", err)
	}
}

func TestRequestPolicyNormalizesUnsafeDirectOverrides(t *testing.T) {
	limits := normalizeRequestLimits(RequestLimits{
		MaxInputBytes:       maxInputLimitBytes + 1,
		MaxResponseBytes:    maxResponseLimitBytes + 1,
		MaxStreamEventBytes: maxStreamEventLimitBytes + 1,
	})
	if limits.MaxInputBytes != DefaultMaxInputBytes || limits.MaxResponseBytes != DefaultMaxResponseBytes || limits.MaxStreamEventBytes != DefaultMaxStreamEventBytes {
		t.Fatalf("normalized limits = %+v, want safe defaults", limits)
	}

	limits = normalizeRequestLimits(RequestLimits{
		MaxInputBytes:       DefaultMaxInputBytes,
		MaxResponseBytes:    64 * 1024,
		MaxStreamEventBytes: DefaultMaxStreamEventBytes,
	})
	if limits.MaxStreamEventBytes != 64*1024 {
		t.Fatalf("stream event limit = %d, want response-bounded 65536", limits.MaxStreamEventBytes)
	}
	if got := normalizeRequestTimeout(maxRequestTimeout + time.Second); got != DefaultRequestTimeout {
		t.Fatalf("normalized timeout = %s, want %s", got, DefaultRequestTimeout)
	}
}

func TestManagerRejectsOversizedInputBeforeProviderCall(t *testing.T) {
	provider := &namedFake{name: "fake"}
	manager := NewManager(Config{
		Enabled:         true,
		DefaultProvider: "fake",
		DefaultModel:    "provider-model",
	})
	manager.providers[provider.name] = provider

	_, err := manager.GenerateText(context.Background(), &TextRequest{
		Input: strings.Repeat("x", 2*1024*1024),
	})
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("GenerateText() error = %v, want bounded input error", err)
	}
	if provider.called {
		t.Fatal("provider was called for an oversized request")
	}
}

func TestManagerRejectsInvalidRequestMetadata(t *testing.T) {
	tests := []struct {
		name string
		req  *TextRequest
		want error
	}{
		{name: "nil context", req: &TextRequest{Input: "x"}, want: ErrContextRequired},
		{name: "invalid model", req: &TextRequest{Input: "x", Model: "model name"}, want: ErrInvalidModel},
		{name: "invalid reasoning", req: &TextRequest{Input: "x", ReasoningEffort: "very high"}, want: ErrInvalidReasoningEffort},
		{name: "invalid UTF-8", req: &TextRequest{Input: string([]byte{0xff})}, want: ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &namedFake{name: "fake"}
			manager := NewManager(Config{
				Enabled:         true,
				DefaultProvider: provider.name,
				DefaultModel:    "provider-model",
			})
			manager.providers[provider.name] = provider
			ctx := context.Background()
			var err error
			if tt.name == "nil context" {
				_, err = manager.GenerateText(nil, tt.req) //nolint:staticcheck // verifies defensive nil-context handling
			} else {
				_, err = manager.GenerateText(ctx, tt.req)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("GenerateText() error = %v, want %v", err, tt.want)
			}
			if provider.called {
				t.Fatal("provider was called for an invalid request")
			}
		})
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

type namedFake struct {
	name   string
	called bool
}

func (f *namedFake) Name() string { return f.name }
func (f *namedFake) GenerateText(context.Context, *TextRequest) (*TextResponse, error) {
	f.called = true
	return &TextResponse{Text: "ok"}, nil
}

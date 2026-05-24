// Package ai provides a provider-neutral AI capability surface.
//
// The package exposes two contracts:
//
//   - [Provider] — the minimum contract every provider must satisfy
//     (one-shot text generation).
//   - [StreamingProvider] — an OPTIONAL extension. Providers that can
//     deliver tokens incrementally implement it; callers ask the
//     [Manager] for a stream and get [ErrStreamingUnsupported] when the
//     selected provider only supports one-shot.
//
// Today the only built-in provider is OpenAI. Adding a new provider is
// a single file: implement [Provider] (and optionally [StreamingProvider])
// and register it in [NewManager].
package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// ProviderOpenAI is the built-in OpenAI Responses-API provider.
	ProviderOpenAI = "openai"
)

var (
	ErrDisabled             = errors.New("ai: capability is disabled")
	ErrInputRequired        = errors.New("ai: input is required")
	ErrModelRequired        = errors.New("ai: model is required")
	ErrProviderRequired     = errors.New("ai: provider is required")
	ErrProviderUnavailable  = errors.New("ai: provider is not configured")
	ErrEmptyResponseText    = errors.New("ai: provider returned empty text")
	ErrStreamingUnsupported = errors.New("ai: provider does not support streaming")
)

// ProviderConfig configures a concrete provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}

// Config defines the provider-neutral AI capability configuration.
//
// To add a new provider, extend this struct with a new ProviderConfig
// field and register the provider in NewManager.
type Config struct {
	Enabled         bool
	DefaultProvider string
	DefaultModel    string
	RequestTimeout  time.Duration
	OpenAI          ProviderConfig
}

// TextRequest is a provider-neutral text generation request.
type TextRequest struct {
	Provider        string
	Model           string
	Input           string
	Instructions    string
	ReasoningEffort string
}

// TextResponse is a provider-neutral text generation response.
type TextResponse struct {
	ID       string
	Provider string
	Model    string
	Text     string
}

// StreamChunk is one delta emitted by a streaming provider.
//
// A chunk carries either Delta text or a terminal Err. The channel is
// closed when the stream ends cleanly (provider sent a "done" signal).
type StreamChunk struct {
	Delta string
	Err   error
}

// Provider is the minimum contract for an AI provider.
type Provider interface {
	Name() string
	GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error)
}

// StreamingProvider is an optional extension implemented by providers
// that can stream tokens. Callers use [Manager.GenerateTextStream]
// rather than type-asserting providers directly.
type StreamingProvider interface {
	Provider
	GenerateTextStream(ctx context.Context, req *TextRequest) (<-chan StreamChunk, error)
}

// Manager routes requests to configured providers.
type Manager struct {
	enabled         bool
	defaultProvider string
	defaultModel    string
	providers       map[string]Provider
}

// NewManager creates a provider manager from AI capability config.
//
// Providers without an API key are skipped — `len(m.ProviderNames())`
// reports what's actually live.
func NewManager(cfg Config) *Manager {
	manager := &Manager{
		enabled:         cfg.Enabled,
		defaultProvider: normalizeProvider(cfg.DefaultProvider),
		defaultModel:    strings.TrimSpace(cfg.DefaultModel),
		providers:       make(map[string]Provider),
	}

	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	if strings.TrimSpace(cfg.OpenAI.APIKey) != "" {
		manager.providers[ProviderOpenAI] = NewOpenAIProvider(cfg.OpenAI, timeout)
	}

	return manager
}

// ProviderNames returns the configured provider names in no guaranteed order.
func (m *Manager) ProviderNames() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// GenerateText routes a request to the selected provider.
func (m *Manager) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	provider, normalized, err := m.resolve(req)
	if err != nil {
		return nil, err
	}
	return provider.GenerateText(ctx, normalized)
}

// GenerateTextStream routes a streaming request to the selected provider.
//
// Returns [ErrStreamingUnsupported] if the resolved provider does not
// implement [StreamingProvider]. The returned channel is closed when the
// stream ends or fails — see [StreamChunk] for the per-message contract.
func (m *Manager) GenerateTextStream(ctx context.Context, req *TextRequest) (<-chan StreamChunk, error) {
	provider, normalized, err := m.resolve(req)
	if err != nil {
		return nil, err
	}
	streamer, ok := provider.(StreamingProvider)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrStreamingUnsupported, provider.Name())
	}
	return streamer.GenerateTextStream(ctx, normalized)
}

// resolve validates the request, applies defaults, and returns the
// chosen provider. Shared by GenerateText and GenerateTextStream.
func (m *Manager) resolve(req *TextRequest) (Provider, *TextRequest, error) {
	if !m.enabled {
		return nil, nil, ErrDisabled
	}
	if req == nil {
		return nil, nil, ErrInputRequired
	}

	providerName := normalizeProvider(req.Provider)
	if providerName == "" {
		providerName = m.defaultProvider
	}
	if providerName == "" {
		return nil, nil, ErrProviderRequired
	}

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, providerName)
	}

	normalized := *req
	normalized.Provider = providerName
	normalized.Input = strings.TrimSpace(normalized.Input)
	normalized.Instructions = strings.TrimSpace(normalized.Instructions)
	normalized.ReasoningEffort = strings.TrimSpace(normalized.ReasoningEffort)
	if normalized.Input == "" {
		return nil, nil, ErrInputRequired
	}
	if strings.TrimSpace(normalized.Model) == "" {
		normalized.Model = m.defaultModel
	}
	if normalized.Model == "" {
		return nil, nil, ErrModelRequired
	}

	return provider, &normalized, nil
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

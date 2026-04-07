// Package ai provides provider-agnostic AI capabilities.
// It offers a small text-generation surface that business modules and CLI
// commands can use without depending on provider-specific request shapes.
package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ProviderOpenAI = "openai"
)

var (
	ErrDisabled            = errors.New("ai: capability is disabled")
	ErrInputRequired       = errors.New("ai: input is required")
	ErrModelRequired       = errors.New("ai: model is required")
	ErrProviderRequired    = errors.New("ai: provider is required")
	ErrProviderUnavailable = errors.New("ai: provider is not configured")
	ErrEmptyResponseText   = errors.New("ai: provider returned empty text")
)

// ProviderConfig configures a concrete provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}

// Config defines the provider-neutral AI capability configuration.
type Config struct {
	Enabled         bool
	DefaultProvider string
	DefaultModel    string
	RequestTimeout  time.Duration
	OpenAI          ProviderConfig
	Anthropic       ProviderConfig
	Gemini          ProviderConfig
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

// Provider defines the minimal provider contract used by the scaffold.
type Provider interface {
	Name() string
	GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error)
}

// Manager routes requests to configured providers.
type Manager struct {
	enabled         bool
	defaultProvider string
	defaultModel    string
	providers       map[string]Provider
}

// NewManager creates a provider manager from AI capability config.
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
	if !m.enabled {
		return nil, ErrDisabled
	}
	if req == nil {
		return nil, ErrInputRequired
	}

	providerName := normalizeProvider(req.Provider)
	if providerName == "" {
		providerName = m.defaultProvider
	}
	if providerName == "" {
		return nil, ErrProviderRequired
	}

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, providerName)
	}

	normalized := *req
	normalized.Provider = providerName
	normalized.Input = strings.TrimSpace(normalized.Input)
	normalized.Instructions = strings.TrimSpace(normalized.Instructions)
	normalized.ReasoningEffort = strings.TrimSpace(normalized.ReasoningEffort)
	if normalized.Input == "" {
		return nil, ErrInputRequired
	}
	if strings.TrimSpace(normalized.Model) == "" {
		normalized.Model = m.defaultModel
	}
	if normalized.Model == "" {
		return nil, ErrModelRequired
	}

	return provider.GenerateText(ctx, &normalized)
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

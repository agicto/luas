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
// Today the only built-in provider is OpenAI. Adding another provider requires
// an adapter, typed configuration and validation, and registration in
// [NewManager]; see api/docs/AI.md for the complete boundary.
package ai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// ProviderOpenAI is the built-in OpenAI Responses-API provider.
	ProviderOpenAI = "openai"

	// DefaultRequestTimeout bounds both one-shot calls and complete streaming sessions.
	DefaultRequestTimeout = 120 * time.Second
	// DefaultMaxInputBytes bounds input plus instructions before provider serialization.
	DefaultMaxInputBytes = 1024 * 1024
	// DefaultMaxResponseBytes bounds a decompressed one-shot provider response.
	DefaultMaxResponseBytes int64 = 4 * 1024 * 1024
	// DefaultMaxStreamEventBytes bounds one SSE line from a streaming provider.
	DefaultMaxStreamEventBytes = 1024 * 1024

	minRequestLimitBytes     = 1024
	maxRequestTimeout        = 15 * time.Minute
	maxInputLimitBytes       = 16 * 1024 * 1024
	maxResponseLimitBytes    = 32 * 1024 * 1024
	maxStreamEventLimitBytes = 4 * 1024 * 1024
	maxProviderNameBytes     = 64
	maxModelNameBytes        = 256
	maxReasoningValueBytes   = 64
)

var (
	ErrDisabled                 = errors.New("ai: capability is disabled")
	ErrContextRequired          = errors.New("ai: context is required")
	ErrInputRequired            = errors.New("ai: input is required")
	ErrInvalidInput             = errors.New("ai: input and instructions must be valid UTF-8")
	ErrInputTooLarge            = errors.New("ai: input exceeds limit")
	ErrModelRequired            = errors.New("ai: model is required")
	ErrInvalidModel             = errors.New("ai: model is invalid")
	ErrProviderRequired         = errors.New("ai: provider is required")
	ErrInvalidProvider          = errors.New("ai: provider is invalid")
	ErrProviderNotConfigured    = errors.New("ai: provider is not configured")
	ErrInvalidReasoningEffort   = errors.New("ai: reasoning effort is invalid")
	ErrProviderRequestFailed    = errors.New("ai: provider request failed")
	ErrProviderResponseTooLarge = errors.New("ai: provider response exceeds limit")
	ErrInvalidProviderResponse  = errors.New("ai: provider returned an invalid response")
	ErrEmptyResponseText        = errors.New("ai: provider returned empty text")
	ErrStreamingUnsupported     = errors.New("ai: provider does not support streaming")
)

// ProviderError reports a provider failure without retaining or exposing its response body.
type ProviderError struct {
	Provider   string
	StatusCode int
	Retryable  bool
}

func (e *ProviderError) Error() string {
	provider := "configured"
	if e != nil && e.Provider != "" {
		provider = e.Provider
	}
	if e != nil && e.StatusCode > 0 {
		return fmt.Sprintf("ai: %s provider returned HTTP status %d", provider, e.StatusCode)
	}
	return fmt.Sprintf("ai: %s provider request failed", provider)
}

// Unwrap gives callers a stable category while ProviderError retains safe status metadata.
func (e *ProviderError) Unwrap() error {
	return ErrProviderRequestFailed
}

// ProviderConfig configures a concrete provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}

// RequestLimits are provider-neutral memory and protocol boundaries.
type RequestLimits struct {
	MaxInputBytes       int
	MaxResponseBytes    int64
	MaxStreamEventBytes int
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
	Limits          RequestLimits
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
	ProviderResponseID string
	Provider           string
	Model              string
	Text               string
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
	limits          RequestLimits
	providers       map[string]Provider
}

// NewManager creates a provider manager from AI capability config.
//
// Providers without an API key are skipped — `len(m.ProviderNames())`
// reports what's actually live.
func NewManager(cfg Config) *Manager {
	limits := normalizeRequestLimits(cfg.Limits)
	defaultProvider := normalizeProvider(cfg.DefaultProvider)
	if defaultProvider != "" && !validOpaqueName(defaultProvider, maxProviderNameBytes) {
		defaultProvider = ""
	}
	manager := &Manager{
		enabled:         cfg.Enabled,
		defaultProvider: defaultProvider,
		defaultModel:    strings.TrimSpace(cfg.DefaultModel),
		limits:          limits,
		providers:       make(map[string]Provider),
	}

	timeout := normalizeRequestTimeout(cfg.RequestTimeout)

	if strings.TrimSpace(cfg.OpenAI.APIKey) != "" {
		manager.providers[ProviderOpenAI] = NewOpenAIProvider(cfg.OpenAI, timeout, limits)
	}

	return manager
}

// ProviderNames returns configured provider names in stable lexical order.
func (m *Manager) ProviderNames() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GenerateText routes a request to the selected provider.
func (m *Manager) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	if ctx == nil {
		return nil, ErrContextRequired
	}
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
	if ctx == nil {
		return nil, ErrContextRequired
	}
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

	if strings.TrimSpace(req.Provider) != "" && !validOpaqueName(strings.TrimSpace(req.Provider), maxProviderNameBytes) {
		return nil, nil, ErrInvalidProvider
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
		return nil, nil, fmt.Errorf("%w: %s", ErrProviderNotConfigured, providerName)
	}

	normalized := *req
	normalized.Provider = providerName
	if inputExceedsLimit(&normalized, m.limits.MaxInputBytes) {
		return nil, nil, ErrInputTooLarge
	}
	normalizeTextRequest(&normalized)
	if normalized.Model == "" {
		normalized.Model = m.defaultModel
	}
	if err := validateTextRequest(&normalized, m.limits.MaxInputBytes); err != nil {
		return nil, nil, err
	}

	return provider, &normalized, nil
}

func normalizeRequestLimits(limits RequestLimits) RequestLimits {
	if limits.MaxInputBytes < minRequestLimitBytes || limits.MaxInputBytes > maxInputLimitBytes {
		limits.MaxInputBytes = DefaultMaxInputBytes
	}
	if limits.MaxResponseBytes < minRequestLimitBytes || limits.MaxResponseBytes > maxResponseLimitBytes {
		limits.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if limits.MaxStreamEventBytes < minRequestLimitBytes ||
		limits.MaxStreamEventBytes > maxStreamEventLimitBytes ||
		int64(limits.MaxStreamEventBytes) > limits.MaxResponseBytes {
		limits.MaxStreamEventBytes = min(DefaultMaxStreamEventBytes, int(limits.MaxResponseBytes))
	}
	return limits
}

func normalizeRequestTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 || timeout > maxRequestTimeout {
		return DefaultRequestTimeout
	}
	return timeout
}

func normalizeTextRequest(req *TextRequest) {
	req.Model = strings.TrimSpace(req.Model)
	req.Input = strings.TrimSpace(req.Input)
	req.Instructions = strings.TrimSpace(req.Instructions)
	req.ReasoningEffort = strings.TrimSpace(req.ReasoningEffort)
}

func inputExceedsLimit(req *TextRequest, maxInputBytes int) bool {
	if req == nil {
		return false
	}
	if maxInputBytes <= 0 {
		maxInputBytes = DefaultMaxInputBytes
	}
	return len(req.Input) > maxInputBytes || len(req.Instructions) > maxInputBytes-len(req.Input)
}

func validateTextRequest(req *TextRequest, maxInputBytes int) error {
	if req == nil {
		return ErrInputRequired
	}
	if maxInputBytes <= 0 {
		maxInputBytes = DefaultMaxInputBytes
	}
	if inputExceedsLimit(req, maxInputBytes) {
		return ErrInputTooLarge
	}
	if !utf8.ValidString(req.Input) || !utf8.ValidString(req.Instructions) {
		return ErrInvalidInput
	}
	if req.Input == "" {
		return ErrInputRequired
	}
	if req.Model == "" {
		return ErrModelRequired
	}
	if !validOpaqueName(req.Model, maxModelNameBytes) {
		return ErrInvalidModel
	}
	if req.ReasoningEffort != "" && !validOpaqueName(req.ReasoningEffort, maxReasoningValueBytes) {
		return ErrInvalidReasoningEffort
	}
	return nil
}

func validOpaqueName(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if unicode.IsSpace(char) || unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

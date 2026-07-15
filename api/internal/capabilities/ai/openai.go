package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxProviderResponseHeaderBytes = 64 * 1024
	maxProviderErrorDrainBytes     = 64 * 1024
)

var errProviderRedirectBlocked = fmt.Errorf("%w: redirect blocked", ErrProviderRequestFailed)

// OpenAIProvider implements text generation with the OpenAI Responses API.
//
// Implements both [Provider] (one-shot) and [StreamingProvider] (SSE).
type OpenAIProvider struct {
	apiKey   string
	endpoint string
	timeout  time.Duration
	limits   RequestLimits
	client   *http.Client
}

// NewOpenAIProvider creates a new OpenAI-backed provider.
func NewOpenAIProvider(cfg ProviderConfig, timeout time.Duration, limits RequestLimits) *OpenAIProvider {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	timeout = normalizeRequestTimeout(timeout)
	limits = normalizeRequestLimits(limits)

	return &OpenAIProvider{
		apiKey:   strings.TrimSpace(cfg.APIKey),
		endpoint: providerEndpoint(baseURL),
		timeout:  timeout,
		limits:   limits,
		client:   newProviderHTTPClient(timeout),
	}
}

func providerEndpoint(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	return strings.TrimRight(parsed.String(), "/") + "/responses"
}

func newProviderHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   min(timeout, 10*time.Second),
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:      true,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    10,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    min(timeout, 10*time.Second),
		ResponseHeaderTimeout:  timeout,
		ExpectContinueTimeout:  time.Second,
		MaxResponseHeaderBytes: maxProviderResponseHeaderBytes,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errProviderRedirectBlocked
		},
	}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return ProviderOpenAI
}

// GenerateText calls the OpenAI Responses API and aggregates output_text items.
func (p *OpenAIProvider) GenerateText(ctx context.Context, req *TextRequest) (*TextResponse, error) {
	normalized, err := p.prepareRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	body := p.requestBody(normalized, false)
	requestPayload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: encode request: %v", ErrProviderRequestFailed, err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, p.endpoint, bytes.NewReader(requestPayload))
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrProviderRequestFailed, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		if reqCtx.Err() != nil {
			return nil, providerRequestError(reqCtx.Err())
		}
		return nil, providerRequestError(err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		drainProviderErrorBody(httpResp.Body)
		return nil, newProviderStatusError(httpResp.StatusCode)
	}
	respBody, err := readProviderBody(httpResp.Body, p.limits.MaxResponseBytes)
	if err != nil {
		return nil, err
	}

	var responsePayload openAIResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &responsePayload); err != nil {
			return nil, fmt.Errorf("%w: decode JSON: %v", ErrInvalidProviderResponse, err)
		}
	}

	text := responsePayload.outputText()
	if text == "" {
		return nil, ErrEmptyResponseText
	}

	return &TextResponse{
		ProviderResponseID: responsePayload.ID,
		Provider:           ProviderOpenAI,
		Model:              responsePayload.Model,
		Text:               text,
	}, nil
}

// GenerateTextStream calls the OpenAI Responses API with stream=true and
// returns a channel of [StreamChunk]. The channel is closed when the
// stream ends or fails.
//
// SSE wire format: each event is a `data: {...}` line. We forward
// `response.output_text.delta` events as Delta chunks and treat
// `response.completed`, provider failures, and `[DONE]` as terminal.
func (p *OpenAIProvider) GenerateTextStream(ctx context.Context, req *TextRequest) (<-chan StreamChunk, error) {
	normalized, err := p.prepareRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	body := p.requestBody(normalized, true)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: encode stream request: %v", ErrProviderRequestFailed, err)
	}

	streamCtx, cancel := context.WithTimeout(ctx, p.timeout)
	httpReq, err := http.NewRequestWithContext(streamCtx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%w: build stream request: %v", ErrProviderRequestFailed, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// bodyclose cannot track this ownership transfer: successful response
	// bodies are closed by the stream goroutine; early-return paths close here.
	httpResp, err := p.client.Do(httpReq) //nolint:bodyclose // closed in goroutine or in error paths above
	if err != nil {
		ctxErr := streamCtx.Err()
		cancel()
		if ctxErr != nil {
			return nil, providerRequestError(ctxErr)
		}
		return nil, providerRequestError(err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		defer cancel()
		defer httpResp.Body.Close()
		drainProviderErrorBody(httpResp.Body)
		return nil, newProviderStatusError(httpResp.StatusCode)
	}

	out := make(chan StreamChunk, 16)
	go func() {
		defer httpResp.Body.Close()
		defer cancel()
		defer close(out)

		scanner := bufio.NewScanner(httpResp.Body)
		initialBuffer := min(64*1024, p.limits.MaxStreamEventBytes)
		scanner.Buffer(make([]byte, 0, initialBuffer), p.limits.MaxStreamEventBytes)

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
				sendOrAbort(streamCtx, out, StreamChunk{Err: ErrInvalidProviderResponse})
				return
			}

			switch ev.Type {
			case "response.output_text.delta":
				if ev.Delta != "" {
					if !sendOrAbort(streamCtx, out, StreamChunk{Delta: ev.Delta}) {
						sendTerminal(out, providerRequestError(streamCtx.Err()))
						return
					}
				}
			case "response.completed":
				return
			case "response.failed", "response.incomplete", "response.error", "error":
				sendOrAbort(streamCtx, out, StreamChunk{Err: &ProviderError{Provider: ProviderOpenAI}})
				return
			}
		}

		if streamCtx.Err() != nil {
			sendTerminal(out, providerRequestError(streamCtx.Err()))
			return
		}
		if scanner.Err() != nil {
			sendOrAbort(streamCtx, out, StreamChunk{Err: ErrInvalidProviderResponse})
			return
		}
		sendOrAbort(streamCtx, out, StreamChunk{Err: ErrInvalidProviderResponse})
	}()

	return out, nil
}

// requestBody assembles the JSON body shared by streaming and non-streaming
// calls. `stream` toggles the SSE behavior on the Responses API.
func (p *OpenAIProvider) requestBody(req *TextRequest, stream bool) openAIRequest {
	body := openAIRequest{
		Model:        req.Model,
		Input:        req.Input,
		Instructions: req.Instructions,
		Stream:       stream,
	}
	if req.ReasoningEffort != "" {
		body.Reasoning = &openAIReasoning{Effort: req.ReasoningEffort}
	}
	return body
}

func (p *OpenAIProvider) prepareRequest(ctx context.Context, req *TextRequest) (*TextRequest, error) {
	if ctx == nil {
		return nil, ErrContextRequired
	}
	if p == nil || p.apiKey == "" || p.endpoint == "" || p.client == nil {
		return nil, ErrProviderNotConfigured
	}
	if req == nil {
		return nil, ErrInputRequired
	}
	normalized := *req
	if inputExceedsLimit(&normalized, p.limits.MaxInputBytes) {
		return nil, ErrInputTooLarge
	}
	normalizeTextRequest(&normalized)
	if err := validateTextRequest(&normalized, p.limits.MaxInputBytes); err != nil {
		return nil, err
	}
	return &normalized, nil
}

func readProviderBody(body io.Reader, maxBytes int64) ([]byte, error) {
	responseBody, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %v", ErrProviderRequestFailed, err)
	}
	if int64(len(responseBody)) > maxBytes {
		return nil, ErrProviderResponseTooLarge
	}
	return responseBody, nil
}

func drainProviderErrorBody(body io.Reader) {
	_, err := io.Copy(io.Discard, io.LimitReader(body, maxProviderErrorDrainBytes))
	if err != nil {
		return
	}
}

func providerRequestError(cause error) error {
	if cause == nil {
		return ErrProviderRequestFailed
	}
	return fmt.Errorf("%w: %w", ErrProviderRequestFailed, cause)
}

func newProviderStatusError(statusCode int) *ProviderError {
	return &ProviderError{
		Provider:   ProviderOpenAI,
		StatusCode: statusCode,
		Retryable:  statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooEarly || statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError,
	}
}

func sendOrAbort(ctx context.Context, out chan<- StreamChunk, chunk StreamChunk) bool {
	select {
	case out <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}

func sendTerminal(out chan<- StreamChunk, err error) {
	if err == nil {
		return
	}
	select {
	case out <- StreamChunk{Err: err}:
	default:
	}
}

type openAIRequest struct {
	Model        string           `json:"model"`
	Input        string           `json:"input"`
	Instructions string           `json:"instructions,omitempty"`
	Reasoning    *openAIReasoning `json:"reasoning,omitempty"`
	Stream       bool             `json:"stream,omitempty"`
}

type openAIReasoning struct {
	Effort string `json:"effort"`
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
}

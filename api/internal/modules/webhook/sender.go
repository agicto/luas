package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	capabilitycrypto "github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

const webhookUserAgent = "Luas-Webhooks/1.0"

type outboundWebhook struct {
	URL             string
	MessageID       string
	EventType       string
	OccurredAt      time.Time
	PayloadJSON     string
	CurrentSecret   []byte
	PreviousSecret  []byte
	PreviousExpires *time.Time
}

type sendResult struct {
	HTTPStatus        *int
	FailureCode       string
	Retryable         bool
	ResponseTruncated bool
	Duration          time.Duration
}

type webhookSender interface {
	Send(context.Context, outboundWebhook) (sendResult, error)
}

type sender struct {
	policy           *targetPolicy
	timeout          time.Duration
	maxResponseBytes int64
	now              func() time.Time
}

// NewSender creates the bounded Standard Webhooks transport adapter.
func NewSender(cfg *config.Config, policy *targetPolicy) *sender {
	value := &sender{policy: policy, now: time.Now}
	if cfg != nil {
		value.timeout = cfg.Webhook.RequestTimeout
		value.maxResponseBytes = cfg.Webhook.MaxResponseBytes
	}
	return value
}

func (s *sender) Send(ctx context.Context, outbound outboundWebhook) (sendResult, error) {
	result := sendResult{}
	if s == nil || s.policy == nil || s.timeout <= 0 || s.maxResponseBytes <= 0 || s.now == nil {
		return result, domain.ErrServiceUnavailable
	}
	started := s.now().UTC()
	_, parsed, err := s.policy.normalizeSyntax(outbound.URL)
	if err != nil {
		result.FailureCode = "WEBHOOK.INVALID_TARGET"
		return result, nil
	}
	body, err := encodeWebhookBody(outbound)
	if err != nil {
		result.FailureCode = "WEBHOOK.INVALID_MESSAGE"
		return result, nil
	}
	timestamp := strconv.FormatInt(started.Unix(), 10)
	signature, err := webhookSignatures(outbound, timestamp, body, started)
	if err != nil {
		return result, err
	}

	requestContext, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return result, fmt.Errorf("create webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", webhookUserAgent)
	request.Header.Set("webhook-id", outbound.MessageID)
	request.Header.Set("webhook-timestamp", timestamp)
	request.Header.Set("webhook-signature", signature)

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           s.policy.DialContext,
		DisableKeepAlives:     true,
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   min(s.timeout, 10*time.Second),
		ResponseHeaderTimeout: s.timeout,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: parsed.Hostname(),
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   s.timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errWebhookRedirectBlocked
		},
	}
	response, requestErr := client.Do(request)
	if requestErr != nil {
		result.Duration = webhookElapsed(s.now().UTC(), started)
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		result.FailureCode, result.Retryable = classifyWebhookNetworkError(requestErr)
		return result, nil
	}
	defer response.Body.Close()
	status := response.StatusCode
	result.HTTPStatus = &status
	read, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, s.maxResponseBytes+1))
	result.Duration = webhookElapsed(s.now().UTC(), started)
	result.ResponseTruncated = read > s.maxResponseBytes
	if readErr != nil {
		result.FailureCode, result.Retryable = classifyWebhookNetworkError(readErr)
		return result, nil
	}
	switch {
	case status >= 200 && status < 300:
		return result, nil
	case status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests || status >= 500:
		result.FailureCode = webhookHTTPFailureCode(status)
		result.Retryable = true
	default:
		result.FailureCode = webhookHTTPFailureCode(status)
	}
	return result, nil
}

func webhookElapsed(completed time.Time, started time.Time) time.Duration {
	duration := completed.Sub(started)
	if duration < 0 {
		return 0
	}
	return duration
}

func encodeWebhookBody(outbound outboundWebhook) ([]byte, error) {
	if outbound.MessageID == "" || !validWebhookEventType(outbound.EventType) || outbound.OccurredAt.IsZero() ||
		len(outbound.PayloadJSON) < 2 || len(outbound.CurrentSecret) != 32 {
		return nil, domain.ErrInvalidInput
	}
	var data json.RawMessage = []byte(outbound.PayloadJSON)
	return json.Marshal(struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Timestamp time.Time       `json:"timestamp"`
		Data      json.RawMessage `json:"data"`
	}{
		ID:        outbound.MessageID,
		Type:      outbound.EventType,
		Timestamp: outbound.OccurredAt.UTC(),
		Data:      data,
	})
}

func webhookSignatures(outbound outboundWebhook, timestamp string, body []byte, now time.Time) (string, error) {
	if len(outbound.CurrentSecret) != 32 {
		return "", domain.ErrServiceUnavailable
	}
	signed := make([]byte, 0, len(outbound.MessageID)+len(timestamp)+len(body)+2)
	signed = append(signed, outbound.MessageID...)
	signed = append(signed, '.')
	signed = append(signed, timestamp...)
	signed = append(signed, '.')
	signed = append(signed, body...)
	values := []string{"v1," + base64.StdEncoding.EncodeToString(capabilitycrypto.HMACSHA256(signed, outbound.CurrentSecret))}
	if len(outbound.PreviousSecret) == 32 && outbound.PreviousExpires != nil && now.Before(*outbound.PreviousExpires) {
		values = append(values, "v1,"+base64.StdEncoding.EncodeToString(capabilitycrypto.HMACSHA256(signed, outbound.PreviousSecret)))
	}
	return strings.Join(values, " "), nil
}

func classifyWebhookNetworkError(err error) (string, bool) {
	switch {
	case errors.Is(err, errWebhookRedirectBlocked):
		return "WEBHOOK.REDIRECT_BLOCKED", false
	case errors.Is(err, domain.ErrWebhookInvalidTarget):
		return "WEBHOOK.INVALID_TARGET", false
	case errors.Is(err, context.DeadlineExceeded):
		return "WEBHOOK.TIMEOUT", true
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return "WEBHOOK.DNS_FAILURE", true
	}
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Timeout() {
		return "WEBHOOK.TIMEOUT", true
	}
	return "WEBHOOK.NETWORK_FAILURE", true
}

func webhookHTTPFailureCode(status int) string {
	return fmt.Sprintf("WEBHOOK.HTTP_%d", status)
}

var errWebhookRedirectBlocked = errors.New("webhook redirects are disabled")

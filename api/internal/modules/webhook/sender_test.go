package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	capabilitycrypto "github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestSenderSignsTheExactBodyAndHeaders(t *testing.T) {
	requestChannel := make(chan capturedWebhookRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		requestChannel <- capturedWebhookRequest{Header: request.Header.Clone(), Body: body}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	sender := newLocalWebhookSender(2*time.Second, 1024)
	sender.now = func() time.Time { return now }
	secret := bytes.Repeat([]byte{0x42}, 32)
	result, err := sender.Send(context.Background(), outboundWebhook{
		URL:           server.URL + "/webhook",
		MessageID:     "msg_0123456789abcdef",
		EventType:     "webhook.test",
		OccurredAt:    now.Add(-time.Minute),
		PayloadJSON:   `{"endpoint_id":9,"organization_id":42}`,
		CurrentSecret: secret,
	})
	require.NoError(t, err)
	assert.Empty(t, result.FailureCode)
	assert.Equal(t, http.StatusNoContent, *result.HTTPStatus)

	captured := <-requestChannel
	assert.Equal(t, "msg_0123456789abcdef", captured.Header.Get("webhook-id"))
	assert.Equal(t, strconv.FormatInt(now.Unix(), 10), captured.Header.Get("webhook-timestamp"))
	assert.Equal(t, webhookUserAgent, captured.Header.Get("User-Agent"))
	assert.Equal(t, "application/json", captured.Header.Get("Content-Type"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(captured.Body, &body))
	assert.Equal(t, "webhook.test", body["type"])

	signed := []byte("msg_0123456789abcdef." + strconv.FormatInt(now.Unix(), 10) + "." + string(captured.Body))
	expected := "v1," + base64.StdEncoding.EncodeToString(capabilitycrypto.HMACSHA256(signed, secret))
	assert.Equal(t, expected, captured.Header.Get("webhook-signature"))
}

func TestSenderDoesNotFollowRedirects(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		destinationCalls.Add(1)
	}))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()

	sender := newLocalWebhookSender(time.Second, 1024)
	result, err := sender.Send(context.Background(), testOutboundWebhook(redirect.URL))
	require.NoError(t, err)
	assert.Equal(t, "WEBHOOK.REDIRECT_BLOCKED", result.FailureCode)
	assert.False(t, result.Retryable)
	assert.Zero(t, destinationCalls.Load())
}

func TestSenderBoundsResponsesAndClassifiesRetryableFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusTooManyRequests)
		_, _ = response.Write(bytes.Repeat([]byte("x"), 128))
	}))
	defer server.Close()

	sender := newLocalWebhookSender(time.Second, 32)
	result, err := sender.Send(context.Background(), testOutboundWebhook(server.URL))
	require.NoError(t, err)
	assert.Equal(t, "WEBHOOK.HTTP_429", result.FailureCode)
	assert.True(t, result.Retryable)
	assert.True(t, result.ResponseTruncated)
}

func TestSenderClassifiesIncompleteResponseBodyAsRetryableNetworkFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Length", "10")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("short"))
	}))
	defer server.Close()

	sender := newLocalWebhookSender(time.Second, 1024)
	result, err := sender.Send(context.Background(), testOutboundWebhook(server.URL))
	require.NoError(t, err)
	require.NotNil(t, result.HTTPStatus)
	assert.Equal(t, http.StatusOK, *result.HTTPStatus)
	assert.Equal(t, "WEBHOOK.NETWORK_FAILURE", result.FailureCode)
	assert.True(t, result.Retryable)
}

func TestSenderDurationIncludesResponseBodyDrain(t *testing.T) {
	delay := 80 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Length", "2")
		response.WriteHeader(http.StatusOK)
		response.(http.Flusher).Flush()
		time.Sleep(delay)
		_, _ = response.Write([]byte("{}"))
	}))
	defer server.Close()

	sender := newLocalWebhookSender(time.Second, 1024)
	result, err := sender.Send(context.Background(), testOutboundWebhook(server.URL))
	require.NoError(t, err)
	assert.Empty(t, result.FailureCode)
	assert.GreaterOrEqual(t, result.Duration, delay/2)
}

func TestSenderClassifiesTimeoutWithoutPersistingProviderText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newLocalWebhookSender(20*time.Millisecond, 1024)
	result, err := sender.Send(context.Background(), testOutboundWebhook(server.URL))
	require.NoError(t, err)
	assert.Equal(t, "WEBHOOK.TIMEOUT", result.FailureCode)
	assert.True(t, result.Retryable)
}

type capturedWebhookRequest struct {
	Header http.Header
	Body   []byte
}

func newLocalWebhookSender(timeout time.Duration, maxResponse int64) *sender {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			RequestTimeout:      timeout,
			MaxResponseBytes:    maxResponse,
			AllowInsecureHTTP:   true,
			AllowPrivateTargets: true,
		},
	}
	policy := NewTargetPolicy(cfg)
	return NewSender(cfg, policy)
}

func testOutboundWebhook(target string) outboundWebhook {
	return outboundWebhook{
		URL:           target,
		MessageID:     "msg_test",
		EventType:     "webhook.test",
		OccurredAt:    time.Now().UTC(),
		PayloadJSON:   `{"endpoint_id":9,"organization_id":42}`,
		CurrentSecret: bytes.Repeat([]byte{0x11}, 32),
	}
}

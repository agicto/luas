package email

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func configuredEmailService(client httpDoer, endpoint string, timeout time.Duration) *Service {
	return newService(config.EmailConfig{
		From:           "Luas <noreply@example.com>",
		ResendAPIKey:   "resend-test-key",
		RequestTimeout: timeout,
	}, client, endpoint)
}

func validRecipients(count int) []string {
	recipients := make([]string, count)
	for index := range recipients {
		recipients[index] = "alice@example.com"
	}
	return recipients
}

func TestServiceSendEmailUsesDocumentedProviderContract(t *testing.T) {
	type capturedRequest struct {
		authorization string
		contentType   string
		method        string
		path          string
		payload       resendSendRequest
	}
	captured := make(chan capturedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload resendSendRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		captured <- capturedRequest{
			authorization: r.Header.Get("Authorization"),
			contentType:   r.Header.Get("Content-Type"),
			method:        r.Method,
			path:          r.URL.Path,
			payload:       payload,
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL+"/emails", time.Second)
	err := service.SendEmail(context.Background(), []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	if err != nil {
		t.Fatalf("SendEmail() error = %v", err)
	}

	request := <-captured
	if request.authorization != "Bearer resend-test-key" {
		t.Fatalf("Authorization = %q", request.authorization)
	}
	if request.contentType != "application/json" {
		t.Fatalf("Content-Type = %q", request.contentType)
	}
	if request.method != http.MethodPost || request.path != "/emails" {
		t.Fatalf("request target = %s %s, want POST /emails", request.method, request.path)
	}
	if request.payload.From != "Luas <noreply@example.com>" {
		t.Fatalf("From = %q", request.payload.From)
	}
	if len(request.payload.To) != 1 || request.payload.To[0] != "alice@example.com" {
		t.Fatalf("To = %v", request.payload.To)
	}
	if request.payload.Subject != "Subject" || request.payload.HTML != "<p>Hello</p>" {
		t.Fatalf("message = subject %q, HTML %q", request.payload.Subject, request.payload.HTML)
	}
}

func TestServiceSendEmailIdempotentForwardsStableProviderKey(t *testing.T) {
	keys := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys <- r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	err := service.SendEmailIdempotent(
		context.Background(),
		[]string{"alice@example.com"},
		"Subject",
		"<p>Hello</p>",
		"notification-email-42",
	)
	requireNoEmailError(t, err)
	if key := <-keys; key != "notification-email-42" {
		t.Fatalf("Idempotency-Key = %q", key)
	}
}

func TestServiceSendEmailIdempotentRejectsUnsafeKeyBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	err := service.SendEmailIdempotent(
		context.Background(),
		[]string{"alice@example.com"},
		"Subject",
		"<p>Hello</p>",
		"bad\nkey",
	)
	if !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("SendEmailIdempotent() error = %v, want ErrInvalidMessage", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0", calls.Load())
	}
}

func requireNoEmailError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("email send error = %v", err)
	}
}

func TestServiceSendEmailRejectsUnconfiguredCapability(t *testing.T) {
	service := NewService(&config.Config{})
	err := service.SendEmail(context.Background(), []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("SendEmail() error = %v, want ErrNotConfigured", err)
	}
}

func TestServiceSendEmailRejectsInvalidMessageBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)
	service := configuredEmailService(server.Client(), server.URL, time.Second)

	tests := []struct {
		name    string
		to      []string
		subject string
		html    string
	}{
		{name: "no recipients", subject: "Subject", html: "<p>Hello</p>"},
		{name: "too many recipients", to: validRecipients(maxResendRecipients + 1), subject: "Subject", html: "<p>Hello</p>"},
		{name: "invalid recipient", to: []string{"not-an-email"}, subject: "Subject", html: "<p>Hello</p>"},
		{name: "empty subject", to: []string{"alice@example.com"}, html: "<p>Hello</p>"},
		{name: "empty HTML", to: []string{"alice@example.com"}, subject: "Subject"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SendEmail(context.Background(), tt.to, tt.subject, tt.html)
			if !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("SendEmail() error = %v, want ErrInvalidMessage", err)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0", calls.Load())
	}
}

func TestServiceSendEmailHonorsCallerCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := service.SendEmail(ctx, []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendEmail() error = %v, want context.Canceled", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0 for pre-canceled context", calls.Load())
	}
}

func TestServiceSendEmailEnforcesRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, 20*time.Millisecond)
	startedAt := time.Now()
	err := service.SendEmail(context.Background(), []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	elapsed := time.Since(startedAt)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendEmail() error = %v, want context deadline", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("SendEmail() elapsed = %s, want bounded cancellation", elapsed)
	}
}

func TestServiceSendEmailDoesNotExposeProviderBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "alice@example.com provider-secret")
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	err := service.SendEmail(context.Background(), []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("SendEmail() error = %v, want ProviderError(500)", err)
	}
	if strings.Contains(err.Error(), "alice@example.com") || strings.Contains(err.Error(), "provider-secret") {
		t.Fatalf("SendEmail() leaked provider response: %v", err)
	}
}

func TestServiceSendEmailBoundsProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, strings.Repeat("x", maxProviderResponseBytes+1))
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	err := service.SendEmail(context.Background(), []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	if !errors.Is(err, ErrProviderResponseTooLarge) {
		t.Fatalf("SendEmail() error = %v, want ErrProviderResponseTooLarge", err)
	}
}

func TestServiceSendEmailRejectsSuccessfulResponseWithoutMessageID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	err := service.SendEmail(context.Background(), []string{"alice@example.com"}, "Subject", "<p>Hello</p>")
	if !errors.Is(err, ErrInvalidProviderResponse) {
		t.Fatalf("SendEmail() error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestServiceWelcomeEmailEscapesTemplateValues(t *testing.T) {
	payloads := make(chan resendSendRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload resendSendRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		payloads <- payload
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"email_123"}`)
	}))
	t.Cleanup(server.Close)

	service := configuredEmailService(server.Client(), server.URL, time.Second)
	err := service.SendWelcomeEmail(context.Background(), "alice@example.com", `<script>alert("x")</script>`)
	if err != nil {
		t.Fatalf("SendWelcomeEmail() error = %v", err)
	}

	html := (<-payloads).HTML
	if strings.Contains(html, "<script>") {
		t.Fatalf("welcome HTML contains an unescaped script: %s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("welcome HTML does not contain escaped username: %s", html)
	}
}

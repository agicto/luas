// Package email provides the Resend-backed outbound email capability.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
)

const (
	defaultResendEndpoint    = "https://api.resend.com/emails"
	maxResendRecipients      = 50
	maxProviderResponseBytes = 64 * 1024
)

var (
	// ErrNotConfigured means the optional email capability has no complete provider configuration.
	ErrNotConfigured = errors.New("email capability is not configured")
	// ErrInvalidMessage means the caller supplied an incomplete or invalid outbound message.
	ErrInvalidMessage = errors.New("email message is invalid")
	// ErrProviderResponseTooLarge prevents unbounded reads from an external provider.
	ErrProviderResponseTooLarge = errors.New("email provider response exceeds limit")
	// ErrInvalidProviderResponse means a successful provider response did not contain its message ID.
	ErrInvalidProviderResponse = errors.New("email provider returned an invalid response")
)

// ProviderError reports a failed provider status without exposing its response body.
type ProviderError struct {
	StatusCode int
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "email provider request failed"
	}
	return fmt.Sprintf("email provider returned HTTP status %d", e.StatusCode)
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Service owns one reusable HTTP client and the provider delivery policy.
type Service struct {
	from           string
	apiKey         string
	endpoint       string
	requestTimeout time.Duration
	client         httpDoer
}

type resendSendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type resendSendResponse struct {
	ID string `json:"id"`
}

// NewService constructs the process-wide email adapter from typed configuration.
func NewService(cfg *config.Config) *Service {
	emailConfig := config.EmailConfig{RequestTimeout: config.DefaultEmailRequestTimeout}
	if cfg != nil {
		emailConfig = cfg.Email
	}
	timeout := normalizedTimeout(emailConfig.RequestTimeout)
	return newService(
		emailConfig,
		&http.Client{Timeout: timeout},
		defaultResendEndpoint,
	)
}

func newService(cfg config.EmailConfig, client httpDoer, endpoint string) *Service {
	timeout := normalizedTimeout(cfg.RequestTimeout)
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultResendEndpoint
	}
	return &Service{
		from:           strings.TrimSpace(cfg.From),
		apiKey:         strings.TrimSpace(cfg.ResendAPIKey),
		endpoint:       endpoint,
		requestTimeout: timeout,
		client:         client,
	}
}

func normalizedTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return config.DefaultEmailRequestTimeout
	}
	return timeout
}

// IsConfigured reports whether this adapter can attempt provider delivery.
func (s *Service) IsConfigured() bool {
	return s != nil && s.from != "" && s.apiKey != "" && s.endpoint != "" && s.client != nil
}

// SendEmail sends one bounded, context-aware provider request.
func (s *Service) SendEmail(ctx context.Context, to []string, subject, htmlContent string) error {
	return s.sendEmail(ctx, to, subject, htmlContent, "")
}

// SendEmailIdempotent sends one message with a stable provider idempotency key.
func (s *Service) SendEmailIdempotent(
	ctx context.Context,
	to []string,
	subject string,
	htmlContent string,
	idempotencyKey string,
) error {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || len(idempotencyKey) > 256 || containsControl(idempotencyKey) {
		return fmt.Errorf("%w: idempotency key is invalid", ErrInvalidMessage)
	}
	return s.sendEmail(ctx, to, subject, htmlContent, idempotencyKey)
}

func (s *Service) sendEmail(
	ctx context.Context,
	to []string,
	subject string,
	htmlContent string,
	idempotencyKey string,
) error {
	if !s.IsConfigured() {
		return ErrNotConfigured
	}
	if ctx == nil {
		return fmt.Errorf("%w: context is required", ErrInvalidMessage)
	}
	if err := validateMessage(to, subject, htmlContent); err != nil {
		return err
	}

	payload, err := json.Marshal(resendSendRequest{
		From:    s.from,
		To:      to,
		Subject: subject,
		HTML:    htmlContent,
	})
	if err != nil {
		return fmt.Errorf("encode email request: %w", err)
	}

	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		s.endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("build email request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.apiKey)
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("send email request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read email provider response: %w", err)
	}
	if len(body) > maxProviderResponseBytes {
		return ErrProviderResponseTooLarge
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &ProviderError{StatusCode: response.StatusCode}
	}

	var providerResponse resendSendResponse
	if err := json.Unmarshal(body, &providerResponse); err != nil || strings.TrimSpace(providerResponse.ID) == "" {
		return ErrInvalidProviderResponse
	}
	return nil
}

func containsControl(value string) bool {
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return true
		}
	}
	return false
}

func validateMessage(to []string, subject, htmlContent string) error {
	if len(to) == 0 {
		return fmt.Errorf("%w: at least one recipient is required", ErrInvalidMessage)
	}
	if len(to) > maxResendRecipients {
		return fmt.Errorf("%w: at most %d recipients are allowed", ErrInvalidMessage, maxResendRecipients)
	}
	for index, recipient := range to {
		if _, err := mail.ParseAddress(strings.TrimSpace(recipient)); err != nil {
			return fmt.Errorf("%w: recipient %d is invalid", ErrInvalidMessage, index)
		}
	}
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidMessage)
	}
	if strings.TrimSpace(htmlContent) == "" {
		return fmt.Errorf("%w: HTML content is required", ErrInvalidMessage)
	}
	return nil
}

// SendPasswordResetEmail sends a password reset token email.
func (s *Service) SendPasswordResetEmail(ctx context.Context, to, resetToken string) error {
	subject := "Password Reset Request"
	htmlContent := fmt.Sprintf(`
		<h2>Password Reset Request</h2>
		<p>We received a request to reset your password.</p>
		<p>Use the token below to complete the reset:</p>
		<p style="font-size: 18px; font-weight: bold; color: #333;">%s</p>
		<p>This token expires in 30 minutes and can only be used once.</p>
		<p>If you did not request this, you can ignore this email.</p>
	`, html.EscapeString(resetToken))

	return s.SendEmail(ctx, []string{to}, subject, htmlContent)
}

// SendWelcomeEmail sends a welcome email.
func (s *Service) SendWelcomeEmail(ctx context.Context, to, username string) error {
	subject := "Welcome to Luas"
	htmlContent := fmt.Sprintf(`
		<h2>Welcome to Luas</h2>
		<p>Dear %s,</p>
		<p>Thank you for registering as our user!</p>
		<p>If you have any questions, please feel free to contact our support team.</p>
	`, html.EscapeString(username))

	return s.SendEmail(ctx, []string{to}, subject, htmlContent)
}

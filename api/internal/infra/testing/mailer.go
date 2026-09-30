package testing

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
)

// SentMail is one message captured by FakeMailer.
type SentMail struct {
	To             []string
	Subject        string
	HTML           string
	IdempotencyKey string
	// Kind is "generic", "password_reset", or "welcome".
	Kind string
	// Token and Username carry the template inputs of the typed helpers.
	Token    string
	Username string
}

// FakeMailer records outgoing email instead of sending it. It implements every method of the email
// service that starter mail seams depend on, so one value can stand in for user, invitation, and
// notification mailers. The zero value is configured and succeeds.
type FakeMailer struct {
	// Unconfigured makes IsConfigured report false, as a deployment without email would.
	Unconfigured bool
	// Err, when set, is returned by every send and nothing is recorded.
	Err error

	mu   sync.Mutex
	sent []SentMail
}

// NewFakeMailer returns a configured mailer that records every message.
func NewFakeMailer() *FakeMailer {
	return &FakeMailer{}
}

// IsConfigured reports whether the fake behaves like a configured email service.
func (m *FakeMailer) IsConfigured() bool {
	return !m.Unconfigured
}

// SendEmail records a generic message.
func (m *FakeMailer) SendEmail(_ context.Context, to []string, subject, html string) error {
	return m.record(SentMail{To: slices.Clone(to), Subject: subject, HTML: html, Kind: "generic"})
}

// SendEmailIdempotent records a generic message with its provider idempotency key.
func (m *FakeMailer) SendEmailIdempotent(_ context.Context, to []string, subject, html, key string) error {
	return m.record(SentMail{To: slices.Clone(to), Subject: subject, HTML: html, IdempotencyKey: key, Kind: "generic"})
}

// SendPasswordResetEmail records a password reset message and its token.
func (m *FakeMailer) SendPasswordResetEmail(_ context.Context, to, token string) error {
	return m.record(SentMail{To: []string{to}, Kind: "password_reset", Token: token})
}

// SendWelcomeEmail records a welcome message.
func (m *FakeMailer) SendWelcomeEmail(_ context.Context, to, username string) error {
	return m.record(SentMail{To: []string{to}, Kind: "welcome", Username: username})
}

func (m *FakeMailer) record(mail SentMail) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, mail)
	return nil
}

// Sent returns a copy of every recorded message in send order.
func (m *FakeMailer) Sent() []SentMail {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.sent)
}

// SentTo returns the recorded messages addressed to one recipient.
func (m *FakeMailer) SentTo(recipient string) []SentMail {
	var matches []SentMail
	for _, mail := range m.Sent() {
		if slices.ContainsFunc(mail.To, func(to string) bool { return strings.EqualFold(to, recipient) }) {
			matches = append(matches, mail)
		}
	}
	return matches
}

// AssertSent fails the test unless a message of the given kind reached the recipient.
func (m *FakeMailer) AssertSent(t testing.TB, recipient, kind string) SentMail {
	t.Helper()
	for _, mail := range m.SentTo(recipient) {
		if mail.Kind == kind {
			return mail
		}
	}
	t.Fatalf("no %q email sent to %s; sent: %+v", kind, recipient, m.Sent())
	return SentMail{}
}

// AssertNothingSent fails the test if any message was recorded.
func (m *FakeMailer) AssertNothingSent(t testing.TB) {
	t.Helper()
	if sent := m.Sent(); len(sent) > 0 {
		t.Fatalf("expected no email, got %d: %+v", len(sent), sent)
	}
}

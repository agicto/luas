package testing

import (
	"context"
	"errors"
	"testing"

	"github.com/zgiai/luas/api/internal/infra/email"
)

// mailService is every email.Service method that starter mail seams use. Both the real service and
// the fake must satisfy it, so the fake cannot drift from the production signatures.
type mailService interface {
	IsConfigured() bool
	SendEmail(ctx context.Context, to []string, subject, html string) error
	SendEmailIdempotent(ctx context.Context, to []string, subject, html, key string) error
	SendPasswordResetEmail(ctx context.Context, to, token string) error
	SendWelcomeEmail(ctx context.Context, to, username string) error
}

var (
	_ mailService = (*email.Service)(nil)
	_ mailService = (*FakeMailer)(nil)
)

func TestFakeMailerRecordsAndAsserts(t *testing.T) {
	mailer := NewFakeMailer()
	ctx := context.Background()

	if err := mailer.SendPasswordResetEmail(ctx, "Ada@Example.test", "reset-token"); err != nil {
		t.Fatal(err)
	}
	if err := mailer.SendEmailIdempotent(ctx, []string{"ops@example.test"}, "Hello", "<p>Hi</p>", "key-1"); err != nil {
		t.Fatal(err)
	}

	reset := mailer.AssertSent(t, "ada@example.test", "password_reset")
	if reset.Token != "reset-token" {
		t.Fatalf("token = %q", reset.Token)
	}
	generic := mailer.AssertSent(t, "ops@example.test", "generic")
	if generic.IdempotencyKey != "key-1" || generic.Subject != "Hello" {
		t.Fatalf("unexpected message: %+v", generic)
	}
	if len(mailer.Sent()) != 2 {
		t.Fatalf("sent %d messages", len(mailer.Sent()))
	}
}

func TestFakeMailerSimulatesOutagesAndMissingConfiguration(t *testing.T) {
	outage := errors.New("provider down")
	mailer := &FakeMailer{Err: outage, Unconfigured: true}

	if mailer.IsConfigured() {
		t.Fatal("unconfigured mailer reported configured")
	}
	if err := mailer.SendWelcomeEmail(context.Background(), "ada@example.test", "ada"); !errors.Is(err, outage) {
		t.Fatalf("err = %v", err)
	}
	mailer.AssertNothingSent(t)
}

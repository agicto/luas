package organization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/email"
)

type invitationEmailSenderStub struct {
	configured bool
	sendCalls  int
	to         []string
	subject    string
	html       string
}

func (s *invitationEmailSenderStub) IsConfigured() bool {
	return s.configured
}

func (s *invitationEmailSenderStub) SendEmail(_ context.Context, to []string, subject, htmlContent string) error {
	s.sendCalls++
	s.to = to
	s.subject = subject
	s.html = htmlContent
	return nil
}

func TestInvitationMailerEscapesFeatureOwnedTemplate(t *testing.T) {
	sender := &invitationEmailSenderStub{configured: true}
	mailer := newInvitationMailer(sender)
	expiresAt := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)

	err := mailer.SendInvitation(
		context.Background(),
		"member@example.com",
		"Acme <script>alert(1)</script>",
		domain.OrganizationRoleAdmin,
		"oinv_test.<unsafe>",
		expiresAt,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, sender.sendCalls)
	assert.Equal(t, []string{"member@example.com"}, sender.to)
	assert.Equal(t, "Organization invitation", sender.subject)
	assert.Contains(t, sender.html, "Acme &lt;script&gt;alert(1)&lt;/script&gt;")
	assert.Contains(t, sender.html, "oinv_test.&lt;unsafe&gt;")
	assert.Contains(t, sender.html, expiresAt.Format(time.RFC3339))
	assert.NotContains(t, sender.html, "<script>")
}

func TestInvitationMailerRejectsUnconfiguredDelivery(t *testing.T) {
	sender := &invitationEmailSenderStub{}
	mailer := newInvitationMailer(sender)

	err := mailer.SendInvitation(
		context.Background(),
		"member@example.com",
		"Acme",
		domain.OrganizationRoleMember,
		"oinv_test.secret",
		time.Now(),
	)
	require.ErrorIs(t, err, email.ErrNotConfigured)
	assert.Zero(t, sender.sendCalls)
}

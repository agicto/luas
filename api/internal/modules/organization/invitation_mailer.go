package organization

import (
	"context"
	"fmt"
	"html"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/email"
)

// InvitationMailer is the organization-owned seam for invitation delivery.
type InvitationMailer interface {
	IsConfigured() bool
	SendInvitation(ctx context.Context, to, organizationName string, role domain.OrganizationRole, token string, expiresAt time.Time) error
}

type invitationEmailSender interface {
	IsConfigured() bool
	SendEmail(ctx context.Context, to []string, subject, htmlContent string) error
}

type invitationMailer struct {
	sender invitationEmailSender
}

// NewInvitationMailer adapts the shared email capability to organization-owned copy.
func NewInvitationMailer(service *email.Service) InvitationMailer {
	return newInvitationMailer(service)
}

func newInvitationMailer(sender invitationEmailSender) *invitationMailer {
	return &invitationMailer{sender: sender}
}

func (m *invitationMailer) IsConfigured() bool {
	return m != nil && m.sender != nil && m.sender.IsConfigured()
}

func (m *invitationMailer) SendInvitation(
	ctx context.Context,
	to string,
	organizationName string,
	role domain.OrganizationRole,
	token string,
	expiresAt time.Time,
) error {
	if !m.IsConfigured() {
		return email.ErrNotConfigured
	}
	htmlContent := fmt.Sprintf(`
		<h2>Organization invitation</h2>
		<p>You have been invited to join <strong>%s</strong> as <strong>%s</strong>.</p>
		<p>Sign in with this email address and enter the one-time token below:</p>
		<p><code>%s</code></p>
		<p>This invitation expires at %s.</p>
	`,
		html.EscapeString(organizationName),
		html.EscapeString(string(role)),
		html.EscapeString(token),
		html.EscapeString(expiresAt.UTC().Format(time.RFC3339)),
	)
	return m.sender.SendEmail(ctx, []string{to}, "Organization invitation", htmlContent)
}

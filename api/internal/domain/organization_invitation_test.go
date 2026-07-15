package domain

import (
	"testing"
	"time"
)

func TestOrganizationInvitationStatus(t *testing.T) {
	now := time.Date(2026, time.July, 15, 20, 0, 0, 0, time.UTC)
	acceptedAt := now.Add(-time.Minute)
	revokedAt := now.Add(-2 * time.Minute)

	tests := []struct {
		name       string
		invitation OrganizationInvitation
		want       OrganizationInvitationStatus
	}{
		{
			name:       "pending",
			invitation: OrganizationInvitation{ExpiresAt: now.Add(time.Hour)},
			want:       OrganizationInvitationStatusPending,
		},
		{
			name:       "expired at boundary",
			invitation: OrganizationInvitation{ExpiresAt: now},
			want:       OrganizationInvitationStatusExpired,
		},
		{
			name:       "revoked takes precedence over expiry",
			invitation: OrganizationInvitation{ExpiresAt: now.Add(-time.Hour), RevokedAt: &revokedAt},
			want:       OrganizationInvitationStatusRevoked,
		},
		{
			name:       "accepted takes precedence",
			invitation: OrganizationInvitation{ExpiresAt: now.Add(-time.Hour), AcceptedAt: &acceptedAt},
			want:       OrganizationInvitationStatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.invitation.Status(now); got != tt.want {
				t.Fatalf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

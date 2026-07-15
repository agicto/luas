package domain

import "context"

// AuthenticationIdentity is the current server-resolved user session identity.
// Authorization remains owned by the relevant starter or product policy.
type AuthenticationIdentity struct {
	UserID    uint
	Username  string
	SessionID string
}

// SessionAuthenticator resolves one opaque bearer credential against current persistence.
type SessionAuthenticator interface {
	Authenticate(ctx context.Context, credential string) (*AuthenticationIdentity, error)
}

// AuthenticationSessionMaintainer owns bounded retention cleanup for terminal sessions.
type AuthenticationSessionMaintainer interface {
	PruneAuthenticationSessions(ctx context.Context, batch int) (int64, error)
}

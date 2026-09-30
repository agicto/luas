package domain

import (
	"context"
	"time"
)

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

// IssuedSession is a newly created authentication session. Credential is plaintext and returned
// exactly once; callers must place it only in server-controlled custody such as an HttpOnly cookie.
type IssuedSession struct {
	Credential string
	ExpiresAt  time.Time
	User       *User
}

// CredentialSignIn verifies account credentials and issues a session. The authorize callback runs
// after verification and before issuance, so a caller that is rejected never receives a session.
// Unknown, wrong, and disabled accounts all return ErrInvalidCredentials.
type CredentialSignIn interface {
	SignIn(
		ctx context.Context,
		identifier string,
		password string,
		authorize func(context.Context, *User) error,
	) (*IssuedSession, error)
}

// SessionRevoker ends one session by its credential without revealing whether it existed.
type SessionRevoker interface {
	RevokeSession(ctx context.Context, credential string) error
}

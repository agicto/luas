package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

const (
	authenticationSessionIDBytes    = 16
	authenticationCredentialBytes   = 32
	maxAuthenticationCredentialSize = 128
	defaultSessionPruneBatch        = 500
)

const (
	sessionRevocationLogout          = "logout"
	sessionRevocationPasswordChange  = "password_change"
	sessionRevocationPasswordReset   = "password_reset"
	sessionRevocationAccountDeleted  = "account_deleted"
	sessionRevocationAccountDisabled = "account_disabled"
	sessionRevocationExpired         = "expired"
	sessionRevocationIdleTimeout     = "idle_timeout"
)

// IssuedAuthenticationSession is the one-time plaintext session response.
type IssuedAuthenticationSession struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
}

// SessionService owns opaque user session issuance, resolution, revocation, and retention.
type SessionService struct {
	repo        *repository
	policy      config.AuthenticationConfig
	now         func() time.Time
	randomBytes func(int) ([]byte, error)
}

var (
	_ domain.SessionAuthenticator            = (*SessionService)(nil)
	_ domain.AuthenticationSessionMaintainer = (*SessionService)(nil)
)

// NewSessionService creates the default user starter's persistent session authority.
func NewSessionService(repo *repository, cfg *config.Config) *SessionService {
	policy := config.AuthenticationConfig{}
	if cfg != nil {
		policy = cfg.Authentication
	}
	policy = normalizedAuthenticationPolicy(policy)
	return &SessionService{
		repo:        repo,
		policy:      policy,
		now:         time.Now,
		randomBytes: crypto.GenerateKey,
	}
}

// Issue creates one hash-only persistent session and returns its plaintext credential once.
func (s *SessionService) Issue(ctx context.Context, user *domain.User) (*IssuedAuthenticationSession, error) {
	if user == nil || user.ID == 0 {
		return nil, domain.ErrAuthenticationRequired
	}
	if !user.IsActive() {
		return nil, domain.ErrAccountDisabled
	}
	if s == nil || s.repo == nil || s.randomBytes == nil {
		return nil, domain.ErrServiceUnavailable
	}

	sessionIDBytes, err := s.randomBytes(authenticationSessionIDBytes)
	if err != nil {
		return nil, fmt.Errorf("generate authentication session ID: %w", err)
	}
	credentialBytes, err := s.randomBytes(authenticationCredentialBytes)
	if err != nil {
		return nil, fmt.Errorf("generate authentication session credential: %w", err)
	}

	now := s.currentTime()
	expiresAt := now.Add(s.policy.SessionTTL)
	idleExpiresAt := minTime(now.Add(s.policy.SessionIdleTimeout), expiresAt)
	credential := base64.RawURLEncoding.EncodeToString(credentialBytes)
	record := &AuthenticationSessionPO{
		ID:            base64.RawURLEncoding.EncodeToString(sessionIDBytes),
		UserID:        user.ID,
		TokenHash:     crypto.SHA256Hex(credential),
		ExpiresAt:     expiresAt,
		IdleExpiresAt: idleExpiresAt,
		LastSeenAt:    now,
	}
	if err := s.repo.createAuthenticationSession(ctx, record); err != nil {
		return nil, err
	}

	return &IssuedAuthenticationSession{
		AccessToken: credential,
		TokenType:   "Bearer",
		ExpiresIn:   max(int64(1), int64(expiresAt.Sub(now)/time.Second)),
	}, nil
}

// Authenticate resolves one opaque bearer credential against current session and user state.
func (s *SessionService) Authenticate(ctx context.Context, credential string) (*domain.AuthenticationIdentity, error) {
	if s == nil || s.repo == nil {
		return nil, domain.ErrServiceUnavailable
	}
	credential = strings.TrimSpace(credential)
	if !validAuthenticationCredential(credential) {
		return nil, domain.ErrAuthenticationRequired
	}
	return s.repo.authenticateSession(
		ctx,
		crypto.SHA256Hex(credential),
		s.currentTime(),
		s.policy.SessionTouchInterval,
		s.policy.SessionIdleTimeout,
	)
}

// Revoke invalidates a credential without revealing whether it existed.
func (s *SessionService) Revoke(ctx context.Context, credential, reason string) error {
	if s == nil || s.repo == nil {
		return domain.ErrServiceUnavailable
	}
	credential = strings.TrimSpace(credential)
	if !validAuthenticationCredential(credential) {
		return nil
	}
	return s.repo.revokeAuthenticationSessionByTokenHash(
		ctx,
		crypto.SHA256Hex(credential),
		s.currentTime(),
		normalizeSessionRevocationReason(reason),
	)
}

// RevokeByID invalidates the already-authenticated current session.
func (s *SessionService) RevokeByID(ctx context.Context, sessionID, reason string) error {
	if s == nil || s.repo == nil {
		return domain.ErrServiceUnavailable
	}
	if strings.TrimSpace(sessionID) == "" {
		return domain.ErrAuthenticationRequired
	}
	return s.repo.revokeAuthenticationSessionByID(
		ctx,
		strings.TrimSpace(sessionID),
		s.currentTime(),
		normalizeSessionRevocationReason(reason),
	)
}

// RevokeUser invalidates every active session for one user.
func (s *SessionService) RevokeUser(ctx context.Context, userID uint, reason string) error {
	if s == nil || s.repo == nil {
		return domain.ErrServiceUnavailable
	}
	return s.repo.revokeUserAuthenticationSessions(
		ctx,
		userID,
		s.currentTime(),
		normalizeSessionRevocationReason(reason),
	)
}

// PruneAuthenticationSessions removes terminal rows older than the configured retention window.
func (s *SessionService) PruneAuthenticationSessions(ctx context.Context, batch int) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, domain.ErrServiceUnavailable
	}
	if batch <= 0 || batch > 10_000 {
		batch = defaultSessionPruneBatch
	}
	return s.repo.pruneAuthenticationSessions(
		ctx,
		s.currentTime().Add(-s.policy.SessionRetention),
		batch,
	)
}

func (s *SessionService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func validAuthenticationCredential(value string) bool {
	if value == "" || len(value) > maxAuthenticationCredentialSize {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == authenticationCredentialBytes
}

func normalizeSessionRevocationReason(value string) string {
	switch strings.TrimSpace(value) {
	case sessionRevocationLogout,
		sessionRevocationPasswordChange,
		sessionRevocationPasswordReset,
		sessionRevocationAccountDeleted,
		sessionRevocationAccountDisabled,
		sessionRevocationExpired,
		sessionRevocationIdleTimeout:
		return strings.TrimSpace(value)
	default:
		return "security_event"
	}
}

func normalizedAuthenticationPolicy(policy config.AuthenticationConfig) config.AuthenticationConfig {
	if policy.SessionTTL < 15*time.Minute || policy.SessionTTL > 180*24*time.Hour {
		policy.SessionTTL = config.DefaultAuthenticationSessionTTL
	}
	if policy.SessionIdleTimeout < 5*time.Minute || policy.SessionIdleTimeout > policy.SessionTTL {
		policy.SessionIdleTimeout = min(config.DefaultAuthenticationSessionIdleTimeout, policy.SessionTTL)
	}
	if policy.SessionTouchInterval <= 0 ||
		policy.SessionTouchInterval > time.Hour ||
		policy.SessionTouchInterval > policy.SessionIdleTimeout {
		policy.SessionTouchInterval = min(config.DefaultAuthenticationSessionTouchInterval, policy.SessionIdleTimeout)
	}
	if policy.SessionRetention < 0 || policy.SessionRetention > 365*24*time.Hour {
		policy.SessionRetention = config.DefaultAuthenticationSessionRetention
	}
	return policy
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

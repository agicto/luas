package domain

import (
	"context"
	"strings"
	"time"
)

// APIKey represents an application API key owned by a user.
type APIKey struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"user_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// IsActive returns whether the API key can still be used.
func (k *APIKey) IsActive(now time.Time) bool {
	if k == nil {
		return false
	}
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(now) {
		return false
	}
	return true
}

// HasScope checks whether the key grants a required scope.
func (k *APIKey) HasScope(scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return true
	}

	for _, candidate := range k.Scopes {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == scope {
			return true
		}
	}
	return false
}

// APIKeyRepository defines the contract for API key persistence.
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	Update(ctx context.Context, key *APIKey) error
	FindByID(ctx context.Context, id uint) (*APIKey, error)
	FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*APIKey, int64, error)
	FindByHash(ctx context.Context, hash string) (*APIKey, error)
}

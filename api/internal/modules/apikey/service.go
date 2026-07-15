package apikey

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/capabilities/idgen"
	"github.com/zgiai/luas/api/internal/domain"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

// lastUsedAtThrottle skips the LastUsedAt write if the previous update is
// fresher than this window. Sub-minute precision on "last used" tracking
// is not useful and the write amplification on hot keys is real.
const lastUsedAtThrottle = time.Minute

const (
	MaxAPIKeyScopes      = 32
	MaxAPIKeyScopeLength = 65
)

var apiKeyScopePattern = regexp.MustCompile(`^(?:\*|[a-z][a-z0-9_-]{0,31}:[a-z][a-z0-9_-]{0,31})$`)

// Service defines API key operations.
type Service interface {
	CreateForUser(ctx context.Context, userID uint, req *APIKeyCreateRequest) (*CreateResult, error)
	ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.APIKey, int64, error)
	RevokeForUser(ctx context.Context, userID, id uint) error
	Validate(ctx context.Context, plaintext string, requiredScopes ...string) (*domain.APIKey, error)
}

// CreateResult carries the persisted key metadata plus the one-time plaintext key.
type CreateResult struct {
	APIKey       *domain.APIKey
	PlaintextKey string
}

type service struct {
	repo domain.APIKeyRepository
}

// NewService creates a new API key service.
func NewService(repo domain.APIKeyRepository) *service {
	return &service{repo: repo}
}

func (s *service) CreateForUser(ctx context.Context, userID uint, req *APIKeyCreateRequest) (*CreateResult, error) {
	if userID == 0 || req == nil {
		return nil, domain.ErrInvalidInput
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return nil, domain.ErrInvalidInput
	}
	scopes, err := normalizeAPIKeyScopes(req.Scopes)
	if err != nil {
		return nil, err
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrInvalidInput
	}

	secret, err := crypto.GenerateKeyHex(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key secret: %w", err)
	}

	keyPrefix := "luas_" + strings.ToLower(idgen.ShortID())
	plaintext := keyPrefix + "." + secret

	apiKey := &domain.APIKey{
		UserID:    userID,
		Name:      name,
		KeyPrefix: keyPrefix,
		KeyHash:   crypto.SHA256Hex(plaintext),
		Scopes:    scopes,
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "create",
		Resource:   "api_keys",
		TargetType: "api_key",
		TargetID:   strconv.FormatUint(uint64(apiKey.ID), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"name":       {After: apiKey.Name},
			"scopes":     {After: append([]string(nil), apiKey.Scopes...)},
			"expires_at": {After: apiKey.ExpiresAt},
		},
	})

	return &CreateResult{
		APIKey:       apiKey,
		PlaintextKey: plaintext,
	}, nil
}

func (s *service) ListForUser(ctx context.Context, userID uint, page, pageSize int) ([]*domain.APIKey, int64, error) {
	if userID == 0 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.repo.FindByUserID(ctx, userID, page, pageSize)
}

func (s *service) RevokeForUser(ctx context.Context, userID, id uint) error {
	if userID == 0 || id == 0 {
		return domain.ErrInvalidInput
	}

	now := time.Now()
	changed, err := s.repo.Revoke(ctx, userID, id, now)
	if err != nil {
		if errors.Is(err, domain.ErrServiceUnavailable) || errors.Is(err, domain.ErrAPIKeyNotFound) {
			return err
		}
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	if !changed {
		return nil
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "revoke",
		Resource:   "api_keys",
		TargetType: "api_key",
		TargetID:   strconv.FormatUint(uint64(id), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"revoked_at": {After: now},
		},
	})
	return nil
}

func (s *service) Validate(ctx context.Context, plaintext string, requiredScopes ...string) (*domain.APIKey, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, domain.ErrAPIKeyInvalid
	}

	key, err := s.repo.FindByHash(ctx, crypto.SHA256Hex(plaintext))
	if err != nil {
		if errors.Is(err, domain.ErrServiceUnavailable) {
			return nil, err
		}
		return nil, domain.ErrAPIKeyInvalid
	}

	now := time.Now()
	if key.RevokedAt != nil {
		return nil, domain.ErrAPIKeyRevoked
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(now) {
		return nil, domain.ErrAPIKeyExpired
	}
	normalizedRequiredScopes, err := normalizeAPIKeyScopes(requiredScopes)
	if err != nil {
		return nil, fmt.Errorf("invalid required api key scope configuration: %w", err)
	}
	for _, scope := range normalizedRequiredScopes {
		if !key.HasScope(scope) {
			return nil, domain.ErrPermissionDenied
		}
	}

	if key.LastUsedAt == nil || now.Sub(*key.LastUsedAt) >= lastUsedAtThrottle {
		if err := s.repo.RecordUse(ctx, key.ID, now, now.Add(-lastUsedAtThrottle)); err != nil {
			// Auth already succeeded; degrade gracefully on the write.
			log.Printf("apikey: failed to update LastUsedAt for key %d: %v", key.ID, err)
		} else {
			key.LastUsedAt = &now
		}
	}

	return key, nil
}

func normalizeAPIKeyScopes(scopes []string) ([]string, error) {
	if len(scopes) > MaxAPIKeyScopes {
		return nil, domain.ErrInvalidInput
	}

	normalized := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))

	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		if scope == "" || len(scope) > MaxAPIKeyScopeLength || !apiKeyScopePattern.MatchString(scope) {
			return nil, domain.ErrInvalidInput
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}

	slices.Sort(normalized)
	return normalized, nil
}

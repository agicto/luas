package apikey

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/zgiai/zgo/internal/capabilities/crypto"
	"github.com/zgiai/zgo/internal/capabilities/idgen"
	"github.com/zgiai/zgo/internal/domain"
	auditstarter "github.com/zgiai/zgo/internal/modules/audit"
)

// lastUsedAtThrottle skips the LastUsedAt write if the previous update is
// fresher than this window. Sub-minute precision on "last used" tracking
// is not useful and the write amplification on hot keys is real.
const lastUsedAtThrottle = time.Minute

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
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	secret, err := crypto.GenerateKeyHex(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key secret: %w", err)
	}

	keyPrefix := "zgo_" + strings.ToLower(idgen.ShortID())
	plaintext := keyPrefix + "." + secret

	apiKey := &domain.APIKey{
		UserID:    userID,
		Name:      name,
		KeyPrefix: keyPrefix,
		KeyHash:   crypto.SHA256Hex(plaintext),
		Scopes:    normalizeScopes(req.Scopes),
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
	key, err := s.repo.FindByID(ctx, id)
	if err != nil || key.UserID != userID {
		return domain.ErrAPIKeyNotFound
	}

	now := time.Now()
	before := key.RevokedAt
	key.RevokedAt = &now
	if err := s.repo.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "revoke",
		Resource:   "api_keys",
		TargetType: "api_key",
		TargetID:   strconv.FormatUint(uint64(key.ID), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"revoked_at": {Before: before, After: now},
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
		return nil, domain.ErrAPIKeyInvalid
	}

	now := time.Now()
	if key.RevokedAt != nil {
		return nil, domain.ErrAPIKeyRevoked
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(now) {
		return nil, domain.ErrAPIKeyExpired
	}
	for _, scope := range normalizeScopes(requiredScopes) {
		if !key.HasScope(scope) {
			return nil, domain.ErrPermissionDenied
		}
	}

	if key.LastUsedAt == nil || now.Sub(*key.LastUsedAt) >= lastUsedAtThrottle {
		key.LastUsedAt = &now
		if err := s.repo.Update(ctx, key); err != nil {
			// Auth already succeeded; degrade gracefully on the write.
			log.Printf("apikey: failed to update LastUsedAt for key %d: %v", key.ID, err)
		}
	}

	return key, nil
}

func normalizeScopes(scopes []string) []string {
	normalized := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))

	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}

	slices.Sort(normalized)
	return normalized
}

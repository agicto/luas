package apikey

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeRepository struct {
	nextID        uint
	keys          map[uint]*domain.APIKey
	findByHashErr error
	revokeErr     error
	revokeWrites  int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		nextID: 1,
		keys:   make(map[uint]*domain.APIKey),
	}
}

func (r *fakeRepository) Create(ctx context.Context, key *domain.APIKey) error {
	key.ID = r.nextID
	r.nextID++
	r.keys[key.ID] = cloneAPIKey(key)
	return nil
}

func (r *fakeRepository) Revoke(
	ctx context.Context,
	userID, id uint,
	revokedAt time.Time,
) (bool, error) {
	if r.revokeErr != nil {
		return false, r.revokeErr
	}
	key, ok := r.keys[id]
	if !ok || key.UserID != userID {
		return false, domain.ErrAPIKeyNotFound
	}
	if key.RevokedAt != nil {
		return false, nil
	}
	updated := cloneAPIKey(key)
	updated.RevokedAt = &revokedAt
	r.keys[id] = updated
	r.revokeWrites++
	return true, nil
}

func (r *fakeRepository) RecordUse(
	ctx context.Context,
	id uint,
	usedAt, staleBefore time.Time,
) error {
	key, ok := r.keys[id]
	if !ok || key.RevokedAt != nil || (key.ExpiresAt != nil && !key.ExpiresAt.After(usedAt)) {
		return nil
	}
	if key.LastUsedAt != nil && key.LastUsedAt.After(staleBefore) {
		return nil
	}
	updated := cloneAPIKey(key)
	updated.LastUsedAt = &usedAt
	r.keys[id] = updated
	return nil
}

func (r *fakeRepository) FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*domain.APIKey, int64, error) {
	items := make([]*domain.APIKey, 0)
	for _, key := range r.keys {
		if key.UserID == userID {
			items = append(items, cloneAPIKey(key))
		}
	}
	return items, int64(len(items)), nil
}

func (r *fakeRepository) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	if r.findByHashErr != nil {
		return nil, r.findByHashErr
	}
	for _, key := range r.keys {
		if key.KeyHash == hash {
			return cloneAPIKey(key), nil
		}
	}
	return nil, domain.ErrAPIKeyNotFound
}

func cloneAPIKey(key *domain.APIKey) *domain.APIKey {
	if key == nil {
		return nil
	}
	copyKey := *key
	copyKey.Scopes = append([]string(nil), key.Scopes...)
	return &copyKey
}

func TestServiceCreateAndValidate(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo)

	created, err := service.CreateForUser(context.Background(), 42, &APIKeyCreateRequest{
		Name:   "deploy",
		Scopes: []string{"models:invoke", "models:invoke"},
	})
	if err != nil {
		t.Fatalf("CreateForUser() error = %v", err)
	}

	if created.PlaintextKey == "" {
		t.Fatal("expected plaintext key to be returned")
	}
	if created.APIKey.KeyHash == created.PlaintextKey {
		t.Fatal("expected stored hash to differ from plaintext")
	}

	validated, err := service.Validate(context.Background(), created.PlaintextKey, "models:invoke")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.UserID != 42 {
		t.Fatalf("validated.UserID = %d, want 42", validated.UserID)
	}
	if validated.LastUsedAt == nil {
		t.Fatal("expected last_used_at to be updated")
	}
}

func TestServiceCanonicalizesScopesAndAcceptsWildcard(t *testing.T) {
	repo := newFakeRepository()
	created, err := NewService(repo).CreateForUser(context.Background(), 42, &APIKeyCreateRequest{
		Name:   " deploy ",
		Scopes: []string{" Models:Read ", "*", "models:read"},
	})
	require.NoError(t, err)
	assert.Equal(t, "deploy", created.APIKey.Name)
	assert.Equal(t, []string{"*", "models:read"}, created.APIKey.Scopes)

	_, err = NewService(repo).Validate(context.Background(), created.PlaintextKey, "jobs:write")
	require.NoError(t, err)
}

func TestServiceRejectsExpiredKeyAtCreation(t *testing.T) {
	repo := newFakeRepository()
	expiresAt := time.Now().Add(-time.Minute)

	_, err := NewService(repo).CreateForUser(context.Background(), 42, &APIKeyCreateRequest{
		Name:      "expired",
		Scopes:    []string{"models:read"},
		ExpiresAt: &expiresAt,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Empty(t, repo.keys)
}

func TestServiceValidateScopeDenied(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo)

	created, err := service.CreateForUser(context.Background(), 7, &APIKeyCreateRequest{
		Name:   "readonly",
		Scopes: []string{"models:read"},
	})
	if err != nil {
		t.Fatalf("CreateForUser() error = %v", err)
	}

	_, err = service.Validate(context.Background(), created.PlaintextKey, "models:write")
	if err != domain.ErrPermissionDenied {
		t.Fatalf("Validate() error = %v, want %v", err, domain.ErrPermissionDenied)
	}
}

func TestServiceRejectsInvalidScopesBeforePersistence(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
	}{
		{name: "ambiguous delimiter", scopes: []string{"models:read,admin:all"}},
		{name: "missing action", scopes: []string{"models"}},
		{name: "empty element", scopes: []string{"models:read", " "}},
		{name: "too many", scopes: apiKeyTestScopes(33)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepository()

			_, err := NewService(repo).CreateForUser(
				context.Background(),
				7,
				&APIKeyCreateRequest{Name: "invalid", Scopes: tt.scopes},
			)

			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("CreateForUser() error = %v, want invalid input", err)
			}
			if len(repo.keys) != 0 {
				t.Fatalf("persisted %d key(s) for invalid scopes", len(repo.keys))
			}
		})
	}
}

func apiKeyTestScopes(count int) []string {
	scopes := make([]string, count)
	for i := range scopes {
		scopes[i] = fmt.Sprintf("models%d:read", i)
	}
	return scopes
}

func TestServiceRevocationIsIdempotent(t *testing.T) {
	repo := newFakeRepository()
	created, err := NewService(repo).CreateForUser(
		context.Background(),
		7,
		&APIKeyCreateRequest{Name: "deploy", Scopes: []string{"models:invoke"}},
	)
	if err != nil {
		t.Fatalf("CreateForUser() error = %v", err)
	}

	service := NewService(repo)
	if err := service.RevokeForUser(context.Background(), 7, created.APIKey.ID); err != nil {
		t.Fatalf("first RevokeForUser() error = %v", err)
	}
	if err := service.RevokeForUser(context.Background(), 7, created.APIKey.ID); err != nil {
		t.Fatalf("second RevokeForUser() error = %v", err)
	}
	if repo.revokeWrites != 1 {
		t.Fatalf("Revoke() writes = %d, want 1", repo.revokeWrites)
	}
}

func TestServicePreservesServiceUnavailable(t *testing.T) {
	t.Run("revoke", func(t *testing.T) {
		repo := newFakeRepository()
		repo.revokeErr = domain.ErrServiceUnavailable

		err := NewService(repo).RevokeForUser(context.Background(), 7, 1)

		if !errors.Is(err, domain.ErrServiceUnavailable) {
			t.Fatalf("RevokeForUser() error = %v, want service unavailable", err)
		}
	})

	t.Run("validate", func(t *testing.T) {
		repo := newFakeRepository()
		repo.findByHashErr = domain.ErrServiceUnavailable

		_, err := NewService(repo).Validate(context.Background(), "luas_test.secret")

		if !errors.Is(err, domain.ErrServiceUnavailable) {
			t.Fatalf("Validate() error = %v, want service unavailable", err)
		}
	})
}

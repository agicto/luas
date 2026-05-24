package apikey

import (
	"context"
	"testing"

	"github.com/zgiai/zgo/internal/domain"
)

type fakeRepository struct {
	nextID uint
	keys   map[uint]*domain.APIKey
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

func (r *fakeRepository) Update(ctx context.Context, key *domain.APIKey) error {
	r.keys[key.ID] = cloneAPIKey(key)
	return nil
}

func (r *fakeRepository) FindByID(ctx context.Context, id uint) (*domain.APIKey, error) {
	key, ok := r.keys[id]
	if !ok {
		return nil, domain.ErrAPIKeyNotFound
	}
	return cloneAPIKey(key), nil
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

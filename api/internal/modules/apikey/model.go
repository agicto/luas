package apikey

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

// APIKeyPO is the persistent object for API keys.
type APIKeyPO struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	UserID     uint           `gorm:"not null;index"`
	Name       string         `gorm:"size:100;not null"`
	KeyPrefix  string         `gorm:"size:32;not null;index"`
	KeyHash    string         `gorm:"size:64;not null;uniqueIndex"`
	Scopes     string         `gorm:"type:text"`
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
}

// TableName specifies the database table name.
func (APIKeyPO) TableName() string {
	return "api_keys"
}

func (po *APIKeyPO) toDomain() (*domain.APIKey, error) {
	if po == nil {
		return nil, nil
	}
	scopes, err := decodeScopes(po.Scopes)
	if err != nil {
		return nil, fmt.Errorf("decode api key scopes: %w", err)
	}

	return &domain.APIKey{
		ID:         po.ID,
		UserID:     po.UserID,
		Name:       po.Name,
		KeyPrefix:  po.KeyPrefix,
		KeyHash:    po.KeyHash,
		Scopes:     scopes,
		LastUsedAt: po.LastUsedAt,
		ExpiresAt:  po.ExpiresAt,
		RevokedAt:  po.RevokedAt,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}, nil
}

func newAPIKeyPO(key *domain.APIKey) (*APIKeyPO, error) {
	if key == nil {
		return nil, nil
	}
	encodedScopes, err := encodeScopes(key.Scopes)
	if err != nil {
		return nil, fmt.Errorf("encode api key scopes: %w", err)
	}

	return &APIKeyPO{
		ID:         key.ID,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
		UserID:     key.UserID,
		Name:       key.Name,
		KeyPrefix:  key.KeyPrefix,
		KeyHash:    key.KeyHash,
		Scopes:     encodedScopes,
		LastUsedAt: key.LastUsedAt,
		ExpiresAt:  key.ExpiresAt,
		RevokedAt:  key.RevokedAt,
	}, nil
}

func toDomainList(items []*APIKeyPO) ([]*domain.APIKey, error) {
	result := make([]*domain.APIKey, len(items))
	for i, item := range items {
		key, err := item.toDomain()
		if err != nil {
			return nil, err
		}
		result[i] = key
	}
	return result, nil
}

func decodeScopes(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}

	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "[") {
		var scopes []string
		if err := json.Unmarshal([]byte(trimmed), &scopes); err != nil {
			return nil, err
		}
		if scopes == nil {
			return []string{}, nil
		}
		return scopes, nil
	}

	// Compatibility for rows written before scopes became JSON encoded.
	parts := strings.Split(trimmed, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return scopes, nil
}

func encodeScopes(scopes []string) (string, error) {
	if scopes == nil {
		scopes = []string{}
	}
	encoded, err := json.Marshal(scopes)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

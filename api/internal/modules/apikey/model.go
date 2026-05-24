package apikey

import (
	"strings"
	"time"

	"github.com/zgiai/zgo/internal/domain"
	"gorm.io/gorm"
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

func (po *APIKeyPO) toDomain() *domain.APIKey {
	if po == nil {
		return nil
	}

	return &domain.APIKey{
		ID:         po.ID,
		UserID:     po.UserID,
		Name:       po.Name,
		KeyPrefix:  po.KeyPrefix,
		KeyHash:    po.KeyHash,
		Scopes:     splitScopes(po.Scopes),
		LastUsedAt: po.LastUsedAt,
		ExpiresAt:  po.ExpiresAt,
		RevokedAt:  po.RevokedAt,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func newAPIKeyPO(key *domain.APIKey) *APIKeyPO {
	if key == nil {
		return nil
	}

	return &APIKeyPO{
		ID:         key.ID,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
		UserID:     key.UserID,
		Name:       key.Name,
		KeyPrefix:  key.KeyPrefix,
		KeyHash:    key.KeyHash,
		Scopes:     joinScopes(key.Scopes),
		LastUsedAt: key.LastUsedAt,
		ExpiresAt:  key.ExpiresAt,
		RevokedAt:  key.RevokedAt,
	}
}

func toDomainList(items []*APIKeyPO) []*domain.APIKey {
	result := make([]*domain.APIKey, len(items))
	for i, item := range items {
		result[i] = item.toDomain()
	}
	return result
}

func splitScopes(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return scopes
}

func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	return strings.Join(scopes, ",")
}

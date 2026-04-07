package apikey

import (
	"time"

	"github.com/zgiai/zgo/internal/domain"
)

// APIKeyCreateRequest represents an API key creation request.
type APIKeyCreateRequest struct {
	Name      string     `json:"name" binding:"required,max=100"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse is the standard API key response DTO.
type APIKeyResponse struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"user_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// APIKeyCreateResponse returns the one-time plaintext key plus metadata.
type APIKeyCreateResponse struct {
	APIKey       *APIKeyResponse `json:"api_key"`
	PlaintextKey string          `json:"plaintext_key"`
}

func toResponse(key *domain.APIKey) *APIKeyResponse {
	if key == nil {
		return nil
	}

	return &APIKeyResponse{
		ID:         key.ID,
		UserID:     key.UserID,
		Name:       key.Name,
		KeyPrefix:  key.KeyPrefix,
		Scopes:     append([]string(nil), key.Scopes...),
		LastUsedAt: key.LastUsedAt,
		ExpiresAt:  key.ExpiresAt,
		RevokedAt:  key.RevokedAt,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
	}
}

func toCreateResponse(result *CreateResult) *APIKeyCreateResponse {
	if result == nil {
		return nil
	}

	return &APIKeyCreateResponse{
		APIKey:       toResponse(result.APIKey),
		PlaintextKey: result.PlaintextKey,
	}
}

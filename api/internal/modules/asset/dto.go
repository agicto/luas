package asset

import (
	"time"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
	"github.com/zgiai/luas/api/internal/domain"
)

type createUploadIntentRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	OriginalName   string `json:"original_name" binding:"required"`
	MediaType      string `json:"media_type" binding:"required"`
	SizeBytes      int64  `json:"size_bytes" binding:"required"`
}

type AssetResponse struct {
	ID           string             `json:"id"`
	OriginalName string             `json:"original_name"`
	MediaType    string             `json:"media_type"`
	SizeBytes    int64              `json:"size_bytes"`
	Status       domain.AssetStatus `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	ReadyAt      *time.Time         `json:"ready_at"`
}

type TransferGrantResponse struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type UploadIntentResponse struct {
	Asset  *AssetResponse        `json:"asset"`
	Upload TransferGrantResponse `json:"upload"`
}

func toAssetResponse(asset *domain.Asset) *AssetResponse {
	if asset == nil {
		return nil
	}
	return &AssetResponse{
		ID:           asset.ID,
		OriginalName: asset.OriginalName,
		MediaType:    asset.MediaType,
		SizeBytes:    asset.SizeBytes,
		Status:       asset.Status,
		CreatedAt:    asset.CreatedAt,
		ReadyAt:      cloneAssetTime(asset.ReadyAt),
	}
}

func toTransferGrantResponse(grant storagecap.TransferGrant) TransferGrantResponse {
	headers := grant.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	return TransferGrantResponse{
		Method:    grant.Method,
		URL:       grant.URL,
		Headers:   headers,
		ExpiresAt: grant.ExpiresAt,
	}
}

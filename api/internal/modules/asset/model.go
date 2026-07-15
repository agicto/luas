package asset

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

const (
	operationComplete = "complete"
	operationDelete   = "delete"
	operationPrune    = "prune"
)

// AssetPO stores private metadata for one user-owned object lifecycle.
type AssetPO struct {
	ID               string    `gorm:"size:36;primaryKey"`
	UserID           uint      `gorm:"not null;uniqueIndex:idx_assets_user_idempotency,priority:1;index:idx_assets_user_deleted_created,priority:1;index:idx_assets_user_status_deleted_created,priority:1"`
	IdempotencyKey   string    `gorm:"size:128;not null;uniqueIndex:idx_assets_user_idempotency,priority:2"`
	RequestHash      string    `gorm:"size:64;not null"`
	OriginalName     string    `gorm:"size:255;not null"`
	MediaType        string    `gorm:"size:100;not null"`
	SizeBytes        int64     `gorm:"not null"`
	Status           string    `gorm:"size:24;not null;index:idx_assets_user_status_deleted_created,priority:2;index:idx_assets_cleanup,priority:1;check:assets_status_check,status IN ('pending','ready','rejected','deleted')"`
	StagingKey       string    `gorm:"size:255;not null"`
	ObjectKey        string    `gorm:"size:255;not null"`
	ChecksumSHA256   string    `gorm:"size:64;not null;default:''"`
	RejectionCode    string    `gorm:"size:64;not null;default:''"`
	PendingExpiresAt time.Time `gorm:"not null;index:idx_assets_cleanup,priority:2"`
	ReadyAt          *time.Time
	DeletedAt        *time.Time `gorm:"index:idx_assets_user_deleted_created,priority:2;index:idx_assets_user_status_deleted_created,priority:3"`
	OperationKind    string     `gorm:"size:24;not null;default:''"`
	OperationToken   string     `gorm:"size:64;not null;default:''"`
	OperationUntil   *time.Time `gorm:"index"`
	CreatedAt        time.Time  `gorm:"not null;index:idx_assets_user_deleted_created,priority:3,sort:desc;index:idx_assets_user_status_deleted_created,priority:4,sort:desc"`
	UpdatedAt        time.Time  `gorm:"not null"`

	User user.UserPO `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (AssetPO) TableName() string { return "assets" }

func assetFromPO(po *AssetPO) *domain.Asset {
	if po == nil {
		return nil
	}
	return &domain.Asset{
		ID:               po.ID,
		UserID:           po.UserID,
		IdempotencyKey:   po.IdempotencyKey,
		RequestHash:      po.RequestHash,
		OriginalName:     po.OriginalName,
		MediaType:        po.MediaType,
		SizeBytes:        po.SizeBytes,
		Status:           domain.AssetStatus(po.Status),
		StagingKey:       po.StagingKey,
		ObjectKey:        po.ObjectKey,
		ChecksumSHA256:   po.ChecksumSHA256,
		RejectionCode:    po.RejectionCode,
		PendingExpiresAt: po.PendingExpiresAt,
		ReadyAt:          cloneAssetTime(po.ReadyAt),
		DeletedAt:        cloneAssetTime(po.DeletedAt),
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}

func assetToPO(value *domain.Asset) *AssetPO {
	if value == nil {
		return nil
	}
	return &AssetPO{
		ID:               value.ID,
		UserID:           value.UserID,
		IdempotencyKey:   value.IdempotencyKey,
		RequestHash:      value.RequestHash,
		OriginalName:     value.OriginalName,
		MediaType:        value.MediaType,
		SizeBytes:        value.SizeBytes,
		Status:           string(value.Status),
		StagingKey:       value.StagingKey,
		ObjectKey:        value.ObjectKey,
		ChecksumSHA256:   value.ChecksumSHA256,
		RejectionCode:    value.RejectionCode,
		PendingExpiresAt: value.PendingExpiresAt,
		ReadyAt:          cloneAssetTime(value.ReadyAt),
		DeletedAt:        cloneAssetTime(value.DeletedAt),
		CreatedAt:        value.CreatedAt,
		UpdatedAt:        value.UpdatedAt,
	}
}

func cloneAssetTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

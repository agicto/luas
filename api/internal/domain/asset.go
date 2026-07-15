package domain

import (
	"context"
	"time"
)

// AssetStatus is the externally visible lifecycle of one user-owned stored object.
type AssetStatus string

const (
	AssetStatusPending  AssetStatus = "pending"
	AssetStatusReady    AssetStatus = "ready"
	AssetStatusRejected AssetStatus = "rejected"
	AssetStatusDeleted  AssetStatus = "deleted"
)

// Asset owns metadata and lifecycle. Provider keys remain internal and are never HTTP identity.
type Asset struct {
	ID               string
	UserID           uint
	IdempotencyKey   string
	RequestHash      string
	OriginalName     string
	MediaType        string
	SizeBytes        int64
	Status           AssetStatus
	StagingKey       string
	ObjectKey        string
	ChecksumSHA256   string
	RejectionCode    string
	PendingExpiresAt time.Time
	ReadyAt          *time.Time
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AssetReference is the provider-neutral evidence another module may persist as a relation.
type AssetReference struct {
	ID        string
	MediaType string
	SizeBytes int64
}

// AssetReader lets downstream modules require a current, ready, user-owned asset.
type AssetReader interface {
	ReadyForUser(ctx context.Context, userID uint, assetID string) (AssetReference, error)
}

// AssetMaintainer exposes bounded cleanup to the CLI without leaking repository details.
type AssetMaintainer interface {
	Prune(ctx context.Context, limit int) (int, error)
}

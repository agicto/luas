// Package storage defines the provider-neutral object-storage capability.
package storage

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

const (
	DriverLocal = "local"
	DriverR2    = "r2"
)

var (
	ErrInvalidObjectKey          = errors.New("invalid object key")
	ErrObjectNotFound            = errors.New("stored object not found")
	ErrObjectSizeMismatch        = errors.New("stored object size mismatch")
	ErrObjectStoreUnavailable    = errors.New("object storage unavailable")
	ErrDirectTransferUnsupported = errors.New("direct transfer is unsupported")
)

var objectKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)

// ObjectInfo is authoritative metadata returned by a storage provider.
type ObjectInfo struct {
	Size         int64
	MediaType    string
	ETag         string
	LastModified time.Time
}

// TransferGrant is one short-lived provider request. URL is a bearer credential.
type TransferGrant struct {
	Method    string
	URL       string
	Headers   map[string]string
	ExpiresAt time.Time
}

// UploadGrantOptions constrains one direct upload grant.
type UploadGrantOptions struct {
	Key       string
	MediaType string
	TTL       time.Duration
}

// DownloadGrantOptions constrains one attachment download grant.
type DownloadGrantOptions struct {
	Key          string
	MediaType    string
	DownloadName string
	TTL          time.Duration
}

// ObjectStore owns opaque object bytes. Business modules own metadata and lifecycle.
type ObjectStore interface {
	Driver() string
	Put(ctx context.Context, key string, body io.Reader, size int64, mediaType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Copy(ctx context.Context, sourceKey, destinationKey string) error
	Delete(ctx context.Context, key string) error
}

// DirectTransferStore can issue provider-native requests that bypass the API byte path.
type DirectTransferStore interface {
	ObjectStore
	PresignUpload(ctx context.Context, options UploadGrantOptions) (TransferGrant, error)
	PresignDownload(ctx context.Context, options DownloadGrantOptions) (TransferGrant, error)
}

// ValidateObjectKey accepts only canonical application-generated relative keys.
func ValidateObjectKey(key string) error {
	if key == "" || len(key) > 1024 || key != strings.TrimSpace(key) || !objectKeyPattern.MatchString(key) {
		return ErrInvalidObjectKey
	}
	if strings.Contains(key, "//") || strings.Contains(key, `\`) || strings.HasSuffix(key, "/") {
		return ErrInvalidObjectKey
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return ErrInvalidObjectKey
		}
	}
	return nil
}

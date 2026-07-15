package asset

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/domain"
)

var assetIdempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

var allowedAssetExtensions = map[string]map[string]struct{}{
	"image/jpeg":      {".jpg": {}, ".jpeg": {}},
	"image/png":       {".png": {}},
	"image/webp":      {".webp": {}},
	"application/pdf": {".pdf": {}},
	"text/plain":      {".txt": {}},
	"text/csv":        {".csv": {}},
}

func normalizeUploadMetadata(
	idempotencyKey string,
	originalName string,
	mediaType string,
	sizeBytes int64,
	maxBytes int64,
) (string, string, string, error) {
	if !assetIdempotencyKeyPattern.MatchString(idempotencyKey) {
		return "", "", "", domain.ErrInvalidInput
	}
	originalName = strings.TrimSpace(originalName)
	if !validAssetFilename(originalName) {
		return "", "", "", domain.ErrAssetInvalidMediaType
	}
	mediaType = strings.TrimSpace(mediaType)
	extensions, allowed := allowedAssetExtensions[mediaType]
	if !allowed {
		return "", "", "", domain.ErrAssetInvalidMediaType
	}
	extension := strings.ToLower(filepath.Ext(originalName))
	if _, allowed := extensions[extension]; !allowed {
		return "", "", "", domain.ErrAssetInvalidMediaType
	}
	if sizeBytes <= 0 {
		return "", "", "", domain.ErrInvalidInput
	}
	if maxBytes <= 0 || sizeBytes > maxBytes {
		return "", "", "", domain.ErrAssetSizeExceeded
	}
	return idempotencyKey, originalName, mediaType, nil
}

func validAssetFilename(name string) bool {
	if name == "" || len(name) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, `/\\`) {
		return false
	}
	for _, value := range name {
		if value < 0x20 || value == 0x7f {
			return false
		}
	}
	return filepath.Base(name) == name && name != "." && name != ".."
}

func validAssetStatusFilter(status string) bool {
	switch domain.AssetStatus(status) {
	case domain.AssetStatusPending, domain.AssetStatusReady, domain.AssetStatusRejected:
		return true
	default:
		return status == "all"
	}
}

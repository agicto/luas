package asset

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/domain"
)

type contentInspector interface {
	Inspect(context.Context, io.Reader, int64, int64, string) (string, error)
}

// baselineInspector validates a narrow signature allowlist and hashes the bounded stream.
// It is intentionally not described as malware scanning.
type baselineInspector struct{}

func newBaselineInspector() contentInspector { return &baselineInspector{} }

func (*baselineInspector) Inspect(
	ctx context.Context,
	reader io.Reader,
	expectedSize int64,
	maxBytes int64,
	mediaType string,
) (string, error) {
	if reader == nil || expectedSize <= 0 || maxBytes <= 0 {
		return "", domain.ErrInvalidInput
	}
	if expectedSize > maxBytes {
		return "", domain.ErrAssetSizeExceeded
	}

	hash := sha256.New()
	buffer := make([]byte, 32*1024)
	prefix := make([]byte, 0, 512)
	var textValidator *utf8StreamValidator
	if mediaType == "text/plain" || mediaType == "text/csv" {
		textValidator = &utf8StreamValidator{}
	}
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		count, readErr := reader.Read(buffer)
		if count > 0 {
			total += int64(count)
			if total > expectedSize || total > maxBytes {
				return "", domain.ErrAssetSizeExceeded
			}
			chunk := buffer[:count]
			_, _ = hash.Write(chunk)
			if len(prefix) < cap(prefix) {
				remaining := cap(prefix) - len(prefix)
				prefix = append(prefix, chunk[:min(remaining, len(chunk))]...)
			}
			if textValidator != nil && !textValidator.Write(chunk) {
				return "", domain.ErrAssetInvalidMediaType
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return "", domain.ErrServiceUnavailable
			}
			break
		}
	}
	if total != expectedSize {
		return "", domain.ErrAssetSizeExceeded
	}
	if textValidator != nil {
		if !textValidator.Valid() {
			return "", domain.ErrAssetInvalidMediaType
		}
	} else if !matchesAssetSignature(mediaType, prefix) {
		return "", domain.ErrAssetInvalidMediaType
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func matchesAssetSignature(mediaType string, prefix []byte) bool {
	switch mediaType {
	case "image/jpeg":
		return len(prefix) >= 3 && bytes.Equal(prefix[:3], []byte{0xff, 0xd8, 0xff})
	case "image/png":
		return len(prefix) >= 8 && bytes.Equal(prefix[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	case "image/webp":
		return len(prefix) >= 12 && string(prefix[:4]) == "RIFF" && string(prefix[8:12]) == "WEBP"
	case "application/pdf":
		return len(prefix) >= 5 && string(prefix[:5]) == "%PDF-"
	default:
		return false
	}
}

type utf8StreamValidator struct {
	carry []byte
	valid bool
}

func (v *utf8StreamValidator) Write(chunk []byte) bool {
	if v == nil || v.valid && len(chunk) == 0 {
		return v != nil
	}
	if bytes.IndexByte(chunk, 0) >= 0 {
		return false
	}
	data := append(append([]byte(nil), v.carry...), chunk...)
	v.carry = v.carry[:0]
	for len(data) > 0 {
		if !utf8.FullRune(data) {
			v.carry = append(v.carry, data...)
			v.valid = true
			return true
		}
		value, size := utf8.DecodeRune(data)
		if value == utf8.RuneError && size == 1 {
			return false
		}
		data = data[size:]
	}
	v.valid = true
	return true
}

func (v *utf8StreamValidator) Valid() bool {
	return v != nil && v.valid && len(v.carry) == 0
}

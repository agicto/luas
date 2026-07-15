package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestNormalizeUploadMetadataUsesExactAllowlist(t *testing.T) {
	_, name, mediaType, err := normalizeUploadMetadata(
		"upload-1",
		" report.PDF ",
		"application/pdf",
		100,
		1024,
	)
	require.NoError(t, err)
	assert.Equal(t, "report.PDF", name)
	assert.Equal(t, "application/pdf", mediaType)

	tests := []struct {
		name      string
		mediaType string
		want      error
	}{
		{name: "report.svg", mediaType: "image/svg+xml", want: domain.ErrAssetInvalidMediaType},
		{name: "report.pdf.exe", mediaType: "application/pdf", want: domain.ErrAssetInvalidMediaType},
		{name: "../report.pdf", mediaType: "application/pdf", want: domain.ErrAssetInvalidMediaType},
		{name: "report.pdf", mediaType: "application/pdf; charset=binary", want: domain.ErrAssetInvalidMediaType},
	}
	for _, tt := range tests {
		_, _, _, err := normalizeUploadMetadata("upload-2", tt.name, tt.mediaType, 100, 1024)
		assert.ErrorIs(t, err, tt.want)
	}
}

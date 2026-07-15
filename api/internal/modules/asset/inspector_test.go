package asset

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestBaselineInspectorAcceptsSignaturesAndUTF8(t *testing.T) {
	inspector := newBaselineInspector()
	tests := []struct {
		name      string
		mediaType string
		content   []byte
	}{
		{name: "jpeg", mediaType: "image/jpeg", content: []byte{0xff, 0xd8, 0xff, 0xdb, 1}},
		{name: "png", mediaType: "image/png", content: append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, 1)},
		{name: "webp", mediaType: "image/webp", content: []byte("RIFF0000WEBPdata")},
		{name: "pdf", mediaType: "application/pdf", content: []byte("%PDF-1.7")},
		{name: "utf8", mediaType: "text/plain", content: []byte("Dia dhuit, 世界")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, err := inspector.Inspect(
				context.Background(),
				bytes.NewReader(tt.content),
				int64(len(tt.content)),
				1024,
				tt.mediaType,
			)
			require.NoError(t, err)
			assert.Len(t, checksum, 64)
		})
	}
}

func TestBaselineInspectorRejectsSpoofedOrUnsafeContent(t *testing.T) {
	inspector := newBaselineInspector()
	tests := []struct {
		mediaType string
		content   []byte
	}{
		{mediaType: "application/pdf", content: []byte("not a pdf")},
		{mediaType: "text/plain", content: []byte{'a', 0, 'b'}},
		{mediaType: "text/plain", content: []byte{0xff, 0xfe}},
	}
	for _, tt := range tests {
		_, err := inspector.Inspect(
			context.Background(),
			bytes.NewReader(tt.content),
			int64(len(tt.content)),
			1024,
			tt.mediaType,
		)
		assert.ErrorIs(t, err, domain.ErrAssetInvalidMediaType)
	}
}

func TestBaselineInspectorHandlesUTF8SplitAcrossReads(t *testing.T) {
	content := "héllo"
	checksum, err := newBaselineInspector().Inspect(
		context.Background(),
		&oneByteReader{reader: strings.NewReader(content)},
		int64(len(content)),
		1024,
		"text/plain",
	)
	require.NoError(t, err)
	assert.Len(t, checksum, 64)
}

type oneByteReader struct {
	reader *strings.Reader
}

func (r *oneByteReader) Read(buffer []byte) (int, error) {
	return r.reader.Read(buffer[:1])
}

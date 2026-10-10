package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestCursorRoundTripsPositionExactly(t *testing.T) {
	position := &domain.AuditLogCursor{CreatedAt: time.Date(2026, 10, 10, 8, 30, 1, 123456789, time.UTC), ID: 42}
	decoded, err := DecodeCursor(EncodeCursor(position))
	require.NoError(t, err)
	assert.Equal(t, position.ID, decoded.ID)
	assert.True(t, position.CreatedAt.Equal(decoded.CreatedAt))

	empty, err := DecodeCursor("")
	require.NoError(t, err)
	assert.Nil(t, empty)
	assert.Empty(t, EncodeCursor(nil))
}

func TestDecodeCursorRejectsTampering(t *testing.T) {
	for _, token := range []string{"not*base64", "bm9kb3Q", "MTIzLjA", "eC4x", string(make([]byte, 200))} {
		_, err := DecodeCursor(token)
		assert.ErrorIs(t, err, domain.ErrInvalidInput, token)
	}
}

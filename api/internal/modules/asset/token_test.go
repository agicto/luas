package asset

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferSignerRejectsTamperAndExpiry(t *testing.T) {
	signer, err := newTransferSigner(strings.Repeat("k", 64))
	require.NoError(t, err)
	now := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return now }
	id := uuid.NewString()
	token, _, err := signer.Sign("upload", id, time.Minute)
	require.NoError(t, err)
	claims, err := signer.Verify(token, "upload")
	require.NoError(t, err)
	assert.Equal(t, id, claims.AssetID)

	_, err = signer.Verify(token+"x", "upload")
	assert.ErrorIs(t, err, errInvalidTransferToken)
	_, err = signer.Verify(token, "download")
	assert.ErrorIs(t, err, errInvalidTransferToken)
	signer.now = func() time.Time { return now.Add(time.Minute) }
	_, err = signer.Verify(token, "upload")
	assert.ErrorIs(t, err, errInvalidTransferToken)
}

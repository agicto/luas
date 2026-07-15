package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAssetMaintainer struct {
	batch int
	count int
	err   error
}

func (f *fakeAssetMaintainer) Prune(_ context.Context, batch int) (int, error) {
	f.batch = batch
	return f.count, f.err
}

func TestParseAssetPruneArgs(t *testing.T) {
	batch, err := parseAssetPruneArgs([]string{"--batch=25"})
	require.NoError(t, err)
	assert.Equal(t, 25, batch)

	_, err = parseAssetPruneArgs([]string{"--batch=0"})
	assert.Error(t, err)
	_, err = parseAssetPruneArgs([]string{"--unknown"})
	assert.Error(t, err)
}

func TestRunAssetPrune(t *testing.T) {
	maintainer := &fakeAssetMaintainer{count: 7}
	count, err := runAssetPrune(context.Background(), maintainer, 25)
	require.NoError(t, err)
	assert.Equal(t, 7, count)
	assert.Equal(t, 25, maintainer.batch)

	want := errors.New("storage unavailable")
	_, err = runAssetPrune(context.Background(), &fakeAssetMaintainer{err: want}, 10)
	assert.ErrorIs(t, err, want)
	_, err = runAssetPrune(context.Background(), nil, 10)
	assert.ErrorIs(t, err, errAssetMaintainerUnavailable)
}

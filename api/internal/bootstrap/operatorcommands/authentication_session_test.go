package operatorcommands

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuthenticationSessionMaintainer struct {
	batch int
	count int64
	err   error
}

func (f *fakeAuthenticationSessionMaintainer) PruneAuthenticationSessions(
	_ context.Context,
	batch int,
) (int64, error) {
	f.batch = batch
	return f.count, f.err
}

func TestParseAuthenticationSessionPruneArgs(t *testing.T) {
	batch, err := parseAuthenticationSessionPruneArgs([]string{"--batch=250"})
	require.NoError(t, err)
	assert.Equal(t, 250, batch)

	batch, err = parseAuthenticationSessionPruneArgs(nil)
	require.NoError(t, err)
	assert.Equal(t, defaultAuthenticationSessionPruneBatch, batch)

	for _, args := range [][]string{{"--batch=0"}, {"--batch=10001"}, {"--unknown"}, {"--batch"}} {
		_, err = parseAuthenticationSessionPruneArgs(args)
		assert.Error(t, err)
	}
}

func TestRunAuthenticationSessionPrune(t *testing.T) {
	maintainer := &fakeAuthenticationSessionMaintainer{count: 7}
	count, err := runAuthenticationSessionPrune(context.Background(), maintainer, 250)
	require.NoError(t, err)
	assert.Equal(t, int64(7), count)
	assert.Equal(t, 250, maintainer.batch)

	want := errors.New("database unavailable")
	_, err = runAuthenticationSessionPrune(
		context.Background(),
		&fakeAuthenticationSessionMaintainer{err: want},
		10,
	)
	assert.ErrorIs(t, err, want)
	_, err = runAuthenticationSessionPrune(context.Background(), nil, 10)
	assert.ErrorIs(t, err, errAuthenticationSessionMaintainerUnavailable)
}

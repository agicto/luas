package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRateLimitPruneArgs(t *testing.T) {
	batch, maxBatches, err := parseRateLimitPruneArgs(nil)
	require.NoError(t, err)
	assert.Equal(t, defaultRateLimitPruneBatch, batch)
	assert.Equal(t, defaultRateLimitPruneMaxBatches, maxBatches)

	batch, maxBatches, err = parseRateLimitPruneArgs([]string{"--batch=250", "--max-batches=4"})
	require.NoError(t, err)
	assert.Equal(t, 250, batch)
	assert.Equal(t, 4, maxBatches)

	for _, args := range [][]string{
		{"--batch=0"},
		{"--batch=10001"},
		{"--max-batches=0"},
		{"--max-batches=1001"},
		{"--batch", "10"},
		{"--older-than=1h"},
	} {
		_, _, err = parseRateLimitPruneArgs(args)
		assert.Error(t, err, "args %v", args)
	}
}

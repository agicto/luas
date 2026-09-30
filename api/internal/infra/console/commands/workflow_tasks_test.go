package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowTaskCommandArguments(t *testing.T) {
	_, err := taskIDArgument("workflow:retry", nil)
	require.Error(t, err)
	_, err = taskIDArgument("workflow:retry", []string{"--queue=mail"})
	require.Error(t, err)
	id, err := taskIDArgument("workflow:retry", []string{" 7b0c "})
	require.NoError(t, err)
	assert.Equal(t, "7b0c", id)

	limit, err := intFlag([]string{"--limit=20"}, "limit", 50, 200)
	require.NoError(t, err)
	assert.Equal(t, 20, limit)
	limit, err = intFlag(nil, "limit", 50, 200)
	require.NoError(t, err)
	assert.Equal(t, 50, limit)
	_, err = intFlag([]string{"--limit=201"}, "limit", 50, 200)
	require.Error(t, err)
	_, err = intFlag([]string{"--limit=abc"}, "limit", 50, 200)
	require.Error(t, err)
}

func TestWorkflowPruneRejectsTooShortRetention(t *testing.T) {
	err := NewWorkflowPruneCommand().Run([]string{"--older-than=5m"})
	require.ErrorContains(t, err, "at least 1h")
	err = NewWorkflowPruneCommand().Run([]string{"--batch=0"})
	require.ErrorContains(t, err, "--batch")
}

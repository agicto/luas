package operatorcommands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeAuditRecorder struct {
	entry *domain.AuditLog
	err   error
}

func (f *fakeAuditRecorder) Record(_ context.Context, entry *domain.AuditLog) error {
	f.entry = entry
	return f.err
}

func TestParseAuditPruneArguments(t *testing.T) {
	parsed, err := parseAuditPruneArguments([]string{
		"--before=2026-01-01T00:00:00Z",
		"--batch",
		"250",
	})
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), parsed.before)
	assert.Equal(t, 250, parsed.batch)

	parsed, err = parseAuditPruneArguments([]string{"--before", "2026-01-01T00:00:00Z"})
	require.NoError(t, err)
	assert.Equal(t, defaultAuditPruneBatch, parsed.batch)

	invalid := [][]string{
		nil,
		{"--before"},
		{"--before=not-a-time"},
		{"--before=2026-01-01T00:00:00Z", "--batch=0"},
		{"--before=2026-01-01T00:00:00Z", "--batch=10001"},
		{"--before=2026-01-01T00:00:00Z", "--unknown=value"},
		{"--before=2026-01-01T00:00:00Z", "--before=2026-01-02T00:00:00Z"},
	}
	for _, args := range invalid {
		_, err = parseAuditPruneArguments(args)
		assert.Error(t, err)
	}
}

func TestRecordAuditPruneUsesPrivacySafeMetadata(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	parsed := &auditPruneArguments{
		before: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		batch:  500,
	}

	err := recordAuditPrune(context.Background(), recorder, parsed, 37)

	require.NoError(t, err)
	require.NotNil(t, recorder.entry)
	assert.Equal(t, domain.AuditActorSystem, recorder.entry.ActorType)
	assert.Equal(t, "prune", recorder.entry.Action)
	assert.Equal(t, "audit_logs", recorder.entry.Resource)
	assert.Equal(t, "2026-01-01T00:00:00Z", recorder.entry.TargetID)
	assert.Equal(t, 500, recorder.entry.Metadata["batch"])
	assert.Equal(t, int64(37), recorder.entry.Metadata["deleted"])

	want := errors.New("audit unavailable")
	recorder.err = want
	assert.ErrorIs(t, recordAuditPrune(context.Background(), recorder, parsed, 0), want)
	assert.ErrorIs(t, recordAuditPrune(context.Background(), nil, parsed, 0), errAuditMaintainerUnavailable)
}

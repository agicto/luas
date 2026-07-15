package operatorcommands

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/console"
	infracommands "github.com/zgiai/luas/api/internal/infra/console/commands"
)

func TestParseUsageMutationArgumentsRequiresExplicitBoundedInput(t *testing.T) {
	parsed, err := parseUsageMutationArguments("usage:record", []string{
		"--scope=user",
		"--subject-id=42",
		"--metric=api.requests",
		"--quantity=-2",
		"--source=api.gateway",
		"--event-id=request-001-correction",
		"--occurred-at=2026-07-15T12:00:00Z",
		`--dimensions={}`,
	}, true)
	require.NoError(t, err)
	assert.Equal(t, domain.UsageScopeUser, parsed.target.Scope)
	assert.Equal(t, uint(42), parsed.target.SubjectID)
	assert.Equal(t, int64(-2), parsed.quantity)
	assert.Equal(t, time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC), parsed.occurredAt)
	assert.Empty(t, parsed.dimensions)

	_, err = parseUsageMutationArguments("usage:record", []string{
		"--scope=user", "--subject-id=42", "--metric=api.requests", "--quantity=1",
		"--source=api.gateway", "--event-id=request-001",
	}, true)
	assert.Error(t, err)

	_, err = parseUsageMutationArguments("usage:consume", []string{
		"--scope=user", "--subject-id=42", "--metric=api.requests", "--quantity=1",
		"--source=api.gateway", "--event-id=request-001", `--dimensions={} trailing`,
	}, false)
	assert.Error(t, err)
}

func TestParseUsageQuotaArgumentsRequiresCASVersion(t *testing.T) {
	parsed, err := parseUsageQuotaArguments("usage:quota:set", []string{
		"--scope=organization",
		"--subject-id=7",
		"--metric=ai.input_tokens",
		"--limit=1000",
		"--expected-version=0",
	}, true)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), parsed.limit)
	assert.Equal(t, uint64(0), parsed.expectedVersion)

	_, err = parseUsageQuotaArguments("usage:quota:set", []string{
		"--scope=organization", "--subject-id=7", "--metric=ai.input_tokens", "--limit=1000",
	}, true)
	assert.Error(t, err)
	_, err = parseUsageQuotaArguments("usage:quota:set", []string{
		"--scope=organization", "--subject-id=7", "--metric=ai.input_tokens",
		"--limit=-1", "--expected-version=0",
	}, true)
	assert.Error(t, err)
}

func TestRecordUsageOperatorAuditContainsNoEventIdentityOrDimensions(t *testing.T) {
	recorder := &settingAuditRecorder{}
	err := recordUsageOperatorAudit(context.Background(), recorder, "usage:consume", &domain.UsageReceipt{
		Source:   "ai.runtime",
		EventID:  "completion-001",
		Target:   domain.UsageTarget{Scope: domain.UsageScopeOrganization, SubjectID: 7},
		Metric:   "ai.input_tokens",
		Quantity: 8,
		Decision: domain.UsageDecisionDenied,
		Limit:    usageInt64(5),
	}, domain.AuditResultFailure)
	require.NoError(t, err)
	require.NotNil(t, recorder.entry)
	assert.Equal(t, domain.AuditActorSystem, recorder.entry.ActorType)
	assert.Equal(t, "organization:7:ai.input_tokens", recorder.entry.TargetID)
	assert.Equal(t, 429, recorder.entry.StatusCode)
	assert.NotContains(t, recorder.entry.Metadata, "source")
	assert.NotContains(t, recorder.entry.Metadata, "event_id")
	assert.NotContains(t, recorder.entry.Metadata, "dimensions")
}

func TestUsageOperatorManifestRegistersAllCommands(t *testing.T) {
	app := console.New("luas", "test")
	infracommands.RegisterManifest(app, Manifest())

	for _, command := range []string{
		"usage:list",
		"usage:record",
		"usage:consume",
		"usage:quota:set",
		"usage:quota:reset",
		"usage:prune",
	} {
		assert.True(t, app.HasCommand(command), command)
	}
}

func usageInt64(value int64) *int64 { return &value }

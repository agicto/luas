package operatorcommands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/console"
	infracommands "github.com/zgiai/luas/api/internal/infra/console/commands"
)

type settingAuditRecorder struct {
	entry *domain.AuditLog
}

func (r *settingAuditRecorder) Record(_ context.Context, entry *domain.AuditLog) error {
	r.entry = entry
	return nil
}

func TestParseSettingSetArgumentsRequiresExplicitTypedFields(t *testing.T) {
	parsed, err := parseSettingSetArguments([]string{
		"--key=branding.display_name",
		`--value="Luas Cloud"`,
		"--expected-version=2",
	})
	require.NoError(t, err)
	assert.Equal(t, "branding.display_name", parsed.key)
	assert.Equal(t, "Luas Cloud", parsed.value)
	assert.Equal(t, uint64(2), parsed.expectedVersion)

	_, err = parseSettingSetArguments([]string{
		"--key=branding.display_name",
		`--value={"unsafe":true}`,
		"--expected-version=2",
	})
	assert.Error(t, err)
	_, err = parseSettingSetArguments([]string{
		"--key=branding.display_name",
		`--value="Luas"`,
	})
	assert.Error(t, err)
}

func TestParseSettingResetArgumentsRejectsAmbiguousVersions(t *testing.T) {
	parsed, err := parseSettingResetArguments([]string{
		"--key", "localization.locale",
		"--expected-version", "0",
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), parsed.expectedVersion)

	_, err = parseSettingResetArguments([]string{
		"--key=localization.locale",
		"--expected-version=01",
	})
	assert.Error(t, err)
	_, err = parseSettingResetArguments([]string{
		"--key=localization.locale",
		"--expected-version=0",
		"--expected-version=0",
	})
	assert.Error(t, err)
}

func TestDecodeSettingCommandValuePreservesJSONInteger(t *testing.T) {
	value, err := decodeSettingCommandValue("42")
	require.NoError(t, err)
	assert.Equal(t, json.Number("42"), value)

	_, err = decodeSettingCommandValue("null")
	assert.Error(t, err)
	_, err = decodeSettingCommandValue("1 2")
	assert.Error(t, err)
	_, err = decodeSettingCommandValue("1.5")
	assert.Error(t, err)
}

func TestRecordSettingCommandAuditContainsMetadataOnly(t *testing.T) {
	recorder := &settingAuditRecorder{}
	err := recordSettingCommandAudit(
		context.Background(),
		recorder,
		"set",
		"branding.display_name",
		2,
		3,
		domain.SettingSourceOverride,
	)
	require.NoError(t, err)
	require.NotNil(t, recorder.entry)
	assert.Equal(t, domain.AuditActorSystem, recorder.entry.ActorType)
	assert.Equal(t, "app:0:branding.display_name", recorder.entry.TargetID)
	assert.Equal(t, "setting:set", recorder.entry.Path)
	assert.Equal(t, map[string]any{
		"operation":      "set",
		"scope":          domain.SettingScopeApp,
		"key":            "branding.display_name",
		"before_version": uint64(2),
		"after_version":  uint64(3),
		"source":         domain.SettingSourceOverride,
	}, recorder.entry.Metadata)
	assert.NotContains(t, recorder.entry.Metadata, "value")
}

func TestSettingOperatorManifestRegistersAllCommands(t *testing.T) {
	app := console.New("luas", "test")
	infracommands.RegisterManifest(app, Manifest())

	assert.True(t, app.HasCommand("setting:list"))
	assert.True(t, app.HasCommand("setting:set"))
	assert.True(t, app.HasCommand("setting:reset"))
}

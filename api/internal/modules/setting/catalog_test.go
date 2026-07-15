package setting

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestDefaultCatalogOwnsFiniteScopedDefinitions(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)

	assert.Len(t, catalog.Definitions(domain.SettingScopeApp), 2)
	assert.Len(t, catalog.Definitions(domain.SettingScopeOrganization), 1)
	assert.Len(t, catalog.Definitions(domain.SettingScopeUser), 2)
	definition, ok := catalog.Definition(domain.SettingScopeApp, "branding.display_name")
	require.True(t, ok)
	assert.Equal(t, domain.SettingVisibilityPublic, definition.Visibility)
	assert.Equal(t, "Luas", definition.Default)
}

func TestCatalogRejectsDuplicateAndPublicTenantDefinitions(t *testing.T) {
	definition := NewBooleanDefinition(
		domain.SettingScopeApp,
		"features.preview",
		domain.SettingVisibilityPrivate,
		false,
	)
	_, err := NewCatalog(definition, definition)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")

	_, err = NewCatalog(NewBooleanDefinition(
		domain.SettingScopeOrganization,
		"features.preview",
		domain.SettingVisibilityPublic,
		false,
	))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only app")
}

func TestCatalogRequiresNamespacedDottedSemanticKeys(t *testing.T) {
	for _, key := range []string{"single", "feature-flag.enabled", "feature.1", "feature..enabled", "feature.enabled_"} {
		_, err := NewCatalog(NewBooleanDefinition(
			domain.SettingScopeApp,
			key,
			domain.SettingVisibilityPrivate,
			false,
		))
		assert.Error(t, err, key)
	}

	_, err := NewCatalog(NewBooleanDefinition(
		domain.SettingScopeApp,
		"feature_flag.preview_mode1",
		domain.SettingVisibilityPrivate,
		false,
	))
	assert.NoError(t, err)
}

func TestCatalogRejectsMutatedKindAndNonEnumOptions(t *testing.T) {
	definition := NewBooleanDefinition(
		domain.SettingScopeApp,
		"features.preview",
		domain.SettingVisibilityPrivate,
		false,
	)
	definition.Kind = domain.SettingKind("object")
	_, err := NewCatalog(definition)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid kind")

	definition = NewBooleanDefinition(
		domain.SettingScopeApp,
		"features.preview",
		domain.SettingVisibilityPrivate,
		false,
	)
	definition.Options = []string{"unexpected"}
	_, err = NewCatalog(definition)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only enum")
}

func TestCatalogNormalizesOnlyTypedBoundedScalars(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)

	display, ok := catalog.Definition(domain.SettingScopeApp, "branding.display_name")
	require.True(t, ok)
	value, encoded, err := normalizeSettingValue(display, "  Luas Cloud  ")
	require.NoError(t, err)
	assert.Equal(t, "Luas Cloud", value)
	assert.Equal(t, `"Luas Cloud"`, encoded)

	locale, ok := catalog.Definition(domain.SettingScopeUser, "localization.locale")
	require.True(t, ok)
	_, _, err = normalizeSettingValue(locale, "fr-FR")
	assert.ErrorIs(t, err, domain.ErrSettingInvalidValue)
	_, _, err = normalizeSettingValue(locale, map[string]any{"value": "en-US"})
	assert.ErrorIs(t, err, domain.ErrSettingInvalidValue)

	integer := NewIntegerDefinition(
		domain.SettingScopeApp,
		"limits.batch",
		domain.SettingVisibilityPrivate,
		10,
		1,
		100,
	)
	value, encoded, err = normalizeSettingValue(integer, json.Number("42"))
	require.NoError(t, err)
	assert.Equal(t, int64(42), value)
	assert.Equal(t, "42", encoded)
	_, _, err = normalizeSettingValue(integer, json.Number("1.5"))
	assert.ErrorIs(t, err, domain.ErrSettingInvalidValue)
}

func TestCatalogReturnsOwnedEnumOptions(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)
	definition, ok := catalog.Definition(domain.SettingScopeApp, "localization.locale")
	require.True(t, ok)
	definition.Options[0] = "mutated"

	again, ok := catalog.Definition(domain.SettingScopeApp, "localization.locale")
	require.True(t, ok)
	assert.Equal(t, []string{"en-US", "zh-Hans"}, again.Options)
}

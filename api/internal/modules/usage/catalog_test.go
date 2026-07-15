package usage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestDefaultUsageCatalogIsFiniteScopedAndUnlimited(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)

	for _, scope := range []domain.UsageScope{domain.UsageScopeUser, domain.UsageScopeOrganization} {
		definitions := catalog.Definitions(scope)
		require.Len(t, definitions, 5)
		assert.Equal(t, "api.requests", definitions[0].Key)
		assert.Equal(t, "request", definitions[0].Unit)
		assert.Equal(t, domain.UsagePeriodMonth, definitions[0].Period)
		assert.Nil(t, definitions[0].DefaultLimit)
	}
}

func TestUsageCatalogRejectsAmbiguousOrUnboundedDefinitions(t *testing.T) {
	definition := NewDefinition(
		domain.UsageScopeUser,
		"api.requests",
		"request",
		domain.UsagePeriodMonth,
		nil,
		[]DimensionDefinition{{Key: "channel", Values: []string{"api", "worker"}}},
	)
	_, err := NewCatalog(definition, definition)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")

	invalid := definition
	invalid.Key = "requests"
	_, err = NewCatalog(invalid)
	assert.Error(t, err)

	invalid = definition
	invalid.Dimensions[0].Values = nil
	_, err = NewCatalog(invalid)
	assert.Error(t, err)
}

func TestUsageDimensionsRequireExactFiniteSchemaAndOwnedCopies(t *testing.T) {
	definition := NewDefinition(
		domain.UsageScopeUser,
		"api.requests",
		"request",
		domain.UsagePeriodDay,
		nil,
		[]DimensionDefinition{{Key: "channel", Values: []string{"api", "worker"}}},
	)
	catalog, err := NewCatalog(definition)
	require.NoError(t, err)
	definition.Dimensions[0].Values[0] = "mutated"

	stored, ok := catalog.Definition(domain.UsageScopeUser, "api.requests")
	require.True(t, ok)
	assert.Equal(t, []string{"api", "worker"}, stored.Dimensions[0].Values)

	values, encoded, err := normalizeUsageDimensions(stored, map[string]string{"channel": "worker"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"channel": "worker"}, values)
	assert.Equal(t, `{"channel":"worker"}`, encoded)

	_, _, err = normalizeUsageDimensions(stored, map[string]string{"channel": "browser"})
	assert.ErrorIs(t, err, domain.ErrUsageInvalidEvent)
	_, _, err = normalizeUsageDimensions(stored, map[string]string{"channel": "api", "freeform": "x"})
	assert.ErrorIs(t, err, domain.ErrUsageInvalidEvent)
}

func TestUsageProducerIdentifiersUseBoundedSemanticGrammar(t *testing.T) {
	assert.True(t, validUsageSource("workflow.worker"))
	assert.True(t, validUsageEventID("job_01:attempt-2"))
	assert.False(t, validUsageSource("Workflow Worker"))
	assert.False(t, validUsageEventID("event with spaces"))
}

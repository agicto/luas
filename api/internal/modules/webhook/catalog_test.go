package webhook

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestDefaultCatalogNormalizesExactTestPayload(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)
	assert.Equal(t, []string{"webhook.test"}, catalog.Types())

	normalized, err := catalog.Normalize("webhook.test", []byte(` { "endpoint_id": 9, "organization_id": 42 } `))
	require.NoError(t, err)
	assert.Equal(t, `{"endpoint_id":9,"organization_id":42}`, normalized)
}

func TestCatalogRejectsUnknownTypeAndAmbiguousPayloads(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)

	_, err = catalog.Normalize("order.created", []byte(`{}`))
	assert.ErrorIs(t, err, domain.ErrWebhookInvalidEventType)

	invalid := []string{
		`{"organization_id":42,"organization_id":43,"endpoint_id":9}`,
		`{"organization_id":42,"endpoint_id":9,"extra":true}`,
		`{"organization_id":42,"endpoint_id":0}`,
		`[{"organization_id":42,"endpoint_id":9}]`,
		`{"organization_id":42,"endpoint_id":9} {}`,
	}
	for _, raw := range invalid {
		_, normalizeErr := catalog.Normalize("webhook.test", []byte(raw))
		assert.True(t, errors.Is(normalizeErr, errWebhookPayload), raw)
	}
}

func TestNewCatalogRejectsInvalidDefinitions(t *testing.T) {
	_, err := NewCatalog(Definition{Type: "Webhook.Created", ValidatePayload: validateTestPayload})
	assert.Error(t, err)
	_, err = NewCatalog(
		Definition{Type: "webhook.test", ValidatePayload: validateTestPayload},
		Definition{Type: "webhook.test", ValidatePayload: validateTestPayload},
	)
	assert.Error(t, err)
}

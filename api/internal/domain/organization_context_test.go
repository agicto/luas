package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationContextRoundTripUsesAValidatedValueCopy(t *testing.T) {
	resolved := OrganizationContext{
		OrganizationID:   42,
		OrganizationName: "Context Org",
		OrganizationSlug: "context-org",
		MembershipID:     91,
		UserID:           17,
		Role:             OrganizationRoleAdmin,
	}
	ctx := WithOrganizationContext(context.Background(), resolved)
	resolved.OrganizationID = 99

	stored, ok := OrganizationContextFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, uint(42), stored.OrganizationID)
	assert.True(t, stored.IsValid())
}

func TestOrganizationContextFromContextRejectsIncompleteValues(t *testing.T) {
	_, ok := OrganizationContextFromContext(context.Background())
	assert.False(t, ok)

	ctx := WithOrganizationContext(context.Background(), OrganizationContext{
		OrganizationID: 42,
		MembershipID:   91,
		UserID:         17,
		Role:           OrganizationRoleAdmin,
	})
	_, ok = OrganizationContextFromContext(ctx)
	assert.False(t, ok)
}

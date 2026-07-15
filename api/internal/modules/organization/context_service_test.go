package organization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestServiceResolvesOrganizationContextWithExplicitArgumentOrder(t *testing.T) {
	repo := &fakeRepository{
		resolveFn: func(_ context.Context, organizationID, userID uint) (*domain.OrganizationContext, error) {
			assert.Equal(t, uint(42), organizationID)
			assert.Equal(t, uint(17), userID)
			return &domain.OrganizationContext{
				OrganizationID:   organizationID,
				OrganizationName: "Context Org",
				OrganizationSlug: "context-org",
				MembershipID:     91,
				UserID:           userID,
				Role:             domain.OrganizationRoleAdmin,
			}, nil
		},
	}

	resolved, err := newOrganizationService(repo).ResolveContext(context.Background(), 17, 42)
	require.NoError(t, err)
	assert.Equal(t, uint(42), resolved.OrganizationID)
	assert.Equal(t, uint(17), resolved.UserID)
}

func TestServiceRejectsMismatchedOrganizationContext(t *testing.T) {
	repo := &fakeRepository{
		resolveFn: func(context.Context, uint, uint) (*domain.OrganizationContext, error) {
			return &domain.OrganizationContext{
				OrganizationID:   99,
				OrganizationName: "Wrong Org",
				OrganizationSlug: "wrong-org",
				MembershipID:     91,
				UserID:           17,
				Role:             domain.OrganizationRoleAdmin,
			}, nil
		},
	}

	_, err := newOrganizationService(repo).ResolveContext(context.Background(), 17, 42)
	require.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

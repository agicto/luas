package organization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestRepositoryResolvesOrganizationContextInOneQuery(t *testing.T) {
	db, repo, organizationID, memberships := createOrganizationMembershipFixture(t)
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(
		"organization_context:count_queries",
		func(*gorm.DB) { queryCount++ },
	))

	resolved, err := repo.ResolveContext(
		context.Background(),
		organizationID,
		memberships["admin"].UserID,
	)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, 1, queryCount)
	assert.Equal(t, organizationID, resolved.OrganizationID)
	assert.Equal(t, "Member Org", resolved.OrganizationName)
	assert.Equal(t, "member-org", resolved.OrganizationSlug)
	assert.Equal(t, memberships["admin"].ID, resolved.MembershipID)
	assert.Equal(t, memberships["admin"].UserID, resolved.UserID)
	assert.Equal(t, domain.OrganizationRoleAdmin, resolved.Role)
	assert.True(t, resolved.IsValid())

	queryCount = 0
	_, err = repo.ResolveContext(context.Background(), organizationID, 999999)
	require.ErrorIs(t, err, domain.ErrOrganizationNotFound)
	assert.Equal(t, 1, queryCount)
}

func TestRepositoryResolveContextRejectsInvalidInputAndMissingDatabase(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)

	_, err := repo.ResolveContext(context.Background(), 0, 1)
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = NewRepository(nil).ResolveContext(context.Background(), 1, 1)
	require.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

package setting

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestRepositoryCompareAndSwapRetainsMonotonicResetHistory(t *testing.T) {
	fixture := newSettingTestFixture(t)
	target := domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID}
	ctx := context.Background()

	created, changed, err := fixture.repository.Set(ctx, target, "localization.locale", `"zh-Hans"`, 0)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, uint64(1), created.Version)

	_, _, err = fixture.repository.Set(ctx, target, "localization.locale", `"en-US"`, 0)
	assert.ErrorIs(t, err, domain.ErrSettingVersionConflict)

	unchanged, changed, err := fixture.repository.Set(ctx, target, "localization.locale", `"zh-Hans"`, 1)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, uint64(1), unchanged.Version)

	version, changed, err := fixture.repository.Reset(ctx, target, "localization.locale", 1)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, uint64(2), version)

	_, _, err = fixture.repository.Set(ctx, target, "localization.locale", `"en-US"`, 1)
	assert.ErrorIs(t, err, domain.ErrSettingVersionConflict)

	created, changed, err = fixture.repository.Set(ctx, target, "localization.locale", `"en-US"`, 2)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, uint64(3), created.Version)
}

func TestRepositoryIsolatesSubjectsAndScopes(t *testing.T) {
	fixture := newSettingTestFixture(t)
	other := &user.UserPO{
		Username: "other-setting-user",
		Email:    "other-setting-user@example.test",
		Password: "hashed-password",
		Status:   1,
	}
	require.NoError(t, fixture.db.Create(other).Error)
	ctx := context.Background()
	key := "localization.locale"

	_, _, err := fixture.repository.Set(ctx, domain.SettingTarget{
		Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID,
	}, key, `"zh-Hans"`, 0)
	require.NoError(t, err)
	_, _, err = fixture.repository.Set(ctx, domain.SettingTarget{
		Scope: domain.SettingScopeOrganization, SubjectID: fixture.organization.ID,
	}, key, `"zh-Hans"`, 0)
	require.NoError(t, err)

	otherRows, err := fixture.repository.List(ctx, domain.SettingTarget{
		Scope: domain.SettingScopeUser, SubjectID: other.ID,
	}, []string{key})
	require.NoError(t, err)
	assert.Empty(t, otherRows)

	userRows, err := fixture.repository.List(ctx, domain.SettingTarget{
		Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID,
	}, []string{key})
	require.NoError(t, err)
	organizationRows, err := fixture.repository.List(ctx, domain.SettingTarget{
		Scope: domain.SettingScopeOrganization, SubjectID: fixture.organization.ID,
	}, []string{key})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), userRows[key].Version)
	assert.Equal(t, uint64(1), organizationRows[key].Version)
}

func TestSettingTableRejectsInvalidHistoryState(t *testing.T) {
	fixture := newSettingTestFixture(t)

	invalidRows := []*SettingPO{
		{
			Scope:        string(domain.SettingScopeApp),
			Key:          "localization.locale",
			ValueJSON:    `"en-US"`,
			IsOverridden: true,
			Version:      0,
		},
		{
			Scope:        string(domain.SettingScopeApp),
			Key:          "localization.locale",
			ValueJSON:    `"en-US"`,
			IsOverridden: false,
			Version:      1,
		},
		{
			Scope:        string(domain.SettingScopeApp),
			Key:          "localization.locale",
			ValueJSON:    "",
			IsOverridden: true,
			Version:      1,
		},
	}

	for _, row := range invalidRows {
		assert.Error(t, fixture.db.Create(row).Error)
	}
}

package setting

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type accountDeletionFailure struct {
	err error
}

func (c *accountDeletionFailure) AccountDeletionCleanerName() string { return "failure" }
func (c *accountDeletionFailure) CleanAccountDeletion(context.Context, uint) error {
	return c.err
}

type accountDeletionBlocker struct {
	err error
}

func (g *accountDeletionBlocker) AccountDeletionGuardName() string { return "blocker" }
func (g *accountDeletionBlocker) CheckAccountDeletion(context.Context, uint) error {
	return g.err
}

func TestServiceResolvesDefaultsOverridesAndResetTombstones(t *testing.T) {
	fixture := newSettingTestFixture(t)
	target := domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID}
	ctx := context.Background()

	values, err := fixture.service.ListSettings(ctx, target)
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.Equal(t, uint64(0), values[0].Version)
	assert.Equal(t, domain.SettingSourceDefault, values[0].Source)

	updated, err := fixture.service.SetSetting(ctx, target, "localization.locale", "zh-Hans", 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), updated.Version)
	assert.Equal(t, domain.SettingSourceOverride, updated.Source)
	assert.Equal(t, "zh-Hans", updated.Value)

	version, err := fixture.service.ResetSetting(ctx, target, "localization.locale", 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), version)
	reset, err := fixture.service.GetSetting(ctx, target, "localization.locale")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), reset.Version)
	assert.Equal(t, domain.SettingSourceDefault, reset.Source)
	assert.Equal(t, "en-US", reset.Value)
	require.NotNil(t, reset.UpdatedAt)
}

func TestServiceFailsClosedForCorruptStoredValue(t *testing.T) {
	fixture := newSettingTestFixture(t)
	target := domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID}
	_, err := fixture.service.SetSetting(context.Background(), target, "localization.locale", "zh-Hans", 0)
	require.NoError(t, err)
	require.NoError(t, fixture.db.Model(&SettingPO{}).
		Where("scope = ? AND subject_id = ?", target.Scope, target.SubjectID).
		Update("value_json", `true`).Error)

	_, err = fixture.service.GetSetting(context.Background(), target, "localization.locale")
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

func TestAccountDeletionCleansUserSettingsInsideTransaction(t *testing.T) {
	fixture := newSettingTestFixture(t)
	target := domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID}
	_, err := fixture.service.SetSetting(context.Background(), target, "localization.locale", "zh-Hans", 0)
	require.NoError(t, err)
	policy := user.NewAccountDeletionPolicy()
	require.NoError(t, policy.RegisterCleaner(fixture.service))

	userRepository := user.NewRepository(fixture.db)
	err = userRepository.DeleteAccount(context.Background(), fixture.user.ID, func(ctx context.Context) error {
		return policy.Prepare(ctx, fixture.user.ID)
	})
	require.NoError(t, err)

	var settings int64
	require.NoError(t, fixture.db.Model(&SettingPO{}).Count(&settings).Error)
	assert.Zero(t, settings)
	var activeUsers int64
	require.NoError(t, fixture.db.Model(&user.UserPO{}).Where("id = ?", fixture.user.ID).Count(&activeUsers).Error)
	assert.Zero(t, activeUsers)
}

func TestAccountDeletionRollsBackSettingCleanupWhenLaterCleanerFails(t *testing.T) {
	fixture := newSettingTestFixture(t)
	target := domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID}
	_, err := fixture.service.SetSetting(context.Background(), target, "localization.locale", "zh-Hans", 0)
	require.NoError(t, err)
	policy := user.NewAccountDeletionPolicy()
	require.NoError(t, policy.RegisterCleaner(fixture.service))
	failure := errors.New("downstream cleanup failed")
	require.NoError(t, policy.RegisterCleaner(&accountDeletionFailure{err: failure}))

	userRepository := user.NewRepository(fixture.db)
	err = userRepository.DeleteAccount(context.Background(), fixture.user.ID, func(ctx context.Context) error {
		return policy.Prepare(ctx, fixture.user.ID)
	})
	require.ErrorIs(t, err, failure)

	var settings int64
	require.NoError(t, fixture.db.Model(&SettingPO{}).Count(&settings).Error)
	assert.EqualValues(t, 1, settings)
	var activeUsers int64
	require.NoError(t, fixture.db.Model(&user.UserPO{}).Where("id = ?", fixture.user.ID).Count(&activeUsers).Error)
	assert.EqualValues(t, 1, activeUsers)
}

func TestAccountDeletionGuardFailureLeavesSettingsUntouched(t *testing.T) {
	fixture := newSettingTestFixture(t)
	target := domain.SettingTarget{Scope: domain.SettingScopeUser, SubjectID: fixture.user.ID}
	_, err := fixture.service.SetSetting(context.Background(), target, "localization.locale", "zh-Hans", 0)
	require.NoError(t, err)
	policy := user.NewAccountDeletionPolicy()
	blocked := errors.New("deletion blocked")
	require.NoError(t, policy.Register(&accountDeletionBlocker{err: blocked}))
	require.NoError(t, policy.RegisterCleaner(fixture.service))

	userRepository := user.NewRepository(fixture.db)
	err = userRepository.DeleteAccount(context.Background(), fixture.user.ID, func(ctx context.Context) error {
		return policy.Prepare(ctx, fixture.user.ID)
	})
	require.ErrorIs(t, err, blocked)

	var settings int64
	require.NoError(t, fixture.db.Model(&SettingPO{}).Count(&settings).Error)
	assert.EqualValues(t, 1, settings)
}

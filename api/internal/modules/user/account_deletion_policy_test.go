package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deletionGuardStub struct {
	name  string
	err   error
	calls int
}

type deletionCleanerStub struct {
	name  string
	err   error
	calls int
	order *[]string
}

func (c *deletionCleanerStub) AccountDeletionCleanerName() string {
	return c.name
}

func (c *deletionCleanerStub) CleanAccountDeletion(context.Context, uint) error {
	c.calls++
	if c.order != nil {
		*c.order = append(*c.order, c.name)
	}
	return c.err
}

func (g *deletionGuardStub) AccountDeletionGuardName() string {
	return g.name
}

func (g *deletionGuardStub) CheckAccountDeletion(context.Context, uint) error {
	g.calls++
	return g.err
}

func TestAccountDeletionPolicyRunsRegisteredGuardsInOrder(t *testing.T) {
	policy := NewAccountDeletionPolicy()
	first := &deletionGuardStub{name: "first"}
	blocked := errors.New("blocked")
	second := &deletionGuardStub{name: "second", err: blocked}
	third := &deletionGuardStub{name: "third"}

	require.NoError(t, policy.Register(first))
	require.NoError(t, policy.Register(second))
	require.NoError(t, policy.Register(third))

	err := policy.Check(context.Background(), 7)
	require.ErrorIs(t, err, blocked)
	assert.Equal(t, 1, first.calls)
	assert.Equal(t, 1, second.calls)
	assert.Zero(t, third.calls)
}

func TestAccountDeletionPolicyRejectsDuplicateGuardNames(t *testing.T) {
	policy := NewAccountDeletionPolicy()
	require.NoError(t, policy.Register(&deletionGuardStub{name: "organization"}))

	err := policy.Register(&deletionGuardStub{name: "organization"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestAccountDeletionPolicyRejectsTypedNilGuard(t *testing.T) {
	policy := NewAccountDeletionPolicy()
	var guard *deletionGuardStub

	assert.NotPanics(t, func() {
		err := policy.Register(guard)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})
}

func TestAccountDeletionPolicyRunsNoCleanerWhenGuardBlocks(t *testing.T) {
	policy := NewAccountDeletionPolicy()
	blocked := errors.New("blocked")
	guard := &deletionGuardStub{name: "guard", err: blocked}
	cleaner := &deletionCleanerStub{name: "cleaner"}
	require.NoError(t, policy.Register(guard))
	require.NoError(t, policy.RegisterCleaner(cleaner))

	err := policy.Prepare(context.Background(), 7)
	require.ErrorIs(t, err, blocked)
	assert.Equal(t, 1, guard.calls)
	assert.Zero(t, cleaner.calls)
}

func TestAccountDeletionPolicyRunsCleanersInOrderAndStopsOnFailure(t *testing.T) {
	policy := NewAccountDeletionPolicy()
	order := []string{}
	failure := errors.New("cleanup failed")
	first := &deletionCleanerStub{name: "first", order: &order}
	second := &deletionCleanerStub{name: "second", err: failure, order: &order}
	third := &deletionCleanerStub{name: "third", order: &order}
	require.NoError(t, policy.RegisterCleaner(first))
	require.NoError(t, policy.RegisterCleaner(second))
	require.NoError(t, policy.RegisterCleaner(third))

	err := policy.Prepare(context.Background(), 7)
	require.ErrorIs(t, err, failure)
	assert.Equal(t, []string{"first", "second"}, order)
	assert.Zero(t, third.calls)
}

func TestAccountDeletionPolicyRejectsDuplicateAndTypedNilCleaners(t *testing.T) {
	policy := NewAccountDeletionPolicy()
	require.NoError(t, policy.RegisterCleaner(&deletionCleanerStub{name: "setting"}))

	err := policy.RegisterCleaner(&deletionCleanerStub{name: "setting"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	var cleaner *deletionCleanerStub
	assert.NotPanics(t, func() {
		err = policy.RegisterCleaner(cleaner)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})
}

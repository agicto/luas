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

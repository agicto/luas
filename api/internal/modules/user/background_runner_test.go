package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/events"
)

func TestBackgroundRunnerOutlivesTheRequestAndKeepsItsValues(t *testing.T) {
	type contextKey string
	runner := newBackgroundRunner("test", 1, time.Second)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey("request"), "req_1"))

	type observation struct {
		err         error
		value       any
		hasDeadline bool
	}
	done := make(chan observation, 1)
	release := make(chan struct{})
	runner.run(ctx, func(taskCtx context.Context) {
		<-release
		_, hasDeadline := taskCtx.Deadline()
		done <- observation{err: taskCtx.Err(), value: taskCtx.Value(contextKey("request")), hasDeadline: hasDeadline}
	})
	cancel()
	close(release)

	seen := <-done
	assert.NoError(t, seen.err, "request cancellation must not cancel the task")
	assert.Equal(t, "req_1", seen.value)
	assert.True(t, seen.hasDeadline)
}

func TestBackgroundRunnerDropsWorkBeyondItsBound(t *testing.T) {
	runner := newBackgroundRunner("test", 1, time.Second)
	release := make(chan struct{})
	started := make(chan struct{})
	runner.run(context.Background(), func(context.Context) {
		close(started)
		<-release
	})
	<-started

	ran := false
	runner.run(context.Background(), func(context.Context) { ran = true })
	close(release)

	require.Eventually(t, func() bool { return len(runner.slots) == 0 }, time.Second, time.Millisecond)
	assert.False(t, ran)
}

func TestServiceRequestPasswordResetReturnsBeforeTheAccountLookup(t *testing.T) {
	lookupStarted := make(chan struct{})
	release := make(chan struct{})
	repo := &fakeRepo{
		findByEmailFn: func(context.Context, string) (*domain.User, error) {
			close(lookupStarted)
			<-release
			return nil, gorm.ErrRecordNotFound
		},
	}
	svc := NewService(repo, repo, &fakeSessionIssuer{}, events.NewEventBus(), &fakeUserMailer{}, NewAccountDeletionPolicy())

	require.NoError(t, svc.RequestPasswordReset(context.Background(), &UserPasswordResetRequest{Email: "a@example.test"}))
	<-lookupStarted // the lookup runs, but only after the request has already returned
	close(release)
}

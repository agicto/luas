package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDriverOperatorTaskManagement(t *testing.T) {
	db := openWorkflowPostgres(t)
	driver, err := NewPostgresDriver(db)
	require.NoError(t, err)
	ctx := context.Background()

	failedID, err := driver.PushTask(ctx, "mail", workflowTestPayload(t, uuid.NewString(), ""))
	require.NoError(t, err)
	pendingID, err := driver.PushTask(ctx, "mail", workflowTestPayload(t, uuid.NewString(), ""))
	require.NoError(t, err)
	otherQueueID, err := driver.PushTask(ctx, "reports", workflowTestPayload(t, uuid.NewString(), ""))
	require.NoError(t, err)

	longAgo := time.Now().UTC().Add(-60 * 24 * time.Hour)
	require.NoError(t, db.Model(&TaskPO{}).Where("id = ?", failedID).Updates(map[string]any{
		"status": taskStatusFailed, "attempts": 3, "failed_at": longAgo,
		"last_failure_code": "handler_error", "updated_at": longAgo,
	}).Error)

	t.Run("lists by queue and status without payloads", func(t *testing.T) {
		failed, err := driver.ListTasks(ctx, "mail", taskStatusFailed, 10)
		require.NoError(t, err)
		require.Len(t, failed, 1)
		assert.Equal(t, failedID, failed[0].ID)
		assert.Equal(t, "handler_error", failed[0].LastFailureCode)
		assert.NotEmpty(t, failed[0].PayloadHash)

		all, err := driver.ListTasks(ctx, "", "", 10)
		require.NoError(t, err)
		assert.Len(t, all, 3)

		_, err = driver.ListTasks(ctx, "mail", "exploded", 10)
		require.Error(t, err)
		_, err = driver.ListTasks(ctx, "mail", "", 201)
		require.Error(t, err)
	})

	t.Run("retries only failed tasks with a fresh attempt budget", func(t *testing.T) {
		require.ErrorIs(t, driver.RetryFailed(ctx, pendingID), ErrTaskNotRetryable)
		require.ErrorIs(t, driver.RetryFailed(ctx, uuid.NewString()), ErrTaskNotFound)

		require.NoError(t, driver.RetryFailed(ctx, failedID))
		var task TaskPO
		require.NoError(t, db.First(&task, "id = ?", failedID).Error)
		assert.Equal(t, taskStatusPending, task.Status)
		assert.Zero(t, task.Attempts)
		assert.Nil(t, task.FailedAt)
		assert.WithinDuration(t, time.Now(), task.AvailableAt, time.Minute)

		require.ErrorIs(t, driver.RetryFailed(ctx, failedID), ErrTaskNotRetryable, "a retried task is no longer failed")

		claimed := map[string]bool{}
		for range 2 {
			claim, claimErr := driver.Claim(ctx, "mail", time.Minute)
			require.NoError(t, claimErr)
			require.NotNil(t, claim)
			claimed[claim.ID] = true
			require.NoError(t, driver.Complete(ctx, claim))
		}
		assert.True(t, claimed[failedID], "a worker must pick up the retried task")
	})

	t.Run("prunes only finished tasks older than the cutoff", func(t *testing.T) {
		require.NoError(t, db.Model(&TaskPO{}).Where("id = ?", otherQueueID).Updates(map[string]any{
			"status": taskStatusCompleted, "completed_at": longAgo, "updated_at": longAgo,
		}).Error)

		deleted, err := driver.PruneFinished(ctx, time.Now().Add(-30*24*time.Hour), 100)
		require.NoError(t, err)
		assert.EqualValues(t, 1, deleted, "only the old completed task qualifies")

		remaining, err := driver.ListTasks(ctx, "", "", 10)
		require.NoError(t, err)
		ids := []string{}
		for _, task := range remaining {
			ids = append(ids, task.ID)
		}
		assert.ElementsMatch(t, []string{failedID, pendingID}, ids)

		_, err = driver.PruneFinished(ctx, time.Now(), 0)
		require.Error(t, err)
	})
}

func TestPostgresDriverStatsHandlesAnEmptyQueue(t *testing.T) {
	db := openWorkflowPostgres(t)
	driver, err := NewPostgresDriver(db)
	require.NoError(t, err)
	ctx := context.Background()

	stats, err := driver.Stats(ctx, "empty", time.Now())
	require.NoError(t, err, "min() over no pending rows is NULL and must not fail")
	assert.Zero(t, stats.Lag)
	assert.Zero(t, stats.Pending)

	_, err = driver.PushTask(ctx, "busy", workflowTestPayload(t, uuid.NewString(), ""))
	require.NoError(t, err)
	stats, err = driver.Stats(ctx, "busy", time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.EqualValues(t, 1, stats.Pending)
	assert.Greater(t, stats.Lag, 50*time.Second)
}

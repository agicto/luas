package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrTaskNotRetryable indicates that a durable task exists but is not in the failed state.
var ErrTaskNotRetryable = errors.New("workflow task is not failed")

const (
	maxTaskListLimit  = 200
	maxTaskPruneBatch = 10000
)

// TaskSummary is the operator view of one durable task. Payloads are never exposed because they
// can carry application data; the payload hash identifies duplicates.
type TaskSummary struct {
	ID              string
	Queue           string
	Status          string
	Attempts        int
	MaxAttempts     int
	LastFailureCode string
	PayloadHash     string
	CreatedAt       time.Time
	AvailableAt     time.Time
	CompletedAt     *time.Time
	FailedAt        *time.Time
}

// ListTasks returns at most limit tasks in one queue, newest first. An empty status lists every
// status; an empty queue lists every queue.
func (d *PostgresDriver) ListTasks(ctx context.Context, queue, status string, limit int) ([]TaskSummary, error) {
	if limit < 1 || limit > maxTaskListLimit {
		return nil, fmt.Errorf("limit must be between 1 and %d", maxTaskListLimit)
	}
	switch status {
	case "", taskStatusPending, taskStatusProcessing, taskStatusCompleted, taskStatusFailed, taskStatusCanceled:
	default:
		return nil, fmt.Errorf("unknown task status %q", status)
	}
	query := d.db.WithContext(ctx).Model(&TaskPO{})
	if queue != "" {
		query = query.Where("queue = ?", queue)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var rows []TaskPO
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	summaries := make([]TaskSummary, len(rows))
	for index, row := range rows {
		summaries[index] = TaskSummary{
			ID:              row.ID,
			Queue:           row.Queue,
			Status:          row.Status,
			Attempts:        row.Attempts,
			MaxAttempts:     row.MaxAttempts,
			LastFailureCode: row.LastFailureCode,
			PayloadHash:     row.PayloadHash,
			CreatedAt:       row.CreatedAt,
			AvailableAt:     row.AvailableAt,
			CompletedAt:     row.CompletedAt,
			FailedAt:        row.FailedAt,
		}
	}
	return summaries, nil
}

// RetryFailed returns one failed task to pending with a fresh attempt budget. The failure code is
// kept for diagnosis until the task succeeds or fails again. Only failed tasks can be retried.
func (d *PostgresDriver) RetryFailed(ctx context.Context, taskID string) error {
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ? AND status = ?", taskID, taskStatusFailed).
		Updates(map[string]any{
			"status":           taskStatusPending,
			"attempts":         0,
			"available_at":     now,
			"failed_at":        nil,
			"lease_token":      "",
			"lease_expires_at": nil,
			"updated_at":       now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var count int64
	if err := d.db.WithContext(ctx).Model(&TaskPO{}).Where("id = ?", taskID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrTaskNotFound
	}
	return ErrTaskNotRetryable
}

// PruneFinished deletes at most batch completed, failed, or canceled tasks that finished before the
// cutoff, oldest first. Pending and processing tasks are never pruned.
func (d *PostgresDriver) PruneFinished(ctx context.Context, before time.Time, batch int) (int64, error) {
	if batch < 1 || batch > maxTaskPruneBatch {
		return 0, fmt.Errorf("batch must be between 1 and %d", maxTaskPruneBatch)
	}
	result := d.db.WithContext(ctx).Exec(`
		DELETE FROM workflow_tasks
		WHERE id IN (
			SELECT id FROM workflow_tasks
			WHERE status IN (?, ?, ?) AND updated_at < ?
			ORDER BY updated_at ASC, id ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)`, taskStatusCompleted, taskStatusFailed, taskStatusCanceled, before.UTC(), batch)
	return result.RowsAffected, result.Error
}

package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	taskStatusPending    = "pending"
	taskStatusProcessing = "processing"
	taskStatusCompleted  = "completed"
	taskStatusFailed     = "failed"
	taskStatusCanceled   = "canceled"
)

// TaskPO is the durable execution ledger for provider-neutral workflow jobs.
type TaskPO struct {
	ID                string     `gorm:"type:uuid;primaryKey"`
	Queue             string     `gorm:"size:128;not null;index:idx_workflow_tasks_claim,priority:1"`
	IdempotencyKey    *string    `gorm:"size:128"`
	Payload           []byte     `gorm:"type:jsonb;not null"`
	PayloadHash       string     `gorm:"size:64;not null"`
	Status            string     `gorm:"size:20;not null;index:idx_workflow_tasks_claim,priority:2;index:idx_workflow_tasks_lease,priority:1;check:workflow_tasks_status_check,status IN ('pending','processing','completed','failed','canceled')"`
	Attempts          int        `gorm:"not null;default:0"`
	MaxAttempts       int        `gorm:"not null"`
	AvailableAt       time.Time  `gorm:"not null;index:idx_workflow_tasks_claim,priority:3"`
	LeaseToken        string     `gorm:"size:64;not null;default:''"`
	LeaseExpiresAt    *time.Time `gorm:"index:idx_workflow_tasks_lease,priority:2"`
	FencingToken      uint64     `gorm:"not null;default:0"`
	CancelRequestedAt *time.Time
	CompletedAt       *time.Time
	FailedAt          *time.Time
	LastFailureCode   string    `gorm:"size:64;not null;default:''"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

func (TaskPO) TableName() string { return "workflow_tasks" }

// DurableClaim identifies one fenced ownership interval for a task.
type DurableClaim struct {
	ID           string
	Payload      []byte
	LeaseToken   string
	FencingToken uint64
	Attempts     int
}

// QueueStats is a bounded-cardinality snapshot for worker observability.
type QueueStats struct {
	Pending    int64
	Processing int64
	Failed     int64
	Lag        time.Duration
}

// DurableDriver adds acknowledgement and lease behavior to a queue driver.
type DurableDriver interface {
	Driver
	Claim(ctx context.Context, queue string, leaseDuration time.Duration) (*DurableClaim, error)
	Complete(ctx context.Context, claim *DurableClaim) error
	Retry(ctx context.Context, claim *DurableClaim, payload []byte, availableAt time.Time, cause error) error
	Fail(ctx context.Context, claim *DurableClaim, cause error) error
	Heartbeat(ctx context.Context, claim *DurableClaim, leaseDuration time.Duration) (bool, error)
	Cancel(ctx context.Context, taskID string) error
	Stats(ctx context.Context, queue string, now time.Time) (QueueStats, error)
}

// PostgresDriver provides durable, multi-replica-safe task execution.
type PostgresDriver struct {
	db     *gorm.DB
	mu     sync.RWMutex
	closed bool
}

func NewPostgresDriver(db *gorm.DB) (*PostgresDriver, error) {
	if db == nil {
		return nil, fmt.Errorf("postgres workflow driver requires a database")
	}
	return &PostgresDriver{db: db}, nil
}

func (d *PostgresDriver) Push(ctx context.Context, queue string, payload []byte) error {
	_, err := d.PushTask(ctx, queue, payload)
	return err
}

func (d *PostgresDriver) PushDelayed(ctx context.Context, queue string, payload []byte, delay time.Duration) error {
	_, err := d.PushTaskDelayed(ctx, queue, payload, delay)
	return err
}

func (d *PostgresDriver) PushTask(ctx context.Context, queue string, payload []byte) (string, error) {
	return d.insert(ctx, queue, payload, time.Now().UTC())
}

func (d *PostgresDriver) PushTaskDelayed(ctx context.Context, queue string, payload []byte, delay time.Duration) (string, error) {
	if delay < 0 {
		delay = 0
	}
	return d.insert(ctx, queue, payload, time.Now().UTC().Add(delay))
}

func (d *PostgresDriver) insert(ctx context.Context, queue string, payload []byte, availableAt time.Time) (string, error) {
	if err := d.ensureOpen(); err != nil {
		return "", err
	}
	queue = strings.TrimSpace(queue)
	if queue == "" || len(queue) > 128 {
		return "", fmt.Errorf("workflow queue name is invalid")
	}
	var envelope JobPayload
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode workflow payload: %w", err)
	}
	if envelope.ID == "" {
		envelope.ID = uuid.NewString()
		encoded, marshalErr := json.Marshal(envelope)
		if marshalErr != nil {
			return "", fmt.Errorf("encode workflow payload: %w", marshalErr)
		}
		payload = encoded
	}
	if envelope.MaxRetries < 0 {
		return "", fmt.Errorf("workflow max retries cannot be negative")
	}
	hash, err := taskFingerprint(envelope)
	if err != nil {
		return "", err
	}
	row := TaskPO{
		ID: envelope.ID, Queue: queue, Payload: append([]byte(nil), payload...),
		PayloadHash: hex.EncodeToString(hash[:]), Status: taskStatusPending,
		MaxAttempts: max(1, envelope.MaxRetries), AvailableAt: availableAt,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if key := strings.TrimSpace(envelope.IdempotencyKey); key != "" {
		if len(key) > 128 {
			return "", fmt.Errorf("workflow idempotency key is too long")
		}
		row.IdempotencyKey = &key
	}
	result := d.db.WithContext(ctx).Create(&row)
	if result.Error == nil {
		return row.ID, nil
	}
	if row.IdempotencyKey == nil {
		return "", fmt.Errorf("insert workflow task: %w", result.Error)
	}
	var existing TaskPO
	findErr := d.db.WithContext(ctx).
		Where("queue = ? AND idempotency_key = ?", queue, *row.IdempotencyKey).
		Take(&existing).Error
	if findErr == nil && existing.PayloadHash == row.PayloadHash {
		return existing.ID, nil
	}
	if findErr == nil {
		return "", ErrIdempotencyConflict
	}
	return "", fmt.Errorf("insert workflow task: %w", result.Error)
}

// Pop is intentionally unsupported because durable work must be acknowledged.
func (d *PostgresDriver) Pop(context.Context, string) ([]byte, error) {
	return nil, fmt.Errorf("postgres workflow driver requires lease-aware worker claims")
}

func (d *PostgresDriver) Claim(ctx context.Context, queue string, leaseDuration time.Duration) (*DurableClaim, error) {
	if err := d.ensureOpen(); err != nil {
		return nil, err
	}
	if leaseDuration <= 0 {
		return nil, fmt.Errorf("workflow lease duration must be positive")
	}
	now := time.Now().UTC()
	var claimed *DurableClaim
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&TaskPO{}).
			Where("queue = ? AND status = ? AND lease_expires_at <= ? AND cancel_requested_at IS NOT NULL", queue, taskStatusProcessing, now).
			Updates(map[string]any{"status": taskStatusCanceled, "lease_token": "", "lease_expires_at": nil, "updated_at": now}).Error; err != nil {
			return fmt.Errorf("recover canceled workflow tasks: %w", err)
		}
		if err := tx.Model(&TaskPO{}).
			Where("queue = ? AND status = ? AND lease_expires_at <= ? AND attempts >= max_attempts", queue, taskStatusProcessing, now).
			Updates(map[string]any{"status": taskStatusFailed, "lease_token": "", "lease_expires_at": nil, "failed_at": now, "last_failure_code": "lease_expired", "updated_at": now}).Error; err != nil {
			return fmt.Errorf("recover exhausted workflow tasks: %w", err)
		}
		var row TaskPO
		err := tx.Where("queue = ? AND cancel_requested_at IS NULL AND attempts < max_attempts", queue).
			Where("(status = ? AND available_at <= ?) OR (status = ? AND lease_expires_at <= ?)", taskStatusPending, now, taskStatusProcessing, now).
			Order("available_at ASC, created_at ASC, id ASC").
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("find claimable workflow task: %w", err)
		}
		leaseToken := uuid.NewString()
		expiresAt := now.Add(leaseDuration)
		result := tx.Model(&TaskPO{}).
			Where("id = ? AND fencing_token = ?", row.ID, row.FencingToken).
			Updates(map[string]any{
				"status": taskStatusProcessing, "attempts": gorm.Expr("attempts + 1"),
				"lease_token": leaseToken, "lease_expires_at": expiresAt,
				"fencing_token": gorm.Expr("fencing_token + 1"), "updated_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("claim workflow task: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrLeaseLost
		}
		claimed = &DurableClaim{ID: row.ID, Payload: append([]byte(nil), row.Payload...), LeaseToken: leaseToken, FencingToken: row.FencingToken + 1, Attempts: row.Attempts + 1}
		return nil
	})
	if err == nil && claimed == nil {
		return nil, ErrQueueEmpty
	}
	return claimed, err
}

func (d *PostgresDriver) Complete(ctx context.Context, claim *DurableClaim) error {
	return d.finish(ctx, claim, taskStatusCompleted, "", nil)
}

func (d *PostgresDriver) Fail(ctx context.Context, claim *DurableClaim, cause error) error {
	return d.finish(ctx, claim, taskStatusFailed, failureCode(cause), nil)
}

func (d *PostgresDriver) finish(ctx context.Context, claim *DurableClaim, status, failure string, availableAt *time.Time) error {
	if claim == nil {
		return ErrLeaseLost
	}
	now := time.Now().UTC()
	updates := map[string]any{"status": status, "lease_token": "", "lease_expires_at": nil, "last_failure_code": failure, "updated_at": now}
	if status == taskStatusCompleted {
		updates["completed_at"] = now
	}
	if status == taskStatusFailed {
		updates["failed_at"] = now
	}
	if availableAt != nil {
		updates["available_at"] = *availableAt
	}
	result := d.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ? AND status = ? AND lease_token = ? AND fencing_token = ? AND cancel_requested_at IS NULL", claim.ID, taskStatusProcessing, claim.LeaseToken, claim.FencingToken).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return d.finishCancellation(ctx, claim, now)
	}
	return nil
}

func (d *PostgresDriver) Retry(ctx context.Context, claim *DurableClaim, payload []byte, availableAt time.Time, cause error) error {
	if claim == nil {
		return ErrLeaseLost
	}
	result := d.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ? AND status = ? AND lease_token = ? AND fencing_token = ? AND cancel_requested_at IS NULL", claim.ID, taskStatusProcessing, claim.LeaseToken, claim.FencingToken).
		Updates(map[string]any{"status": taskStatusPending, "payload": payload, "available_at": availableAt, "lease_token": "", "lease_expires_at": nil, "last_failure_code": failureCode(cause), "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return d.finishCancellation(ctx, claim, time.Now().UTC())
	}
	return nil
}

func (d *PostgresDriver) Heartbeat(ctx context.Context, claim *DurableClaim, leaseDuration time.Duration) (bool, error) {
	if claim == nil {
		return false, ErrLeaseLost
	}
	var row TaskPO
	err := d.db.WithContext(ctx).Model(&TaskPO{}).
		Select("cancel_requested_at").
		Where("id = ? AND status = ? AND lease_token = ? AND fencing_token = ?", claim.ID, taskStatusProcessing, claim.LeaseToken, claim.FencingToken).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, ErrLeaseLost
	}
	if err != nil {
		return false, err
	}
	if row.CancelRequestedAt != nil {
		return true, nil
	}
	result := d.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ? AND status = ? AND lease_token = ? AND fencing_token = ?", claim.ID, taskStatusProcessing, claim.LeaseToken, claim.FencingToken).
		Updates(map[string]any{"lease_expires_at": time.Now().UTC().Add(leaseDuration), "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 {
		return false, ErrLeaseLost
	}
	return false, nil
}

func (d *PostgresDriver) Cancel(ctx context.Context, taskID string) error {
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ? AND status IN ?", taskID, []string{taskStatusPending, taskStatusProcessing}).
		Updates(map[string]any{
			"cancel_requested_at": now,
			"status":              gorm.Expr("CASE WHEN status = 'pending' THEN 'canceled' ELSE status END"),
			"updated_at":          now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (d *PostgresDriver) finishCancellation(ctx context.Context, claim *DurableClaim, now time.Time) error {
	result := d.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ? AND status = ? AND lease_token = ? AND fencing_token = ? AND cancel_requested_at IS NOT NULL", claim.ID, taskStatusProcessing, claim.LeaseToken, claim.FencingToken).
		Updates(map[string]any{"status": taskStatusCanceled, "lease_token": "", "lease_expires_at": nil, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrLeaseLost
	}
	return nil
}

func (d *PostgresDriver) Size(ctx context.Context, queue string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&TaskPO{}).Where("queue = ? AND status IN ?", queue, []string{taskStatusPending, taskStatusProcessing}).Count(&count).Error
	return count, err
}

func (d *PostgresDriver) Stats(ctx context.Context, queue string, now time.Time) (QueueStats, error) {
	var rows []struct {
		Status string
		Count  int64
	}
	if err := d.db.WithContext(ctx).Model(&TaskPO{}).Select("status, count(*) AS count").Where("queue = ?", queue).Group("status").Scan(&rows).Error; err != nil {
		return QueueStats{}, err
	}
	stats := QueueStats{}
	for _, row := range rows {
		switch row.Status {
		case taskStatusPending:
			stats.Pending = row.Count
		case taskStatusProcessing:
			stats.Processing = row.Count
		case taskStatusFailed:
			stats.Failed = row.Count
		}
	}
	var oldest *time.Time
	if err := d.db.WithContext(ctx).Model(&TaskPO{}).Select("min(available_at)").Where("queue = ? AND status = ? AND available_at <= ?", queue, taskStatusPending, now).Scan(&oldest).Error; err != nil {
		return QueueStats{}, err
	}
	if oldest != nil && now.After(*oldest) {
		stats.Lag = now.Sub(*oldest)
	}
	return stats, nil
}

func (d *PostgresDriver) Clear(ctx context.Context, queue string) error {
	return d.db.WithContext(ctx).Where("queue = ?", queue).Delete(&TaskPO{}).Error
}

func (d *PostgresDriver) Close() error { d.mu.Lock(); d.closed = true; d.mu.Unlock(); return nil }
func (d *PostgresDriver) ensureOpen() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		return ErrDriverClosed
	}
	return nil
}

func failureCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "job_handler_error"
	}
}

func taskFingerprint(payload JobPayload) ([32]byte, error) {
	stable, err := json.Marshal(struct {
		Type       string          `json:"type"`
		Data       json.RawMessage `json:"data"`
		MaxRetries int             `json:"max_retries"`
	}{Type: payload.Type, Data: payload.Data, MaxRetries: payload.MaxRetries})
	if err != nil {
		return [32]byte{}, fmt.Errorf("fingerprint workflow payload: %w", err)
	}
	return sha256.Sum256(stable), nil
}

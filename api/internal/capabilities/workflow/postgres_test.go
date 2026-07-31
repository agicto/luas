package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresDriverUsesFencedMultiReplicaClaims(t *testing.T) {
	db := openWorkflowPostgres(t)
	driverA, err := NewPostgresDriver(db)
	require.NoError(t, err)
	driverB, err := NewPostgresDriver(db)
	require.NoError(t, err)

	payload := workflowTestPayload(t, uuid.NewString(), "")
	require.NoError(t, driverA.Push(context.Background(), "default", payload))

	claims := make(chan *DurableClaim, 2)
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for _, driver := range []*PostgresDriver{driverA, driverB} {
		wait.Add(1)
		go func(candidate *PostgresDriver) {
			defer wait.Done()
			claim, claimErr := candidate.Claim(context.Background(), "default", time.Minute)
			claims <- claim
			errs <- claimErr
		}(driver)
	}
	wait.Wait()
	close(claims)
	close(errs)

	claimed := make([]*DurableClaim, 0, 1)
	for claim := range claims {
		if claim != nil {
			claimed = append(claimed, claim)
		}
	}
	require.Len(t, claimed, 1)
	var emptyCount int
	for claimErr := range errs {
		if errors.Is(claimErr, ErrQueueEmpty) {
			emptyCount++
		} else {
			require.NoError(t, claimErr)
		}
	}
	assert.Equal(t, 1, emptyCount)
	require.NoError(t, driverA.Complete(context.Background(), claimed[0]))
}

func TestPostgresDriverRejectsStaleFencingToken(t *testing.T) {
	db := openWorkflowPostgres(t)
	driver, err := NewPostgresDriver(db)
	require.NoError(t, err)
	require.NoError(t, driver.Push(context.Background(), "default", workflowTestPayload(t, uuid.NewString(), "")))

	first, err := driver.Claim(context.Background(), "default", time.Minute)
	require.NoError(t, err)
	require.NoError(t, db.Model(&TaskPO{}).Where("id = ?", first.ID).Update("lease_expires_at", time.Now().Add(-time.Second)).Error)
	second, err := driver.Claim(context.Background(), "default", time.Minute)
	require.NoError(t, err)
	assert.Greater(t, second.FencingToken, first.FencingToken)
	assert.ErrorIs(t, driver.Complete(context.Background(), first), ErrLeaseLost)
	require.NoError(t, driver.Complete(context.Background(), second))
}

func TestPostgresDriverIdempotencyAndCancellation(t *testing.T) {
	db := openWorkflowPostgres(t)
	driver, err := NewPostgresDriver(db)
	require.NoError(t, err)
	taskID := uuid.NewString()
	payload := workflowTestPayload(t, taskID, "invoice:42")
	createdID, err := driver.PushTask(context.Background(), "billing", payload)
	require.NoError(t, err)
	duplicateID, err := driver.PushTask(context.Background(), "billing", workflowTestPayload(t, uuid.NewString(), "invoice:42"))
	require.NoError(t, err)
	assert.Equal(t, createdID, duplicateID)

	conflict := workflowTestPayload(t, uuid.NewString(), "invoice:42")
	var envelope JobPayload
	require.NoError(t, json.Unmarshal(conflict, &envelope))
	envelope.Type = "differentJob"
	conflict, err = json.Marshal(envelope)
	require.NoError(t, err)
	assert.ErrorIs(t, driver.Push(context.Background(), "billing", conflict), ErrIdempotencyConflict)

	require.NoError(t, driver.Cancel(context.Background(), taskID))
	_, err = driver.Claim(context.Background(), "billing", time.Minute)
	assert.ErrorIs(t, err, ErrQueueEmpty)
}

func TestPostgresDriverDeadLettersAnExpiredFinalAttempt(t *testing.T) {
	db := openWorkflowPostgres(t)
	driver, err := NewPostgresDriver(db)
	require.NoError(t, err)
	payload := workflowTestPayload(t, uuid.NewString(), "")
	var envelope JobPayload
	require.NoError(t, json.Unmarshal(payload, &envelope))
	envelope.MaxRetries = 1
	payload, err = json.Marshal(envelope)
	require.NoError(t, err)
	require.NoError(t, driver.Push(context.Background(), "exports", payload))

	claim, err := driver.Claim(context.Background(), "exports", time.Minute)
	require.NoError(t, err)
	require.NoError(t, db.Model(&TaskPO{}).Where("id = ?", claim.ID).Update("lease_expires_at", time.Now().Add(-time.Second)).Error)
	_, err = driver.Claim(context.Background(), "exports", time.Minute)
	assert.ErrorIs(t, err, ErrQueueEmpty)

	var task TaskPO
	require.NoError(t, db.First(&task, "id = ?", claim.ID).Error)
	assert.Equal(t, taskStatusFailed, task.Status)
	assert.Equal(t, "lease_expired", task.LastFailureCode)
}

func openWorkflowPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("LUAS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LUAS_TEST_POSTGRES_DSN is required for PostgreSQL workflow evidence")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	schema := fmt.Sprintf("workflow_%d", time.Now().UnixNano())
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() { admin.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE") })

	schemaDSN := dsn + "&search_path=" + schema
	if !strings.Contains(dsn, "?") {
		schemaDSN = dsn + "?search_path=" + schema
	}
	db, err := gorm.Open(postgres.Open(schemaDSN), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&TaskPO{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_workflow_tasks_queue_idempotency ON workflow_tasks (queue, idempotency_key) WHERE idempotency_key IS NOT NULL`).Error)
	return db
}

func workflowTestPayload(t *testing.T, id, key string) []byte {
	t.Helper()
	payload, err := json.Marshal(JobPayload{ID: id, Type: "testJob", Data: json.RawMessage(`{"value":1}`), MaxRetries: 3, IdempotencyKey: key, CreatedAt: time.Now().UTC()})
	require.NoError(t, err)
	return payload
}

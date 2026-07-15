package webhook

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestRepositoryDurableDeliveryLifecycle(t *testing.T) {
	repository, _, organizationID, actorID := newWebhookRepositoryTest(t)
	ctx := context.Background()
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	endpoint := createWebhookRepositoryEndpoint(t, repository, organizationID, actorID, now)

	publication := testPublicationMutation(organizationID, endpoint.ID, "evt-1", "msg_1", now)
	created, err := repository.Publish(ctx, publication)
	require.NoError(t, err)
	require.True(t, created.Receipt.Created)
	require.Len(t, created.Deliveries, 1)
	assert.Equal(t, "msg_1", created.Deliveries[0].MessageID)

	repeated, err := repository.Publish(ctx, publication)
	require.NoError(t, err)
	assert.False(t, repeated.Receipt.Created)
	assert.Equal(t, created.Receipt.ID, repeated.Receipt.ID)
	assert.Equal(t, created.Receipt.MessageID, repeated.Receipt.MessageID)

	conflict := publication
	conflict.Fingerprint = fmt.Sprintf("%064d", 2)
	_, err = repository.Publish(ctx, conflict)
	assert.ErrorIs(t, err, domain.ErrWebhookIdempotencyConflict)

	claims, err := repository.ClaimDue(ctx, now, time.Minute, 10)
	require.NoError(t, err)
	require.Len(t, claims, 1)
	assert.Equal(t, uint16(1), claims[0].AttemptNumber)
	assert.Equal(t, uint8(1), claims[0].CycleAttempt)
	retryAt := now.Add(5 * time.Minute)
	completed, err := repository.Complete(ctx, claims[0], deliveryCompletion{
		Status:      deliveryStatusPending,
		Outcome:     "retry_scheduled",
		FailureCode: "WEBHOOK.HTTP_503",
		RetryAt:     &retryAt,
		StartedAt:   now,
		CompletedAt: now.Add(time.Second),
		DurationMS:  1000,
	}, webhookDisableAfterFailure)
	require.NoError(t, err)
	assert.True(t, completed)

	claims, err = repository.ClaimDue(ctx, retryAt, time.Minute, 10)
	require.NoError(t, err)
	require.Len(t, claims, 1)
	assert.Equal(t, uint16(2), claims[0].AttemptNumber)
	status := 204
	completed, err = repository.Complete(ctx, claims[0], deliveryCompletion{
		Status:      deliveryStatusDelivered,
		Outcome:     "delivered",
		HTTPStatus:  &status,
		StartedAt:   retryAt,
		CompletedAt: retryAt.Add(50 * time.Millisecond),
		DurationMS:  50,
	}, webhookDisableAfterFailure)
	require.NoError(t, err)
	assert.True(t, completed)

	deliveries, total, err := repository.ListDeliveries(ctx, organizationID, deliveryFilter{}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, deliveries, 1)
	assert.Equal(t, domain.WebhookDeliveryStatusDelivered, deliveries[0].Status)
	assert.Equal(t, uint16(2), deliveries[0].AttemptCount)
	assert.Equal(t, status, *deliveries[0].HTTPStatus)

	attempts, attemptTotal, err := repository.ListAttempts(ctx, organizationID, deliveries[0].ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), attemptTotal)
	require.Len(t, attempts, 2)
	assert.Equal(t, uint16(2), attempts[0].Number)
	assert.Equal(t, uint16(1), attempts[1].Number)

	replayed, err := repository.Replay(ctx, organizationID, deliveries[0].ID, actorID, retryAt.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, domain.WebhookDeliveryStatusPending, replayed.Status)
	assert.Equal(t, uint16(1), replayed.ReplayCount)
	assert.Equal(t, "msg_1", replayed.MessageID)

	claims, err = repository.ClaimDue(ctx, retryAt.Add(time.Hour), time.Minute, 10)
	require.NoError(t, err)
	require.Len(t, claims, 1)
	assert.Equal(t, uint16(3), claims[0].AttemptNumber)
	assert.Equal(t, uint8(1), claims[0].CycleAttempt)
}

func TestRepositoryAutoDisablesEndpointAfterConsecutiveTerminalFailures(t *testing.T) {
	repository, _, organizationID, actorID := newWebhookRepositoryTest(t)
	ctx := context.Background()
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	endpoint := createWebhookRepositoryEndpoint(t, repository, organizationID, actorID, now)

	for index := 1; index <= int(webhookDisableAfterFailure); index++ {
		at := now.Add(time.Duration(index) * time.Minute)
		publication := testPublicationMutation(
			organizationID,
			endpoint.ID,
			fmt.Sprintf("evt-%d", index),
			fmt.Sprintf("msg_%d", index),
			at,
		)
		_, err := repository.Publish(ctx, publication)
		require.NoError(t, err)
		claims, err := repository.ClaimDue(ctx, at, time.Minute, 1)
		require.NoError(t, err)
		require.Len(t, claims, 1)
		completed, err := repository.Complete(ctx, claims[0], deliveryCompletion{
			Status:      deliveryStatusFailed,
			Outcome:     "failed",
			FailureCode: "WEBHOOK.HTTP_400",
			StartedAt:   at,
			CompletedAt: at.Add(time.Second),
			DurationMS:  1000,
		}, webhookDisableAfterFailure)
		require.NoError(t, err)
		assert.True(t, completed)
	}

	endpoints, total, err := repository.ListEndpoints(ctx, organizationID, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, endpoints, 1)
	assert.Equal(t, domain.WebhookEndpointStatusDisabled, endpoints[0].Status)
	assert.Equal(t, "consecutive_failures", endpoints[0].DisabledReason)
	assert.Equal(t, webhookDisableAfterFailure, endpoints[0].ConsecutiveFailures)
	assert.Equal(t, uint64(2), endpoints[0].Version)
}

func TestRepositoryVersionedMutationCancelsOpenWorkAndScrubsSecrets(t *testing.T) {
	repository, db, organizationID, actorID := newWebhookRepositoryTest(t)
	ctx := context.Background()
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	endpoint := createWebhookRepositoryEndpoint(t, repository, organizationID, actorID, now)
	_, err := repository.Publish(ctx, testPublicationMutation(organizationID, endpoint.ID, "evt-open", "msg_open", now))
	require.NoError(t, err)

	updated, err := repository.UpdateEndpoint(ctx, endpoint.ID, endpointMutation{
		OrganizationID:  organizationID,
		ActorID:         actorID,
		Name:            "Updated endpoint",
		URL:             "https://8.8.4.4/next",
		URLHash:         fmt.Sprintf("%064d", 8),
		EventTypes:      []string{"webhook.test"},
		ExpectedVersion: 1,
		Now:             now.Add(time.Minute),
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), updated.Version)
	_, err = repository.UpdateEndpoint(ctx, endpoint.ID, endpointMutation{
		OrganizationID:  organizationID,
		ActorID:         actorID,
		Name:            "Stale",
		URL:             "https://8.8.8.8/stale",
		URLHash:         fmt.Sprintf("%064d", 9),
		EventTypes:      []string{"webhook.test"},
		ExpectedVersion: 1,
		Now:             now.Add(2 * time.Minute),
	})
	assert.ErrorIs(t, err, domain.ErrWebhookEndpointVersionConflict)

	deliveries, _, err := repository.ListDeliveries(ctx, organizationID, deliveryFilter{}, 1, 10)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, domain.WebhookDeliveryStatusCanceled, deliveries[0].Status)

	err = repository.DeleteEndpoint(ctx, organizationID, endpoint.ID, actorID, 2, now.Add(3*time.Minute))
	require.NoError(t, err)
	var persisted EndpointPO
	require.NoError(t, db.Unscoped().First(&persisted, endpoint.ID).Error)
	assert.Empty(t, persisted.SecretCiphertext)
	assert.Empty(t, persisted.PreviousSecretCiphertext)
	assert.True(t, persisted.DeletedAt.Valid)
	assert.Equal(t, "https://deleted.invalid/", persisted.URL)

	_, _, err = repository.ListEndpoints(ctx, organizationID, 1, 10)
	require.NoError(t, err)
}

func TestRepositoryPublicationHonorsOuterTransaction(t *testing.T) {
	repository, db, organizationID, actorID := newWebhookRepositoryTest(t)
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	endpoint := createWebhookRepositoryEndpoint(t, repository, organizationID, actorID, now)
	rollback := errors.New("roll back business mutation")

	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := infradatabase.ContextWithTransaction(context.Background(), tx)
		_, publishErr := repository.Publish(
			ctx,
			testPublicationMutation(organizationID, endpoint.ID, "evt-rollback", "msg_rollback", now),
		)
		require.NoError(t, publishErr)
		var inside int64
		require.NoError(t, tx.Model(&EventPO{}).Count(&inside).Error)
		assert.Equal(t, int64(1), inside)
		return rollback
	})
	assert.ErrorIs(t, err, rollback)
	var events int64
	require.NoError(t, db.Model(&EventPO{}).Count(&events).Error)
	assert.Zero(t, events)
	var deliveries int64
	require.NoError(t, db.Model(&DeliveryPO{}).Count(&deliveries).Error)
	assert.Zero(t, deliveries)
}

func TestWebhookLockQueriesSkipOnlyContendedClaimRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.Dialector = webhookNamedDialector{Dialector: db.Dialector, name: "postgres"}

	rowLock, ok := webhookRowLockQuery(db.Model(&DeliveryPO{})).Statement.Clauses["FOR"].Expression.(clause.Locking)
	require.True(t, ok)
	assert.Equal(t, "UPDATE", rowLock.Strength)
	assert.Empty(t, rowLock.Options)

	claimLock, ok := webhookClaimLockQuery(db.Model(&DeliveryPO{})).Statement.Clauses["FOR"].Expression.(clause.Locking)
	require.True(t, ok)
	assert.Equal(t, "UPDATE", claimLock.Strength)
	assert.Equal(t, "SKIP LOCKED", claimLock.Options)
}

func TestWebhookIndexesMatchWorkerAndManagementQueries(t *testing.T) {
	_, db, _, _ := newWebhookRepositoryTest(t)

	requireWebhookIndexColumns(t, db, "idx_webhook_endpoints_organization_status", []string{
		"organization_id", "status", "id",
	})
	requireWebhookIndexColumns(t, db, "idx_webhook_endpoints_organization_created", []string{
		"organization_id", "created_at", "id",
	})
	requireWebhookIndexColumns(t, db, "idx_webhook_events_created", []string{"created_at", "id"})
	requireWebhookIndexColumns(t, db, "idx_webhook_deliveries_due", []string{
		"status", "available_at", "id",
	})
	requireWebhookIndexColumns(t, db, "idx_webhook_deliveries_lease_expiry", []string{
		"status", "lease_expires_at",
	})
	requireWebhookIndexColumns(t, db, "idx_webhook_deliveries_organization_endpoint_created", []string{
		"organization_id", "endpoint_id", "created_at", "id",
	})
	requireWebhookIndexColumns(t, db, "idx_webhook_deliveries_organization_status_created", []string{
		"organization_id", "status", "created_at", "id",
	})
}

func newWebhookRepositoryTest(t *testing.T) (*repository, *gorm.DB, uint, uint) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=1", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&user.UserPO{},
		&organization.OrganizationPO{},
		&EndpointPO{},
		&SubscriptionPO{},
		&EventPO{},
		&DeliveryPO{},
		&AttemptPO{},
	))
	actor := user.UserPO{Username: "webhook-owner", Email: "owner@example.com", Password: "unused", Status: 1}
	require.NoError(t, db.Create(&actor).Error)
	owner := organization.OrganizationPO{Name: "Webhook Test", Slug: "webhook-test", CreatedBy: actor.ID}
	require.NoError(t, db.Create(&owner).Error)
	var foreignKeys []webhookForeignKey
	require.NoError(t, db.Raw("PRAGMA foreign_key_list('webhook_events')").Scan(&foreignKeys).Error)
	assert.Equal(t, []webhookForeignKey{{Table: "organizations", From: "organization_id", To: "id"}}, foreignKeys)
	foreignKeys = nil
	require.NoError(t, db.Raw("PRAGMA foreign_key_list('webhook_deliveries')").Scan(&foreignKeys).Error)
	assert.ElementsMatch(t, []webhookForeignKey{
		{Table: "webhook_events", From: "event_id", To: "id"},
		{Table: "webhook_endpoints", From: "endpoint_id", To: "id"},
	}, foreignKeys)
	return NewRepository(db), db, owner.ID, actor.ID
}

type webhookForeignKey struct {
	Table string `gorm:"column:table"`
	From  string `gorm:"column:from"`
	To    string `gorm:"column:to"`
}

type webhookNamedDialector struct {
	gorm.Dialector
	name string
}

func (d webhookNamedDialector) Name() string { return d.name }

func requireWebhookIndexColumns(t *testing.T, db *gorm.DB, index string, expected []string) {
	t.Helper()
	var rows []struct {
		Sequence int    `gorm:"column:seqno"`
		Name     string `gorm:"column:name"`
	}
	require.NoError(t, db.Raw("PRAGMA index_info('"+index+"')").Scan(&rows).Error)
	require.Len(t, rows, len(expected), "missing or incomplete index %s", index)
	actual := make([]string, len(rows))
	for _, row := range rows {
		require.GreaterOrEqual(t, row.Sequence, 0)
		require.Less(t, row.Sequence, len(rows))
		actual[row.Sequence] = row.Name
	}
	assert.Equal(t, expected, actual, "unexpected columns for index %s", index)
}

func createWebhookRepositoryEndpoint(
	t *testing.T,
	repository *repository,
	organizationID uint,
	actorID uint,
	now time.Time,
) *domain.WebhookEndpoint {
	t.Helper()
	endpoint, err := repository.CreateEndpoint(context.Background(), endpointMutation{
		OrganizationID:   organizationID,
		ActorID:          actorID,
		Name:             "Order processor",
		URL:              "https://8.8.8.8/hook",
		URLHash:          fmt.Sprintf("%064d", 1),
		EventTypes:       []string{"webhook.test"},
		SecretCiphertext: "encrypted-secret",
		SecretHint:       "abcd1234",
		Now:              now,
	})
	require.NoError(t, err)
	return endpoint
}

func testPublicationMutation(
	organizationID uint,
	endpointID uint,
	eventID string,
	messageID string,
	now time.Time,
) publicationMutation {
	return publicationMutation{
		OrganizationID: organizationID,
		TargetEndpoint: endpointID,
		MessageID:      messageID,
		Source:         "webhook.repository_test",
		EventID:        eventID,
		Fingerprint:    fmt.Sprintf("%064d", len(eventID)),
		EventType:      "webhook.test",
		PayloadJSON:    `{"endpoint_id":1,"organization_id":1}`,
		OccurredAt:     now,
		Now:            now,
	}
}

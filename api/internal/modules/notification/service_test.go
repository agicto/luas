package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/email"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type fakeEmailSender struct {
	mu         sync.Mutex
	configured bool
	errors     []error
	keys       []string
	recipients []string
	bodies     []string
}

func (f *fakeEmailSender) IsConfigured() bool { return f != nil && f.configured }

func (f *fakeEmailSender) SendEmailIdempotent(
	_ context.Context,
	to []string,
	_ string,
	body string,
	key string,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, key)
	if len(to) > 0 {
		f.recipients = append(f.recipients, to[0])
	}
	f.bodies = append(f.bodies, body)
	if len(f.errors) == 0 {
		return nil
	}
	err := f.errors[0]
	f.errors = f.errors[1:]
	return err
}

func TestPublishIsIdempotentAndRejectsConflictingReplay(t *testing.T) {
	service, db, _, now := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	publication := testPublication(userID, "invoice:1042")

	first, err := service.Publish(context.Background(), publication)
	require.NoError(t, err)
	second, err := service.Publish(context.Background(), publication)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	var notificationCount int64
	require.NoError(t, db.Model(&NotificationPO{}).Count(&notificationCount).Error)
	assert.Equal(t, int64(1), notificationCount)
	var deliveries []NotificationDeliveryPO
	require.NoError(t, db.Order("channel ASC").Find(&deliveries).Error)
	require.Len(t, deliveries, 2)
	assert.Equal(t, string(deliveryStatusDelivered), deliveries[1].Status)
	assert.WithinDuration(t, now, *deliveries[1].DeliveredAt, time.Millisecond)

	conflicting := publication
	conflicting.Title = "Different title"
	_, err = service.Publish(context.Background(), conflicting)
	assert.ErrorIs(t, err, domain.ErrNotificationIdempotencyConflict)
}

func TestPublishAppliesPreferencesAndRequiredChannelPriority(t *testing.T) {
	service, db, _, _ := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	_, err := service.ReplacePreference(context.Background(), userID, false, false)
	require.NoError(t, err)

	optional := testPublication(userID, "optional")
	notification, err := service.Publish(context.Background(), optional)
	require.NoError(t, err)
	var optionalDeliveryCount int64
	require.NoError(t, db.Model(&NotificationDeliveryPO{}).
		Where("notification_id = ?", notification.ID).
		Count(&optionalDeliveryCount).Error)
	assert.Zero(t, optionalDeliveryCount)

	required := testPublication(userID, "required")
	required.RequiredChannels = []domain.NotificationChannel{
		domain.NotificationChannelInApp,
		domain.NotificationChannelEmail,
	}
	notification, err = service.Publish(context.Background(), required)
	require.NoError(t, err)
	var channels []string
	require.NoError(t, db.Model(&NotificationDeliveryPO{}).
		Where("notification_id = ?", notification.ID).
		Order("channel ASC").
		Pluck("channel", &channels).Error)
	assert.Equal(t, []string{"email", "in_app"}, channels)
}

func TestNotificationCenterEnforcesOwnershipAndReadHighWaterMark(t *testing.T) {
	service, db, _, _ := newNotificationTestService(t)
	aliceID := createNotificationTestUser(t, db, "alice")
	bobID := createNotificationTestUser(t, db, "bob")

	first, err := service.Publish(context.Background(), testPublication(aliceID, "alice-1"))
	require.NoError(t, err)
	bob, err := service.Publish(context.Background(), testPublication(bobID, "bob-1"))
	require.NoError(t, err)
	second, err := service.Publish(context.Background(), testPublication(aliceID, "alice-2"))
	require.NoError(t, err)

	items, total, err := service.ListForUser(context.Background(), aliceID, notificationFilterAll, 1, 15)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	assert.Equal(t, second.ID, items[0].ID)

	_, err = service.ReplaceReadState(context.Background(), aliceID, bob.ID, true)
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)

	updated, unread, err := service.MarkReadThrough(context.Background(), aliceID, first.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated)
	assert.Equal(t, int64(1), unread)

	firstAfter, err := service.ReplaceReadState(context.Background(), aliceID, first.ID, true)
	require.NoError(t, err)
	assert.True(t, firstAfter.IsRead())
	count, err := service.UnreadCount(context.Background(), aliceID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestDispatchRetriesWithStableProviderIdempotencyKey(t *testing.T) {
	service, db, mailer, initialNow := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	mailer.errors = []error{&email.ProviderError{StatusCode: 500}, nil}
	publication := testPublication(userID, "retry")
	publication.Channels = []domain.NotificationChannel{domain.NotificationChannelEmail}
	_, err := service.Publish(context.Background(), publication)
	require.NoError(t, err)

	processed, err := service.DispatchDue(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	var delivery NotificationDeliveryPO
	require.NoError(t, db.First(&delivery).Error)
	assert.Equal(t, string(deliveryStatusPending), delivery.Status)
	assert.Equal(t, failureProviderUnavailable, delivery.LastFailureCode)
	assert.NotEmpty(t, delivery.DestinationHash)

	currentNow := initialNow.Add(time.Minute)
	service.now = func() time.Time { return currentNow }
	processed, err = service.DispatchDue(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	require.NoError(t, db.First(&delivery, delivery.ID).Error)
	assert.Equal(t, string(deliveryStatusDelivered), delivery.Status)
	assert.Equal(t, uint8(2), delivery.Attempts)
	require.Len(t, mailer.keys, 2)
	assert.Equal(t, mailer.keys[0], mailer.keys[1])
	assert.Equal(t, fmt.Sprintf("notification-email-%d", delivery.ID), mailer.keys[0])
	assert.NotContains(t, mailer.bodies[0], "<script>")
}

func TestDispatchFailsClosedWhenRecipientRouteChangesBetweenAttempts(t *testing.T) {
	service, db, mailer, initialNow := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	mailer.errors = []error{&email.ProviderError{StatusCode: 500}}
	publication := testPublication(userID, "route-change")
	publication.Channels = []domain.NotificationChannel{domain.NotificationChannelEmail}
	_, err := service.Publish(context.Background(), publication)
	require.NoError(t, err)
	_, err = service.DispatchDue(context.Background(), 10)
	require.NoError(t, err)
	require.NoError(t, db.Model(&user.UserPO{}).Where("id = ?", userID).
		Update("email", "new-alice@example.com").Error)

	service.now = func() time.Time { return initialNow.Add(time.Minute) }
	processed, err := service.DispatchDue(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	var delivery NotificationDeliveryPO
	require.NoError(t, db.First(&delivery).Error)
	assert.Equal(t, string(deliveryStatusFailed), delivery.Status)
	assert.Equal(t, failureRouteChanged, delivery.LastFailureCode)
	assert.Len(t, mailer.keys, 1)
}

func TestExpiredLeaseCannotBeCompletedByPreviousWorker(t *testing.T) {
	service, db, _, now := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	publication := testPublication(userID, "lease")
	publication.Channels = []domain.NotificationChannel{domain.NotificationChannelEmail}
	_, err := service.Publish(context.Background(), publication)
	require.NoError(t, err)

	first, err := service.store.ClaimDueEmail(context.Background(), now, time.Minute, maxDeliveryAttempts, 1)
	require.NoError(t, err)
	require.Len(t, first, 1)
	secondNow := now.Add(2 * time.Minute)
	second, err := service.store.ClaimDueEmail(context.Background(), secondNow, time.Minute, maxDeliveryAttempts, 1)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.NotEqual(t, first[0].LeaseToken, second[0].LeaseToken)

	completed, err := service.store.CompleteEmail(context.Background(), first[0].ID, first[0].LeaseToken, deliveryCompletion{
		Status:      deliveryStatusDelivered,
		CompletedAt: secondNow,
	})
	require.NoError(t, err)
	assert.False(t, completed)
	completed, err = service.store.CompleteEmail(context.Background(), second[0].ID, second[0].LeaseToken, deliveryCompletion{
		Status:      deliveryStatusDelivered,
		CompletedAt: secondNow,
	})
	require.NoError(t, err)
	assert.True(t, completed)
}

func TestBindEmailDestinationIsIdempotentOnlyForTheCurrentLeaseAndRoute(t *testing.T) {
	service, db, _, now := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	publication := testPublication(userID, "destination-bind")
	publication.Channels = []domain.NotificationChannel{domain.NotificationChannelEmail}
	_, err := service.Publish(context.Background(), publication)
	require.NoError(t, err)

	claimed, err := service.store.ClaimDueEmail(
		context.Background(),
		now,
		time.Minute,
		maxDeliveryAttempts,
		1,
	)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	bound, err := service.store.BindEmailDestination(
		context.Background(),
		claimed[0].ID,
		claimed[0].LeaseToken,
		"route-hash",
	)
	require.NoError(t, err)
	assert.True(t, bound)

	bound, err = service.store.BindEmailDestination(
		context.Background(),
		claimed[0].ID,
		claimed[0].LeaseToken,
		"route-hash",
	)
	require.NoError(t, err)
	assert.True(t, bound)

	bound, err = service.store.BindEmailDestination(
		context.Background(),
		claimed[0].ID,
		claimed[0].LeaseToken,
		"different-route-hash",
	)
	require.NoError(t, err)
	assert.False(t, bound)
}

func TestPublicationValidationRejectsUnsafeBoundaryValues(t *testing.T) {
	service, db, _, _ := newNotificationTestService(t)
	userID := createNotificationTestUser(t, db, "alice")
	tests := []struct {
		name   string
		mutate func(*domain.NotificationPublication)
		target error
	}{
		{name: "unknown channel", mutate: func(value *domain.NotificationPublication) {
			value.Channels = []domain.NotificationChannel{"sms"}
		}, target: domain.ErrNotificationInvalidChannel},
		{name: "required channel outside declaration", mutate: func(value *domain.NotificationPublication) {
			value.Channels = []domain.NotificationChannel{domain.NotificationChannelInApp}
			value.RequiredChannels = []domain.NotificationChannel{domain.NotificationChannelEmail}
		}, target: domain.ErrNotificationInvalidChannel},
		{name: "absolute URL", mutate: func(value *domain.NotificationPublication) {
			value.ActionURL = "https://example.com/phish"
		}, target: domain.ErrInvalidInput},
		{name: "protocol relative URL", mutate: func(value *domain.NotificationPublication) {
			value.ActionURL = "//example.com/phish"
		}, target: domain.ErrInvalidInput},
		{name: "invalid kind", mutate: func(value *domain.NotificationPublication) {
			value.Kind = "InvoicePaid"
		}, target: domain.ErrInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := testPublication(userID, "unsafe-boundary")
			test.mutate(&publication)
			_, err := service.Publish(context.Background(), publication)
			assert.ErrorIs(t, err, test.target)
		})
	}
}

func newNotificationTestService(
	t *testing.T,
) (*service, *gorm.DB, *fakeEmailSender, time.Time) {
	t.Helper()
	db := testplatform.OpenPostgres(t, &gorm.Config{TranslateError: true},
		&user.UserPO{},
		&NotificationPO{},
		&NotificationDeliveryPO{},
		&NotificationPreferencePO{},
	)
	mailer := &fakeEmailSender{configured: true}
	cfg := &config.Config{Starters: config.StarterConfig{Optional: []string{"notification"}}}
	service := NewService(cfg, NewRepository(db), mailer, nil)
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	return service, db, mailer, now
}

func createNotificationTestUser(t *testing.T, db *gorm.DB, name string) uint {
	t.Helper()
	po := user.UserPO{
		Username: name,
		Email:    name + "@example.com",
		Password: "not-used",
		Status:   1,
	}
	require.NoError(t, db.Create(&po).Error)
	return po.ID
}

func testPublication(userID uint, key string) domain.NotificationPublication {
	return domain.NotificationPublication{
		UserID:         userID,
		IdempotencyKey: key,
		Kind:           "billing.invoice_paid",
		Title:          "Invoice paid",
		Body:           "<script>alert('x')</script>\nInvoice 1042 was paid.",
		ActionURL:      "/console/invoices/1042",
		Channels: []domain.NotificationChannel{
			domain.NotificationChannelInApp,
			domain.NotificationChannelEmail,
		},
	}
}

func TestDisabledServiceFailsClosed(t *testing.T) {
	service := NewService(&config.Config{}, nil, nil, nil)
	_, err := service.Publish(context.Background(), domain.NotificationPublication{})
	assert.True(t, errors.Is(err, domain.ErrServiceUnavailable))
}

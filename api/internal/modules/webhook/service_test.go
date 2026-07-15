package webhook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestServiceCreatesPublishesAndDispatchesWebhook(t *testing.T) {
	var mutex sync.Mutex
	messageIDs := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mutex.Lock()
		messageIDs = append(messageIDs, request.Header.Get("webhook-id"))
		mutex.Unlock()
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service, organizationID, actorID, setNow := newWebhookServiceTest(t)
	endpointSecret, err := service.CreateEndpoint(context.Background(), organizationID, actorID, endpointInput{
		Name:       " Local receiver ",
		URL:        server.URL + "/deliver",
		EventTypes: []string{"webhook.test"},
	})
	require.NoError(t, err)
	require.NotNil(t, endpointSecret.Endpoint)
	assert.Equal(t, "Local receiver", endpointSecret.Endpoint.Name)
	assert.Contains(t, endpointSecret.SigningSecret, webhookSecretPrefix)
	assert.NotContains(t, endpointSecret.Endpoint.SecretHint, endpointSecret.SigningSecret)

	first, err := service.PublishWebhookTest(
		context.Background(), organizationID, endpointSecret.Endpoint.ID, actorID, "setup-test-1",
	)
	require.NoError(t, err)
	setNow(time.Date(2026, time.July, 15, 12, 5, 0, 0, time.UTC))
	repeated, err := service.PublishWebhookTest(
		context.Background(), organizationID, endpointSecret.Endpoint.ID, actorID, "setup-test-1",
	)
	require.NoError(t, err)
	assert.Equal(t, first.ID, repeated.ID)
	assert.Equal(t, first.MessageID, repeated.MessageID)

	processed, err := service.DispatchWebhooks(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	mutex.Lock()
	require.Equal(t, []string{first.MessageID}, messageIDs)
	mutex.Unlock()

	deliveries, total, err := service.ListDeliveries(
		context.Background(), organizationID, deliveryFilter{}, 1, 10,
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, deliveries, 1)
	assert.Equal(t, domain.WebhookDeliveryStatusDelivered, deliveries[0].Status)
	assert.Equal(t, uint16(1), deliveries[0].AttemptCount)
}

func TestServiceRetriesWithStableMessageID(t *testing.T) {
	var mutex sync.Mutex
	var calls int
	var messageIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		calls++
		messageIDs = append(messageIDs, request.Header.Get("webhook-id"))
		if calls == 1 {
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service, organizationID, actorID, setNow := newWebhookServiceTest(t)
	endpoint, err := service.CreateEndpoint(context.Background(), organizationID, actorID, endpointInput{
		Name:       "Retry receiver",
		URL:        server.URL,
		EventTypes: []string{"webhook.test"},
	})
	require.NoError(t, err)
	delivery, err := service.PublishWebhookTest(context.Background(), organizationID, endpoint.Endpoint.ID, actorID, "retry-1")
	require.NoError(t, err)

	processed, err := service.DispatchWebhooks(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	deliveries, _, err := service.ListDeliveries(context.Background(), organizationID, deliveryFilter{}, 1, 10)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, domain.WebhookDeliveryStatusPending, deliveries[0].Status)
	assert.Equal(t, "WEBHOOK.HTTP_503", deliveries[0].FailureCode)
	setNow(deliveries[0].AvailableAt)

	processed, err = service.DispatchWebhooks(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	deliveries, _, err = service.ListDeliveries(context.Background(), organizationID, deliveryFilter{}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, domain.WebhookDeliveryStatusDelivered, deliveries[0].Status)
	assert.Equal(t, uint16(2), deliveries[0].AttemptCount)
	mutex.Lock()
	assert.Equal(t, []string{delivery.MessageID, delivery.MessageID}, messageIDs)
	mutex.Unlock()
}

func TestServiceRotatesAndVersionProtectsEndpoint(t *testing.T) {
	service, organizationID, actorID, _ := newWebhookServiceTest(t)
	created, err := service.CreateEndpoint(context.Background(), organizationID, actorID, endpointInput{
		Name:       "Receiver",
		URL:        "http://127.0.0.1:8080/hook",
		EventTypes: []string{"webhook.test"},
	})
	require.NoError(t, err)

	rotated, err := service.RotateEndpointSecret(
		context.Background(), organizationID, created.Endpoint.ID, actorID, created.Endpoint.Version,
	)
	require.NoError(t, err)
	assert.NotEqual(t, created.SigningSecret, rotated.SigningSecret)
	assert.Equal(t, uint64(2), rotated.Endpoint.Version)
	assert.Equal(t, uint64(2), rotated.Endpoint.SecretVersion)
	assert.NotNil(t, rotated.PreviousSecretExpiry)

	_, err = service.ReplaceEndpointStatus(
		context.Background(), organizationID, created.Endpoint.ID, actorID, 1, false,
	)
	assert.ErrorIs(t, err, domain.ErrWebhookEndpointVersionConflict)
	disabled, err := service.ReplaceEndpointStatus(
		context.Background(), organizationID, created.Endpoint.ID, actorID, 2, false,
	)
	require.NoError(t, err)
	assert.Equal(t, domain.WebhookEndpointStatusDisabled, disabled.Status)
	_, err = service.PublishWebhookTest(
		context.Background(), organizationID, created.Endpoint.ID, actorID, "disabled-test",
	)
	assert.ErrorIs(t, err, domain.ErrWebhookReplayNotAllowed)
}

func TestServiceFailsClosedWhenStarterIsDisabled(t *testing.T) {
	service := NewService(nil, nil, nil, nil, nil, &config.Config{})
	_, err := service.EventTypes(context.Background())
	assert.True(t, errors.Is(err, domain.ErrServiceUnavailable))
	_, err = service.PublishWebhook(context.Background(), domain.WebhookEvent{})
	assert.True(t, errors.Is(err, domain.ErrServiceUnavailable))
}

func newWebhookServiceTest(t *testing.T) (*service, uint, uint, func(time.Time)) {
	t.Helper()
	repository, _, organizationID, actorID := newWebhookRepositoryTest(t)
	cfg := &config.Config{
		Starters: config.StarterConfig{Optional: []string{"organization", "webhook"}},
		Webhook: config.WebhookConfig{
			EncryptionKey:       "webhook-test-encryption-key-at-least-32-bytes",
			RequestTimeout:      time.Second,
			MaxResponseBytes:    1024,
			SecretOverlap:       24 * time.Hour,
			EventRetention:      30 * 24 * time.Hour,
			AllowInsecureHTTP:   true,
			AllowPrivateTargets: true,
		},
	}
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)
	secrets, err := NewSecretProtector(cfg)
	require.NoError(t, err)
	targets := NewTargetPolicy(cfg)
	sender := NewSender(cfg, targets)
	service := NewService(catalog, repository, secrets, targets, sender, cfg)
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	sender.now = func() time.Time { return now }
	setNow := func(value time.Time) { now = value }
	return service, organizationID, actorID, setNow
}

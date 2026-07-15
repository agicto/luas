package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

const (
	webhookMaximumEventAge     = 30 * 24 * time.Hour
	webhookMaximumFutureSkew   = 24 * time.Hour
	webhookDeliveryLease       = time.Minute
	webhookMaximumAttempts     = uint8(10)
	webhookDisableAfterFailure = uint8(5)
	webhookMaximumBatch        = 100
)

var webhookIdempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$`)

type endpointInput struct {
	Name       string
	URL        string
	EventTypes []string
}

// Service is the internal and HTTP-facing webhook application boundary.
type Service interface {
	domain.WebhookPublisher
	domain.WebhookDispatcher
	domain.WebhookTester
	domain.WebhookMaintainer
	EventTypes(context.Context) ([]string, error)
	ListEndpoints(context.Context, uint, int, int) ([]*domain.WebhookEndpoint, int64, error)
	CreateEndpoint(context.Context, uint, uint, endpointInput) (*domain.WebhookEndpointSecret, error)
	UpdateEndpoint(context.Context, uint, uint, uint, uint64, endpointInput) (*domain.WebhookEndpoint, error)
	ReplaceEndpointStatus(context.Context, uint, uint, uint, uint64, bool) (*domain.WebhookEndpoint, error)
	DeleteEndpoint(context.Context, uint, uint, uint, uint64) error
	RotateEndpointSecret(context.Context, uint, uint, uint, uint64) (*domain.WebhookEndpointSecret, error)
	ListDeliveries(context.Context, uint, deliveryFilter, int, int) ([]*domain.WebhookDelivery, int64, error)
	ListAttempts(context.Context, uint, uint64, int, int) ([]*domain.WebhookAttempt, int64, error)
}

type service struct {
	catalog       *Catalog
	store         webhookStore
	secrets       *secretProtector
	targets       *targetPolicy
	sender        webhookSender
	enabled       bool
	secretOverlap time.Duration
	retention     time.Duration
	now           func() time.Time
}

var (
	_ Service                  = (*service)(nil)
	_ domain.WebhookPublisher  = (*service)(nil)
	_ domain.WebhookDispatcher = (*service)(nil)
	_ domain.WebhookTester     = (*service)(nil)
	_ domain.WebhookMaintainer = (*service)(nil)
)

// NewService creates the optional outbound webhook application service.
func NewService(
	catalog *Catalog,
	store webhookStore,
	secrets *secretProtector,
	targets *targetPolicy,
	sender *sender,
	cfg *config.Config,
) *service {
	value := &service{
		catalog: catalog,
		store:   store,
		secrets: secrets,
		targets: targets,
		sender:  sender,
		now:     time.Now,
	}
	if cfg != nil {
		value.enabled = slices.Contains(cfg.Starters.Optional, "webhook")
		value.secretOverlap = cfg.Webhook.SecretOverlap
		value.retention = cfg.Webhook.EventRetention
	}
	return value
}

func (s *service) EventTypes(context.Context) ([]string, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	return s.catalog.Types(), nil
}

func (s *service) ListEndpoints(
	ctx context.Context,
	organizationID uint,
	page int,
	pageSize int,
) ([]*domain.WebhookEndpoint, int64, error) {
	if err := s.available(); err != nil {
		return nil, 0, err
	}
	return s.store.ListEndpoints(ctx, organizationID, page, pageSize)
}

func (s *service) CreateEndpoint(
	ctx context.Context,
	organizationID uint,
	actorID uint,
	input endpointInput,
) (*domain.WebhookEndpointSecret, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	normalized, err := s.normalizeEndpointInput(ctx, input)
	if err != nil {
		return nil, err
	}
	secret, err := s.secrets.Generate()
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	endpoint, err := s.store.CreateEndpoint(ctx, endpointMutation{
		OrganizationID:   organizationID,
		ActorID:          actorID,
		Name:             normalized.Name,
		URL:              normalized.URL,
		URLHash:          normalized.URLHash,
		EventTypes:       normalized.EventTypes,
		SecretCiphertext: secret.Ciphertext,
		SecretHint:       secret.Hint,
		Now:              now,
	})
	if err != nil {
		return nil, err
	}
	recordWebhookAudit(ctx, "create", endpoint.ID, map[string]any{
		"event_types": endpoint.EventTypes,
		"version":     endpoint.Version,
	})
	return &domain.WebhookEndpointSecret{Endpoint: endpoint, SigningSecret: secret.Plaintext}, nil
}

func (s *service) UpdateEndpoint(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	expectedVersion uint64,
	input endpointInput,
) (*domain.WebhookEndpoint, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	normalized, err := s.normalizeEndpointInput(ctx, input)
	if err != nil {
		return nil, err
	}
	endpoint, err := s.store.UpdateEndpoint(ctx, endpointID, endpointMutation{
		OrganizationID:  organizationID,
		ActorID:         actorID,
		Name:            normalized.Name,
		URL:             normalized.URL,
		URLHash:         normalized.URLHash,
		EventTypes:      normalized.EventTypes,
		ExpectedVersion: expectedVersion,
		Now:             s.now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	recordWebhookAudit(ctx, "update", endpoint.ID, map[string]any{
		"event_types":      endpoint.EventTypes,
		"previous_version": expectedVersion,
		"version":          endpoint.Version,
	})
	return endpoint, nil
}

func (s *service) ReplaceEndpointStatus(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	expectedVersion uint64,
	enabled bool,
) (*domain.WebhookEndpoint, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	endpoint, err := s.store.ReplaceEndpointStatus(
		ctx,
		organizationID,
		endpointID,
		actorID,
		enabled,
		expectedVersion,
		s.now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	recordWebhookAudit(ctx, "status", endpoint.ID, map[string]any{
		"status":           endpoint.Status,
		"previous_version": expectedVersion,
		"version":          endpoint.Version,
	})
	return endpoint, nil
}

func (s *service) DeleteEndpoint(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	expectedVersion uint64,
) error {
	if err := s.available(); err != nil {
		return err
	}
	if err := s.store.DeleteEndpoint(ctx, organizationID, endpointID, actorID, expectedVersion, s.now().UTC()); err != nil {
		return err
	}
	recordWebhookAudit(ctx, "delete", endpointID, map[string]any{"previous_version": expectedVersion})
	return nil
}

func (s *service) RotateEndpointSecret(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	expectedVersion uint64,
) (*domain.WebhookEndpointSecret, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	secret, err := s.secrets.Generate()
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	expires := now.Add(s.secretOverlap)
	endpoint, err := s.store.RotateEndpointSecret(ctx, endpointID, endpointMutation{
		OrganizationID:   organizationID,
		ActorID:          actorID,
		SecretCiphertext: secret.Ciphertext,
		SecretHint:       secret.Hint,
		ExpectedVersion:  expectedVersion,
		Now:              now,
	}, expires)
	if err != nil {
		return nil, err
	}
	recordWebhookAudit(ctx, "rotate_secret", endpoint.ID, map[string]any{
		"secret_version":   endpoint.SecretVersion,
		"previous_version": expectedVersion,
		"version":          endpoint.Version,
	})
	return &domain.WebhookEndpointSecret{
		Endpoint:             endpoint,
		SigningSecret:        secret.Plaintext,
		PreviousSecretExpiry: cloneTime(&expires),
	}, nil
}

func (s *service) PublishWebhook(
	ctx context.Context,
	event domain.WebhookEvent,
) (*domain.WebhookReceipt, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	mutation, err := s.normalizePublication(event, 0)
	if err != nil {
		return nil, err
	}
	result, err := s.store.Publish(ctx, mutation)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Receipt == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return result.Receipt, nil
}

func (s *service) PublishWebhookTest(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	idempotencyKey string,
) (*domain.WebhookDelivery, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if organizationID == 0 || endpointID == 0 || actorID == 0 || !webhookIdempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, domain.ErrInvalidInput
	}
	payload, err := json.Marshal(struct {
		OrganizationID uint `json:"organization_id"`
		EndpointID     uint `json:"endpoint_id"`
	}{OrganizationID: organizationID, EndpointID: endpointID})
	if err != nil {
		return nil, domain.ErrServiceUnavailable
	}
	eventIdentity := sha256.Sum256([]byte(strconv.FormatUint(uint64(endpointID), 10) + ":" + idempotencyKey))
	mutation, err := s.normalizePublication(domain.WebhookEvent{
		OrganizationID: organizationID,
		Source:         "webhook.http_test",
		EventID:        "test:" + hex.EncodeToString(eventIdentity[:]),
		Type:           "webhook.test",
		OccurredAt:     s.now().UTC(),
		Data:           payload,
	}, endpointID)
	if err != nil {
		return nil, err
	}
	// Occurrence time is server-generated for this fixed HTTP command. Exclude it from the
	// idempotency fingerprint so a transport retry returns the originally queued delivery.
	testFingerprint := sha256.Sum256([]byte(
		strconv.FormatUint(uint64(organizationID), 10) + ":" +
			strconv.FormatUint(uint64(endpointID), 10) + ":" + idempotencyKey + ":" + string(payload),
	))
	mutation.Fingerprint = hex.EncodeToString(testFingerprint[:])
	result, err := s.store.Publish(ctx, mutation)
	if err != nil {
		return nil, err
	}
	if result == nil || len(result.Deliveries) != 1 || result.Deliveries[0] == nil {
		return nil, domain.ErrServiceUnavailable
	}
	delivery := result.Deliveries[0]
	recordWebhookAudit(ctx, "test", endpointID, map[string]any{
		"delivery_id": delivery.ID,
		"message_id":  delivery.MessageID,
	})
	return delivery, nil
}

func (s *service) ListDeliveries(
	ctx context.Context,
	organizationID uint,
	filter deliveryFilter,
	page int,
	pageSize int,
) ([]*domain.WebhookDelivery, int64, error) {
	if err := s.available(); err != nil {
		return nil, 0, err
	}
	if filter.Status != "" && !validDeliveryStatus(filter.Status) {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.store.ListDeliveries(ctx, organizationID, filter, page, pageSize)
}

func (s *service) ListAttempts(
	ctx context.Context,
	organizationID uint,
	deliveryID uint64,
	page int,
	pageSize int,
) ([]*domain.WebhookAttempt, int64, error) {
	if err := s.available(); err != nil {
		return nil, 0, err
	}
	return s.store.ListAttempts(ctx, organizationID, deliveryID, page, pageSize)
}

func (s *service) DispatchWebhooks(ctx context.Context, limit int) (int, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	if limit < 1 || limit > webhookMaximumBatch {
		return 0, domain.ErrInvalidInput
	}
	claims, err := s.store.ClaimDue(ctx, s.now().UTC(), webhookDeliveryLease, limit)
	if err != nil {
		return 0, err
	}
	completed := 0
	for index := range claims {
		if err := ctx.Err(); err != nil {
			return completed, err
		}
		ok, err := s.dispatch(ctx, claims[index])
		if err != nil {
			return completed, err
		}
		if ok {
			completed++
		}
	}
	return completed, nil
}

func (s *service) ReplayWebhookDelivery(
	ctx context.Context,
	organizationID uint,
	deliveryID uint64,
	actorID uint,
) (*domain.WebhookDelivery, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	delivery, err := s.store.Replay(ctx, organizationID, deliveryID, actorID, s.now().UTC())
	if err != nil {
		return nil, err
	}
	recordWebhookAudit(ctx, "replay", delivery.EndpointID, map[string]any{
		"delivery_id":  delivery.ID,
		"message_id":   delivery.MessageID,
		"replay_count": delivery.ReplayCount,
	})
	return delivery, nil
}

func (s *service) PruneWebhookHistory(
	ctx context.Context,
	before time.Time,
) (*domain.WebhookPruneResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	cutoff := before.UTC()
	if cutoff.IsZero() || cutoff.After(now.Add(-s.retention)) {
		return nil, domain.ErrInvalidInput
	}
	result, err := s.store.Prune(ctx, cutoff, now)
	if err != nil {
		return nil, err
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "prune",
		Resource:   "webhook-history",
		TargetType: "webhook_history",
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"before":     cutoff,
			"attempts":   result.Attempts,
			"deliveries": result.Deliveries,
			"events":     result.Events,
			"secrets":    result.Secrets,
		},
	})
	return result, nil
}

type normalizedEndpointInput struct {
	Name       string
	URL        string
	URLHash    string
	EventTypes []string
}

func (s *service) normalizeEndpointInput(ctx context.Context, input endpointInput) (normalizedEndpointInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || !utf8.ValidString(input.Name) || utf8.RuneCountInString(input.Name) > 100 || containsWebhookControl(input.Name) {
		return normalizedEndpointInput{}, domain.ErrInvalidInput
	}
	eventTypes := slices.Clone(input.EventTypes)
	slices.Sort(eventTypes)
	if len(eventTypes) == 0 || len(eventTypes) > maxWebhookDefinitions {
		return normalizedEndpointInput{}, domain.ErrWebhookInvalidEventType
	}
	for index, eventType := range eventTypes {
		if !s.catalog.Contains(eventType) || (index > 0 && eventTypes[index-1] == eventType) {
			return normalizedEndpointInput{}, domain.ErrWebhookInvalidEventType
		}
	}
	normalizedURL, urlHash, err := s.targets.Normalize(ctx, input.URL)
	if err != nil {
		return normalizedEndpointInput{}, err
	}
	return normalizedEndpointInput{
		Name:       input.Name,
		URL:        normalizedURL,
		URLHash:    urlHash,
		EventTypes: eventTypes,
	}, nil
}

func (s *service) normalizePublication(
	event domain.WebhookEvent,
	targetEndpoint uint,
) (publicationMutation, error) {
	now := s.now().UTC()
	if event.OrganizationID == 0 || !validWebhookSource(event.Source) || !validWebhookEventID(event.EventID) || event.OccurredAt.IsZero() {
		return publicationMutation{}, domain.ErrInvalidInput
	}
	occurredAt := event.OccurredAt.UTC()
	if occurredAt.Before(now.Add(-webhookMaximumEventAge)) || occurredAt.After(now.Add(webhookMaximumFutureSkew)) {
		return publicationMutation{}, domain.ErrInvalidInput
	}
	payload, err := s.catalog.Normalize(event.Type, event.Data)
	if err != nil {
		return publicationMutation{}, err
	}
	fingerprint, err := webhookEventFingerprint(event.OrganizationID, event.Source, event.EventID, event.Type, occurredAt, payload)
	if err != nil {
		return publicationMutation{}, err
	}
	return publicationMutation{
		OrganizationID: event.OrganizationID,
		TargetEndpoint: targetEndpoint,
		MessageID:      "msg_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		Source:         event.Source,
		EventID:        event.EventID,
		Fingerprint:    fingerprint,
		EventType:      event.Type,
		PayloadJSON:    payload,
		OccurredAt:     occurredAt,
		Now:            now,
	}, nil
}

func (s *service) dispatch(ctx context.Context, claim claimedDelivery) (bool, error) {
	started := s.now().UTC()
	if claim.DestinationHash != claim.CurrentDestinationHash {
		return s.completeDispatch(ctx, claim, sendResult{FailureCode: "WEBHOOK.ROUTE_CHANGED"}, started)
	}
	current, err := s.secrets.Decrypt(claim.SecretCiphertext)
	if err != nil {
		return false, err
	}
	defer clear(current)
	var previous []byte
	if claim.PreviousSecretCiphertext != "" && claim.PreviousSecretExpires != nil && started.Before(*claim.PreviousSecretExpires) {
		previous, err = s.secrets.Decrypt(claim.PreviousSecretCiphertext)
		if err != nil {
			return false, err
		}
		defer clear(previous)
	}
	result, err := s.sender.Send(ctx, outboundWebhook{
		URL:             claim.URL,
		MessageID:       claim.MessageID,
		EventType:       claim.EventType,
		OccurredAt:      claim.OccurredAt,
		PayloadJSON:     claim.PayloadJSON,
		CurrentSecret:   current,
		PreviousSecret:  previous,
		PreviousExpires: claim.PreviousSecretExpires,
	})
	if err != nil {
		return false, err
	}
	return s.completeDispatch(ctx, claim, result, started)
}

func (s *service) completeDispatch(
	ctx context.Context,
	claim claimedDelivery,
	result sendResult,
	started time.Time,
) (bool, error) {
	now := s.now().UTC()
	completion := deliveryCompletion{
		HTTPStatus:        result.HTTPStatus,
		FailureCode:       result.FailureCode,
		ResponseTruncated: result.ResponseTruncated,
		StartedAt:         started,
		CompletedAt:       now,
		DurationMS:        durationMilliseconds(result.Duration),
	}
	switch {
	case result.FailureCode == "":
		completion.Status = deliveryStatusDelivered
		completion.Outcome = "delivered"
	case result.Retryable && claim.CycleAttempt < webhookMaximumAttempts:
		retryAt := now.Add(webhookRetryDelay(claim.MessageID, claim.CycleAttempt))
		completion.Status = deliveryStatusPending
		completion.Outcome = "retry_scheduled"
		completion.RetryAt = &retryAt
	default:
		completion.Status = deliveryStatusFailed
		completion.Outcome = "failed"
		if result.Retryable && claim.CycleAttempt >= webhookMaximumAttempts {
			completion.FailureCode = "WEBHOOK.RETRY_EXHAUSTED"
		}
	}
	return s.store.Complete(ctx, claim, completion, webhookDisableAfterFailure)
}

func (s *service) available() error {
	if s == nil || !s.enabled || s.catalog == nil || s.store == nil || s.secrets == nil ||
		s.targets == nil || s.sender == nil || s.now == nil || s.secretOverlap <= 0 || s.retention <= 0 {
		return domain.ErrServiceUnavailable
	}
	return nil
}

func webhookEventFingerprint(
	organizationID uint,
	source string,
	eventID string,
	eventType string,
	occurredAt time.Time,
	payload string,
) (string, error) {
	encoded, err := json.Marshal(struct {
		OrganizationID uint   `json:"organization_id"`
		Source         string `json:"source"`
		EventID        string `json:"event_id"`
		Type           string `json:"type"`
		OccurredAt     string `json:"occurred_at"`
		Payload        string `json:"payload"`
	}{
		OrganizationID: organizationID,
		Source:         source,
		EventID:        eventID,
		Type:           eventType,
		OccurredAt:     occurredAt.UTC().Format(time.RFC3339Nano),
		Payload:        payload,
	})
	if err != nil {
		return "", fmt.Errorf("encode webhook fingerprint: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func webhookRetryDelay(messageID string, attempt uint8) time.Duration {
	schedule := [...]time.Duration{
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		time.Hour,
		3 * time.Hour,
		6 * time.Hour,
		12 * time.Hour,
		18 * time.Hour,
		24 * time.Hour,
	}
	index := int(attempt) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(schedule) {
		index = len(schedule) - 1
	}
	digest := sha256.Sum256([]byte(messageID + ":" + strconv.Itoa(int(attempt))))
	percent := 80 + int(digest[0])%41
	return schedule[index] * time.Duration(percent) / 100
}

func durationMilliseconds(value time.Duration) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value / time.Millisecond)
}

func validDeliveryStatus(status domain.WebhookDeliveryStatus) bool {
	switch status {
	case domain.WebhookDeliveryStatusPending,
		domain.WebhookDeliveryStatusProcessing,
		domain.WebhookDeliveryStatusDelivered,
		domain.WebhookDeliveryStatusFailed,
		domain.WebhookDeliveryStatusCanceled:
		return true
	default:
		return false
	}
}

func containsWebhookControl(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}

func recordWebhookAudit(ctx context.Context, action string, endpointID uint, metadata map[string]any) {
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     action,
		Resource:   "webhook-endpoints",
		TargetType: "webhook_endpoint",
		TargetID:   strconv.FormatUint(uint64(endpointID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata:   metadata,
	})
}

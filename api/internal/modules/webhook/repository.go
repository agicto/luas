package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/modules/organization"
)

const webhookPruneBatch = 1000

type endpointMutation struct {
	OrganizationID   uint
	ActorID          uint
	Name             string
	URL              string
	URLHash          string
	EventTypes       []string
	SecretCiphertext string
	SecretHint       string
	ExpectedVersion  uint64
	Now              time.Time
}

type publicationMutation struct {
	OrganizationID uint
	TargetEndpoint uint
	MessageID      string
	Source         string
	EventID        string
	Fingerprint    string
	EventType      string
	PayloadJSON    string
	OccurredAt     time.Time
	Now            time.Time
}

type publicationResult struct {
	Receipt    *domain.WebhookReceipt
	Deliveries []*domain.WebhookDelivery
}

type deliveryFilter struct {
	EndpointID uint
	Status     domain.WebhookDeliveryStatus
}

type claimedDelivery struct {
	ID                       uint64
	EndpointID               uint
	LeaseToken               string
	AttemptNumber            uint16
	CycleAttempt             uint8
	DestinationHash          string
	CurrentDestinationHash   string
	URL                      string
	SecretCiphertext         string
	PreviousSecretCiphertext string
	PreviousSecretExpires    *time.Time
	MessageID                string
	EventType                string
	PayloadJSON              string
	OccurredAt               time.Time
}

type deliveryCompletion struct {
	Status            string
	Outcome           string
	HTTPStatus        *int
	FailureCode       string
	ResponseTruncated bool
	RetryAt           *time.Time
	StartedAt         time.Time
	CompletedAt       time.Time
	DurationMS        uint64
}

type webhookStore interface {
	ListEndpoints(context.Context, uint, int, int) ([]*domain.WebhookEndpoint, int64, error)
	CreateEndpoint(context.Context, endpointMutation) (*domain.WebhookEndpoint, error)
	UpdateEndpoint(context.Context, uint, endpointMutation) (*domain.WebhookEndpoint, error)
	ReplaceEndpointStatus(context.Context, uint, uint, uint, bool, uint64, time.Time) (*domain.WebhookEndpoint, error)
	DeleteEndpoint(context.Context, uint, uint, uint, uint64, time.Time) error
	RotateEndpointSecret(context.Context, uint, endpointMutation, time.Time) (*domain.WebhookEndpoint, error)
	Publish(context.Context, publicationMutation) (*publicationResult, error)
	ListDeliveries(context.Context, uint, deliveryFilter, int, int) ([]*domain.WebhookDelivery, int64, error)
	ListAttempts(context.Context, uint, uint64, int, int) ([]*domain.WebhookAttempt, int64, error)
	ClaimDue(context.Context, time.Time, time.Duration, int) ([]claimedDelivery, error)
	Complete(context.Context, claimedDelivery, deliveryCompletion, uint8) (bool, error)
	Replay(context.Context, uint, uint64, uint, time.Time) (*domain.WebhookDelivery, error)
	Prune(context.Context, time.Time, time.Time) (*domain.WebhookPruneResult, error)
}

type repository struct {
	db *gorm.DB
}

var _ webhookStore = (*repository)(nil)

// NewRepository creates the durable webhook outbox and delivery ledger adapter.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) ListEndpoints(
	ctx context.Context,
	organizationID uint,
	page int,
	pageSize int,
) ([]*domain.WebhookEndpoint, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	query := db.Model(&EndpointPO{}).Where("organization_id = ?", organizationID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count webhook endpoints: %w", err)
	}
	var rows []EndpointPO
	if err := query.
		Preload("Subscriptions", func(value *gorm.DB) *gorm.DB { return value.Order("event_type ASC") }).
		Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list webhook endpoints: %w", err)
	}
	result := make([]*domain.WebhookEndpoint, len(rows))
	for index := range rows {
		result[index] = endpointFromPO(&rows[index])
	}
	return result, total, nil
}

func (r *repository) CreateEndpoint(ctx context.Context, mutation endpointMutation) (*domain.WebhookEndpoint, error) {
	if err := validateEndpointMutation(mutation, false); err != nil {
		return nil, err
	}
	var result *domain.WebhookEndpoint
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		var owner organization.OrganizationPO
		if err := tx.Select("id").First(&owner, mutation.OrganizationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrOrganizationNotFound
			}
			return fmt.Errorf("find webhook endpoint organization: %w", err)
		}
		po := &EndpointPO{
			OrganizationID:      mutation.OrganizationID,
			Name:                mutation.Name,
			URL:                 mutation.URL,
			URLHash:             mutation.URLHash,
			Status:              endpointStatusActive,
			Version:             1,
			SecretCiphertext:    mutation.SecretCiphertext,
			SecretHint:          mutation.SecretHint,
			SecretVersion:       1,
			CreatedBy:           mutation.ActorID,
			UpdatedBy:           mutation.ActorID,
			CreatedAt:           mutation.Now,
			UpdatedAt:           mutation.Now,
			ConsecutiveFailures: 0,
		}
		if err := tx.Create(po).Error; err != nil {
			return fmt.Errorf("create webhook endpoint: %w", err)
		}
		po.Subscriptions = newSubscriptions(po.ID, mutation.OrganizationID, mutation.EventTypes, mutation.Now)
		if err := tx.Create(&po.Subscriptions).Error; err != nil {
			return fmt.Errorf("create webhook subscriptions: %w", err)
		}
		result = endpointFromPO(po)
		return nil
	})
	return result, err
}

func (r *repository) UpdateEndpoint(
	ctx context.Context,
	endpointID uint,
	mutation endpointMutation,
) (*domain.WebhookEndpoint, error) {
	if endpointID == 0 || mutation.ExpectedVersion == 0 {
		return nil, domain.ErrInvalidInput
	}
	if err := validateEndpointMutation(mutation, true); err != nil {
		return nil, err
	}
	var result *domain.WebhookEndpoint
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		po, err := findEndpointForUpdate(tx, mutation.OrganizationID, endpointID)
		if err != nil {
			return err
		}
		if po.Version != mutation.ExpectedVersion {
			return domain.ErrWebhookEndpointVersionConflict
		}
		urlChanged := po.URLHash != mutation.URLHash
		update := tx.Model(&EndpointPO{}).
			Where("id = ? AND organization_id = ? AND version = ? AND deleted_at IS NULL", endpointID, mutation.OrganizationID, mutation.ExpectedVersion).
			Updates(map[string]any{
				"name":       mutation.Name,
				"url":        mutation.URL,
				"url_hash":   mutation.URLHash,
				"version":    mutation.ExpectedVersion + 1,
				"updated_by": mutation.ActorID,
				"updated_at": mutation.Now,
			})
		if update.Error != nil {
			return fmt.Errorf("update webhook endpoint: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrWebhookEndpointVersionConflict
		}
		if err := tx.Where("endpoint_id = ?", endpointID).Delete(&SubscriptionPO{}).Error; err != nil {
			return fmt.Errorf("replace webhook subscriptions: %w", err)
		}
		subscriptions := newSubscriptions(endpointID, mutation.OrganizationID, mutation.EventTypes, mutation.Now)
		if err := tx.Create(&subscriptions).Error; err != nil {
			return fmt.Errorf("create replacement webhook subscriptions: %w", err)
		}
		if urlChanged {
			if err := cancelOpenDeliveries(tx, endpointID, mutation.Now); err != nil {
				return err
			}
		}
		po.Name = mutation.Name
		po.URL = mutation.URL
		po.URLHash = mutation.URLHash
		po.Version++
		po.UpdatedBy = mutation.ActorID
		po.UpdatedAt = mutation.Now
		po.Subscriptions = subscriptions
		result = endpointFromPO(po)
		return nil
	})
	return result, err
}

func (r *repository) ReplaceEndpointStatus(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	enabled bool,
	expectedVersion uint64,
	now time.Time,
) (*domain.WebhookEndpoint, error) {
	if organizationID == 0 || endpointID == 0 || actorID == 0 || expectedVersion == 0 || now.IsZero() {
		return nil, domain.ErrInvalidInput
	}
	var result *domain.WebhookEndpoint
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		po, err := findEndpointForUpdate(tx, organizationID, endpointID)
		if err != nil {
			return err
		}
		if po.Version != expectedVersion {
			return domain.ErrWebhookEndpointVersionConflict
		}
		status := endpointStatusDisabled
		reason := "manually_disabled"
		if enabled {
			status = endpointStatusActive
			reason = ""
		}
		update := tx.Model(&EndpointPO{}).
			Where("id = ? AND organization_id = ? AND version = ? AND deleted_at IS NULL", endpointID, organizationID, expectedVersion).
			Updates(map[string]any{
				"status":               status,
				"disabled_reason":      reason,
				"consecutive_failures": 0,
				"version":              expectedVersion + 1,
				"updated_by":           actorID,
				"updated_at":           now,
			})
		if update.Error != nil {
			return fmt.Errorf("replace webhook endpoint status: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrWebhookEndpointVersionConflict
		}
		if !enabled {
			if err := cancelOpenDeliveries(tx, endpointID, now); err != nil {
				return err
			}
		}
		po.Status = status
		po.DisabledReason = reason
		po.ConsecutiveFailures = 0
		po.Version++
		po.UpdatedBy = actorID
		po.UpdatedAt = now
		result = endpointFromPO(po)
		return nil
	})
	return result, err
}

func (r *repository) DeleteEndpoint(
	ctx context.Context,
	organizationID uint,
	endpointID uint,
	actorID uint,
	expectedVersion uint64,
	now time.Time,
) error {
	if organizationID == 0 || endpointID == 0 || actorID == 0 || expectedVersion == 0 || now.IsZero() {
		return domain.ErrInvalidInput
	}
	return r.inTransaction(ctx, func(tx *gorm.DB) error {
		po, err := findEndpointForUpdate(tx, organizationID, endpointID)
		if err != nil {
			return err
		}
		if po.Version != expectedVersion {
			return domain.ErrWebhookEndpointVersionConflict
		}
		update := tx.Model(&EndpointPO{}).
			Where("id = ? AND organization_id = ? AND version = ? AND deleted_at IS NULL", endpointID, organizationID, expectedVersion).
			Updates(map[string]any{
				"url":                         "https://deleted.invalid/",
				"url_hash":                    deletedWebhookURLHash,
				"status":                      endpointStatusDisabled,
				"disabled_reason":             "deleted",
				"secret_ciphertext":           "",
				"secret_hint":                 "deleted",
				"previous_secret_ciphertext":  "",
				"previous_secret_valid_until": nil,
				"version":                     expectedVersion + 1,
				"updated_by":                  actorID,
				"updated_at":                  now,
				"deleted_at":                  now,
			})
		if update.Error != nil {
			return fmt.Errorf("delete webhook endpoint: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrWebhookEndpointVersionConflict
		}
		if err := tx.Where("endpoint_id = ?", endpointID).Delete(&SubscriptionPO{}).Error; err != nil {
			return fmt.Errorf("delete webhook subscriptions: %w", err)
		}
		return cancelOpenDeliveries(tx, endpointID, now)
	})
}

func (r *repository) RotateEndpointSecret(
	ctx context.Context,
	endpointID uint,
	mutation endpointMutation,
	previousValidUntil time.Time,
) (*domain.WebhookEndpoint, error) {
	if endpointID == 0 || mutation.OrganizationID == 0 || mutation.ActorID == 0 || mutation.ExpectedVersion == 0 ||
		mutation.SecretCiphertext == "" || mutation.SecretHint == "" || mutation.Now.IsZero() ||
		!previousValidUntil.After(mutation.Now) {
		return nil, domain.ErrInvalidInput
	}
	var result *domain.WebhookEndpoint
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		po, err := findEndpointForUpdate(tx, mutation.OrganizationID, endpointID)
		if err != nil {
			return err
		}
		if po.Version != mutation.ExpectedVersion {
			return domain.ErrWebhookEndpointVersionConflict
		}
		update := tx.Model(&EndpointPO{}).
			Where("id = ? AND organization_id = ? AND version = ? AND deleted_at IS NULL", endpointID, mutation.OrganizationID, mutation.ExpectedVersion).
			Updates(map[string]any{
				"previous_secret_ciphertext":  po.SecretCiphertext,
				"previous_secret_valid_until": previousValidUntil,
				"secret_ciphertext":           mutation.SecretCiphertext,
				"secret_hint":                 mutation.SecretHint,
				"secret_version":              po.SecretVersion + 1,
				"version":                     mutation.ExpectedVersion + 1,
				"updated_by":                  mutation.ActorID,
				"updated_at":                  mutation.Now,
			})
		if update.Error != nil {
			return fmt.Errorf("rotate webhook endpoint secret: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrWebhookEndpointVersionConflict
		}
		po.PreviousSecretCiphertext = po.SecretCiphertext
		po.PreviousSecretValidUntil = cloneTime(&previousValidUntil)
		po.SecretCiphertext = mutation.SecretCiphertext
		po.SecretHint = mutation.SecretHint
		po.SecretVersion++
		po.Version++
		po.UpdatedBy = mutation.ActorID
		po.UpdatedAt = mutation.Now
		result = endpointFromPO(po)
		return nil
	})
	return result, err
}

func (r *repository) Publish(ctx context.Context, mutation publicationMutation) (*publicationResult, error) {
	if err := validatePublicationMutation(mutation); err != nil {
		return nil, err
	}
	var result *publicationResult
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		po := &EventPO{
			OrganizationID:  mutation.OrganizationID,
			MessageID:       mutation.MessageID,
			Source:          mutation.Source,
			ProducerEventID: mutation.EventID,
			Fingerprint:     mutation.Fingerprint,
			EventType:       mutation.EventType,
			PayloadJSON:     mutation.PayloadJSON,
			OccurredAt:      mutation.OccurredAt,
			CreatedAt:       mutation.Now,
		}
		create := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "source"}, {Name: "event_id"}},
			DoNothing: true,
		}).Create(po)
		if create.Error != nil {
			return fmt.Errorf("create webhook event: %w", create.Error)
		}
		created := create.RowsAffected == 1
		if !created {
			if err := tx.Where(
				"organization_id = ? AND source = ? AND event_id = ?",
				mutation.OrganizationID,
				mutation.Source,
				mutation.EventID,
			).First(po).Error; err != nil {
				return fmt.Errorf("find idempotent webhook event: %w", err)
			}
			if po.Fingerprint != mutation.Fingerprint {
				return domain.ErrWebhookIdempotencyConflict
			}
		}

		if created {
			endpoints, err := subscribedEndpoints(tx, mutation)
			if err != nil {
				return err
			}
			rows := make([]DeliveryPO, len(endpoints))
			for index := range endpoints {
				rows[index] = DeliveryPO{
					OrganizationID:  mutation.OrganizationID,
					EndpointID:      endpoints[index].ID,
					EventID:         po.ID,
					DestinationHash: endpoints[index].URLHash,
					Status:          deliveryStatusPending,
					AvailableAt:     mutation.Now,
					CreatedAt:       mutation.Now,
					UpdatedAt:       mutation.Now,
				}
			}
			if len(rows) > 0 {
				if err := tx.Create(&rows).Error; err != nil {
					return fmt.Errorf("create webhook deliveries: %w", err)
				}
			}
		}

		deliveries, err := deliveriesForEvent(tx, po, mutation.TargetEndpoint)
		if err != nil {
			return err
		}
		if mutation.TargetEndpoint != 0 && len(deliveries) != 1 {
			return domain.ErrWebhookReplayNotAllowed
		}
		result = &publicationResult{
			Receipt: &domain.WebhookReceipt{
				ID:             po.ID,
				MessageID:      po.MessageID,
				OrganizationID: po.OrganizationID,
				Type:           po.EventType,
				DeliveryCount:  len(deliveries),
				Created:        created,
				OccurredAt:     po.OccurredAt,
				CreatedAt:      po.CreatedAt,
			},
			Deliveries: deliveries,
		}
		return nil
	})
	return result, err
}

func (r *repository) ListDeliveries(
	ctx context.Context,
	organizationID uint,
	filter deliveryFilter,
	page int,
	pageSize int,
) ([]*domain.WebhookDelivery, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	query := db.Model(&DeliveryPO{}).Where("organization_id = ?", organizationID)
	if filter.EndpointID != 0 {
		query = query.Where("endpoint_id = ?", filter.EndpointID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}
	var rows []DeliveryPO
	if err := query.Preload("Event").
		Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list webhook deliveries: %w", err)
	}
	result := make([]*domain.WebhookDelivery, len(rows))
	for index := range rows {
		result[index] = deliveryFromPO(&rows[index])
	}
	return result, total, nil
}

func (r *repository) ListAttempts(
	ctx context.Context,
	organizationID uint,
	deliveryID uint64,
	page int,
	pageSize int,
) ([]*domain.WebhookAttempt, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if organizationID == 0 || deliveryID == 0 || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	var deliveryCount int64
	if err := db.Model(&DeliveryPO{}).
		Where("id = ? AND organization_id = ?", deliveryID, organizationID).
		Count(&deliveryCount).Error; err != nil {
		return nil, 0, fmt.Errorf("find webhook delivery attempts owner: %w", err)
	}
	if deliveryCount != 1 {
		return nil, 0, domain.ErrWebhookDeliveryNotFound
	}
	query := db.Model(&AttemptPO{}).
		Where("organization_id = ? AND delivery_id = ?", organizationID, deliveryID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count webhook attempts: %w", err)
	}
	var rows []AttemptPO
	if err := query.Order("number DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list webhook attempts: %w", err)
	}
	result := make([]*domain.WebhookAttempt, len(rows))
	for index := range rows {
		result[index] = attemptFromPO(&rows[index])
	}
	return result, total, nil
}

func (r *repository) ClaimDue(
	ctx context.Context,
	now time.Time,
	lease time.Duration,
	limit int,
) ([]claimedDelivery, error) {
	if now.IsZero() || lease <= 0 || limit < 1 || limit > 100 {
		return nil, domain.ErrInvalidInput
	}
	result := make([]claimedDelivery, 0, limit)
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Model(&DeliveryPO{}).
			Where("status = ? AND lease_expires_at <= ?", deliveryStatusProcessing, now).
			Updates(map[string]any{
				"status":           deliveryStatusPending,
				"lease_token":      "",
				"lease_expires_at": nil,
				"updated_at":       now,
			}).Error; err != nil {
			return fmt.Errorf("release expired webhook leases: %w", err)
		}

		query := tx.Model(&DeliveryPO{}).
			Preload("Endpoint").
			Preload("Event").
			Joins("JOIN webhook_endpoints ON webhook_endpoints.id = webhook_deliveries.endpoint_id").
			Where(
				"webhook_deliveries.status = ? AND webhook_deliveries.available_at <= ? AND webhook_endpoints.status = ? AND webhook_endpoints.deleted_at IS NULL",
				deliveryStatusPending,
				now,
				endpointStatusActive,
			).
			Order("webhook_deliveries.available_at ASC, webhook_deliveries.id ASC").
			Limit(limit)
		query = webhookClaimLockQuery(query)
		var rows []DeliveryPO
		if err := query.Find(&rows).Error; err != nil {
			return fmt.Errorf("claim due webhook deliveries: %w", err)
		}
		for index := range rows {
			row := &rows[index]
			if row.Endpoint == nil || row.Event == nil {
				return domain.ErrServiceUnavailable
			}
			token := uuid.NewString()
			expires := now.Add(lease)
			update := tx.Model(&DeliveryPO{}).
				Where("id = ? AND status = ?", row.ID, deliveryStatusPending).
				Updates(map[string]any{
					"status":           deliveryStatusProcessing,
					"attempt_count":    row.AttemptCount + 1,
					"cycle_attempt":    row.CycleAttempt + 1,
					"lease_token":      token,
					"lease_expires_at": expires,
					"updated_at":       now,
				})
			if update.Error != nil {
				return fmt.Errorf("lease webhook delivery: %w", update.Error)
			}
			if update.RowsAffected != 1 {
				continue
			}
			result = append(result, claimedDelivery{
				ID:                       row.ID,
				EndpointID:               row.EndpointID,
				LeaseToken:               token,
				AttemptNumber:            row.AttemptCount + 1,
				CycleAttempt:             row.CycleAttempt + 1,
				DestinationHash:          row.DestinationHash,
				CurrentDestinationHash:   row.Endpoint.URLHash,
				URL:                      row.Endpoint.URL,
				SecretCiphertext:         row.Endpoint.SecretCiphertext,
				PreviousSecretCiphertext: row.Endpoint.PreviousSecretCiphertext,
				PreviousSecretExpires:    cloneTime(row.Endpoint.PreviousSecretValidUntil),
				MessageID:                row.Event.MessageID,
				EventType:                row.Event.EventType,
				PayloadJSON:              row.Event.PayloadJSON,
				OccurredAt:               row.Event.OccurredAt,
			})
		}
		return nil
	})
	return result, err
}

func (r *repository) Complete(
	ctx context.Context,
	claim claimedDelivery,
	completion deliveryCompletion,
	disableAfter uint8,
) (bool, error) {
	if claim.ID == 0 || claim.EndpointID == 0 || claim.LeaseToken == "" || claim.AttemptNumber == 0 ||
		completion.StartedAt.IsZero() || completion.CompletedAt.Before(completion.StartedAt) || disableAfter == 0 {
		return false, domain.ErrInvalidInput
	}
	completed := false
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		var delivery DeliveryPO
		query := webhookRowLockQuery(tx.Model(&DeliveryPO{}))
		if err := query.Where(
			"id = ? AND endpoint_id = ? AND status = ? AND lease_token = ?",
			claim.ID,
			claim.EndpointID,
			deliveryStatusProcessing,
			claim.LeaseToken,
		).First(&delivery).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("find leased webhook delivery: %w", err)
		}
		if delivery.AttemptCount != claim.AttemptNumber {
			return nil
		}
		attempt := &AttemptPO{
			OrganizationID:    delivery.OrganizationID,
			DeliveryID:        delivery.ID,
			Number:            claim.AttemptNumber,
			Outcome:           completion.Outcome,
			HTTPStatus:        cloneInt(completion.HTTPStatus),
			FailureCode:       completion.FailureCode,
			DurationMS:        completion.DurationMS,
			ResponseTruncated: completion.ResponseTruncated,
			StartedAt:         completion.StartedAt,
			CompletedAt:       completion.CompletedAt,
		}
		if err := tx.Create(attempt).Error; err != nil {
			return fmt.Errorf("create webhook delivery attempt: %w", err)
		}

		updates := map[string]any{
			"status":             completion.Status,
			"http_status":        completion.HTTPStatus,
			"failure_code":       completion.FailureCode,
			"response_truncated": completion.ResponseTruncated,
			"lease_token":        "",
			"lease_expires_at":   nil,
			"updated_at":         completion.CompletedAt,
		}
		switch completion.Status {
		case deliveryStatusDelivered:
			updates["delivered_at"] = completion.CompletedAt
			updates["cycle_attempt"] = 0
		case deliveryStatusPending:
			if completion.RetryAt == nil || !completion.RetryAt.After(completion.CompletedAt) {
				return domain.ErrInvalidInput
			}
			updates["available_at"] = *completion.RetryAt
		case deliveryStatusFailed:
		default:
			return domain.ErrInvalidInput
		}
		update := tx.Model(&DeliveryPO{}).
			Where("id = ? AND status = ? AND lease_token = ?", delivery.ID, deliveryStatusProcessing, claim.LeaseToken).
			Updates(updates)
		if update.Error != nil {
			return fmt.Errorf("complete webhook delivery: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return nil
		}

		if completion.Status == deliveryStatusDelivered || completion.Status == deliveryStatusFailed {
			endpoint, err := findEndpointForUpdate(tx, delivery.OrganizationID, delivery.EndpointID)
			if err != nil {
				if errors.Is(err, domain.ErrWebhookEndpointNotFound) {
					completed = true
					return nil
				}
				return err
			}
			failures := uint8(0)
			status := endpoint.Status
			reason := endpoint.DisabledReason
			version := endpoint.Version
			if completion.Status == deliveryStatusFailed {
				failures = endpoint.ConsecutiveFailures + 1
				if failures >= disableAfter {
					status = endpointStatusDisabled
					reason = "consecutive_failures"
					version++
				}
			}
			if err := tx.Model(&EndpointPO{}).Where("id = ?", endpoint.ID).Updates(map[string]any{
				"consecutive_failures": failures,
				"status":               status,
				"disabled_reason":      reason,
				"version":              version,
				"updated_at":           completion.CompletedAt,
			}).Error; err != nil {
				return fmt.Errorf("update webhook endpoint delivery health: %w", err)
			}
			if status == endpointStatusDisabled && endpoint.Status != endpointStatusDisabled {
				if err := cancelOpenDeliveries(tx, endpoint.ID, completion.CompletedAt); err != nil {
					return err
				}
			}
		}
		completed = true
		return nil
	})
	return completed, err
}

func (r *repository) Replay(
	ctx context.Context,
	organizationID uint,
	deliveryID uint64,
	actorID uint,
	now time.Time,
) (*domain.WebhookDelivery, error) {
	if organizationID == 0 || deliveryID == 0 || actorID == 0 || now.IsZero() {
		return nil, domain.ErrInvalidInput
	}
	var result *domain.WebhookDelivery
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		var delivery DeliveryPO
		query := webhookRowLockQuery(tx.Model(&DeliveryPO{})).Preload("Event")
		if err := query.Where("id = ? AND organization_id = ?", deliveryID, organizationID).First(&delivery).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrWebhookDeliveryNotFound
			}
			return fmt.Errorf("find webhook delivery for replay: %w", err)
		}
		if !domain.WebhookDeliveryStatus(delivery.Status).IsTerminal() || delivery.ReplayCount >= 100 || delivery.Event == nil {
			return domain.ErrWebhookReplayNotAllowed
		}
		endpoint, err := findEndpointForUpdate(tx, organizationID, delivery.EndpointID)
		if err != nil || endpoint.Status != endpointStatusActive || endpoint.SecretCiphertext == "" {
			return domain.ErrWebhookReplayNotAllowed
		}
		update := tx.Model(&DeliveryPO{}).Where("id = ?", delivery.ID).Updates(map[string]any{
			"destination_hash":   endpoint.URLHash,
			"status":             deliveryStatusPending,
			"cycle_attempt":      0,
			"replay_count":       delivery.ReplayCount + 1,
			"available_at":       now,
			"lease_token":        "",
			"lease_expires_at":   nil,
			"http_status":        nil,
			"failure_code":       "",
			"response_truncated": false,
			"delivered_at":       nil,
			"updated_at":         now,
		})
		if update.Error != nil {
			return fmt.Errorf("replay webhook delivery: %w", update.Error)
		}
		if update.RowsAffected != 1 {
			return domain.ErrWebhookReplayNotAllowed
		}
		delivery.DestinationHash = endpoint.URLHash
		delivery.Status = deliveryStatusPending
		delivery.CycleAttempt = 0
		delivery.ReplayCount++
		delivery.AvailableAt = now
		delivery.LeaseToken = ""
		delivery.LeaseExpiresAt = nil
		delivery.HTTPStatus = nil
		delivery.FailureCode = ""
		delivery.ResponseTruncated = false
		delivery.DeliveredAt = nil
		delivery.UpdatedAt = now
		result = deliveryFromPO(&delivery)
		return nil
	})
	return result, err
}

func (r *repository) Prune(
	ctx context.Context,
	before time.Time,
	now time.Time,
) (*domain.WebhookPruneResult, error) {
	if before.IsZero() || now.IsZero() || !before.Before(now) {
		return nil, domain.ErrInvalidInput
	}
	result := &domain.WebhookPruneResult{}
	err := r.inTransaction(ctx, func(tx *gorm.DB) error {
		secrets := tx.Model(&EndpointPO{}).
			Where("previous_secret_valid_until IS NOT NULL AND previous_secret_valid_until <= ?", now).
			Updates(map[string]any{
				"previous_secret_ciphertext":  "",
				"previous_secret_valid_until": nil,
			})
		if secrets.Error != nil {
			return fmt.Errorf("prune previous webhook secrets: %w", secrets.Error)
		}
		result.Secrets = secrets.RowsAffected

		var deliveryIDs []uint64
		if err := tx.Model(&DeliveryPO{}).
			Select("webhook_deliveries.id").
			Joins("JOIN webhook_events ON webhook_events.id = webhook_deliveries.event_id").
			Where("webhook_events.created_at < ? AND webhook_deliveries.status IN ?", before, []string{
				deliveryStatusDelivered,
				deliveryStatusFailed,
				deliveryStatusCanceled,
			}).
			Order("webhook_deliveries.id ASC").
			Limit(webhookPruneBatch).
			Pluck("webhook_deliveries.id", &deliveryIDs).Error; err != nil {
			return fmt.Errorf("select webhook deliveries to prune: %w", err)
		}
		if len(deliveryIDs) > 0 {
			attempts := tx.Where("delivery_id IN ?", deliveryIDs).Delete(&AttemptPO{})
			if attempts.Error != nil {
				return fmt.Errorf("prune webhook attempts: %w", attempts.Error)
			}
			result.Attempts = attempts.RowsAffected
			deliveries := tx.Where("id IN ?", deliveryIDs).Delete(&DeliveryPO{})
			if deliveries.Error != nil {
				return fmt.Errorf("prune webhook deliveries: %w", deliveries.Error)
			}
			result.Deliveries = deliveries.RowsAffected
		}

		var eventIDs []uint64
		if err := tx.Model(&EventPO{}).
			Select("webhook_events.id").
			Where("webhook_events.created_at < ?", before).
			Where("NOT EXISTS (?)", tx.Model(&DeliveryPO{}).
				Select("1").
				Where("webhook_deliveries.event_id = webhook_events.id")).
			Order("webhook_events.id ASC").
			Limit(webhookPruneBatch).
			Pluck("webhook_events.id", &eventIDs).Error; err != nil {
			return fmt.Errorf("select webhook events to prune: %w", err)
		}
		if len(eventIDs) > 0 {
			events := tx.Where("id IN ?", eventIDs).Delete(&EventPO{})
			if events.Error != nil {
				return fmt.Errorf("prune webhook events: %w", events.Error)
			}
			result.Events = events.RowsAffected
		}
		return nil
	})
	return result, err
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil {
		return nil, domain.ErrServiceUnavailable
	}
	db := infradatabase.ResolveContextDB(ctx, r.db)
	if db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return db, nil
}

func (r *repository) inTransaction(ctx context.Context, operation func(*gorm.DB) error) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if _, bound := infradatabase.TransactionFromContext(ctx); bound {
		return operation(db)
	}
	return db.Transaction(operation)
}

func validateEndpointMutation(mutation endpointMutation, update bool) error {
	if mutation.OrganizationID == 0 || mutation.ActorID == 0 || mutation.Name == "" || mutation.URL == "" ||
		len(mutation.URLHash) != 64 || len(mutation.EventTypes) == 0 || mutation.Now.IsZero() {
		return domain.ErrInvalidInput
	}
	if !update && (mutation.SecretCiphertext == "" || mutation.SecretHint == "") {
		return domain.ErrInvalidInput
	}
	return nil
}

func validatePublicationMutation(mutation publicationMutation) error {
	if mutation.OrganizationID == 0 || mutation.MessageID == "" || !validWebhookSource(mutation.Source) ||
		!validWebhookEventID(mutation.EventID) || len(mutation.Fingerprint) != 64 ||
		!validWebhookEventType(mutation.EventType) || len(mutation.PayloadJSON) < 2 ||
		mutation.OccurredAt.IsZero() || mutation.Now.IsZero() {
		return domain.ErrInvalidInput
	}
	return nil
}

func findEndpointForUpdate(tx *gorm.DB, organizationID uint, endpointID uint) (*EndpointPO, error) {
	var po EndpointPO
	query := webhookRowLockQuery(tx.Model(&EndpointPO{})).
		Preload("Subscriptions", func(value *gorm.DB) *gorm.DB { return value.Order("event_type ASC") })
	if err := query.Where("id = ? AND organization_id = ?", endpointID, organizationID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrWebhookEndpointNotFound
		}
		return nil, fmt.Errorf("find webhook endpoint: %w", err)
	}
	return &po, nil
}

func subscribedEndpoints(tx *gorm.DB, mutation publicationMutation) ([]EndpointPO, error) {
	query := tx.Model(&EndpointPO{}).
		Select("webhook_endpoints.id", "webhook_endpoints.url_hash").
		Joins("JOIN webhook_subscriptions ON webhook_subscriptions.endpoint_id = webhook_endpoints.id").
		Where(
			"webhook_endpoints.organization_id = ? AND webhook_endpoints.status = ? AND webhook_endpoints.deleted_at IS NULL AND webhook_subscriptions.event_type = ?",
			mutation.OrganizationID,
			endpointStatusActive,
			mutation.EventType,
		)
	if mutation.TargetEndpoint != 0 {
		query = query.Where("webhook_endpoints.id = ?", mutation.TargetEndpoint)
	}
	var rows []EndpointPO
	if err := query.Order("webhook_endpoints.id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find subscribed webhook endpoints: %w", err)
	}
	if mutation.TargetEndpoint != 0 && len(rows) != 1 {
		var endpoint EndpointPO
		if err := tx.Where("id = ? AND organization_id = ?", mutation.TargetEndpoint, mutation.OrganizationID).First(&endpoint).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, domain.ErrWebhookEndpointNotFound
			}
			return nil, fmt.Errorf("find webhook test endpoint: %w", err)
		}
		return nil, domain.ErrWebhookReplayNotAllowed
	}
	return rows, nil
}

func deliveriesForEvent(tx *gorm.DB, event *EventPO, endpointID uint) ([]*domain.WebhookDelivery, error) {
	var rows []DeliveryPO
	query := tx.Where("event_id = ?", event.ID)
	if endpointID != 0 {
		query = query.Where("endpoint_id = ?", endpointID)
	}
	if err := query.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find webhook event deliveries: %w", err)
	}
	result := make([]*domain.WebhookDelivery, len(rows))
	for index := range rows {
		rows[index].Event = event
		result[index] = deliveryFromPO(&rows[index])
	}
	return result, nil
}

func newSubscriptions(endpointID uint, organizationID uint, eventTypes []string, now time.Time) []SubscriptionPO {
	result := make([]SubscriptionPO, len(eventTypes))
	for index, eventType := range eventTypes {
		result[index] = SubscriptionPO{
			EndpointID:     endpointID,
			OrganizationID: organizationID,
			EventType:      eventType,
			CreatedAt:      now,
		}
	}
	return result
}

func cancelOpenDeliveries(tx *gorm.DB, endpointID uint, now time.Time) error {
	if err := tx.Model(&DeliveryPO{}).
		Where("endpoint_id = ? AND status IN ?", endpointID, []string{deliveryStatusPending, deliveryStatusProcessing}).
		Updates(map[string]any{
			"status":           deliveryStatusCanceled,
			"lease_token":      "",
			"lease_expires_at": nil,
			"failure_code":     "WEBHOOK.ENDPOINT_CHANGED",
			"updated_at":       now,
		}).Error; err != nil {
		return fmt.Errorf("cancel open webhook deliveries: %w", err)
	}
	return nil
}

func webhookRowLockQuery(db *gorm.DB) *gorm.DB {
	if db == nil || db.Dialector == nil {
		return db
	}
	switch db.Name() {
	case "postgres", "mysql":
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	default:
		return db
	}
}

func webhookClaimLockQuery(db *gorm.DB) *gorm.DB {
	if db == nil || db.Dialector == nil {
		return db
	}
	switch db.Name() {
	case "postgres", "mysql":
		return db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	default:
		return db
	}
}

const deletedWebhookURLHash = "6cfe132f65419f94a7ab8d973d3925a8837f3c396884a9146fb50a2abac317fa"

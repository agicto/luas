package webhook

import (
	"slices"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/organization"
)

const (
	endpointStatusActive   = "active"
	endpointStatusDisabled = "disabled"

	deliveryStatusPending    = "pending"
	deliveryStatusProcessing = "processing"
	deliveryStatusDelivered  = "delivered"
	deliveryStatusFailed     = "failed"
	deliveryStatusCanceled   = "canceled"
)

// EndpointPO stores one organization-owned outbound target and encrypted signing material.
type EndpointPO struct {
	ID                       uint       `gorm:"primaryKey;index:idx_webhook_endpoints_organization_status,priority:3;index:idx_webhook_endpoints_organization_created,priority:3"`
	OrganizationID           uint       `gorm:"not null;index:idx_webhook_endpoints_organization_status,priority:1;index:idx_webhook_endpoints_organization_created,priority:1"`
	Name                     string     `gorm:"size:100;not null;check:webhook_endpoints_name_check,length(name) BETWEEN 1 AND 100"`
	URL                      string     `gorm:"size:2048;not null;check:webhook_endpoints_url_check,length(url) BETWEEN 1 AND 2048"`
	URLHash                  string     `gorm:"size:64;not null;check:webhook_endpoints_url_hash_check,length(url_hash) = 64"`
	Status                   string     `gorm:"size:16;not null;index:idx_webhook_endpoints_organization_status,priority:2;check:webhook_endpoints_status_check,status IN ('active','disabled')"`
	DisabledReason           string     `gorm:"size:64;not null;default:''"`
	ConsecutiveFailures      uint8      `gorm:"not null;default:0"`
	Version                  uint64     `gorm:"not null;check:webhook_endpoints_version_check,version > 0"`
	SecretCiphertext         string     `gorm:"type:text;not null"`
	SecretHint               string     `gorm:"size:16;not null"`
	SecretVersion            uint64     `gorm:"not null;check:webhook_endpoints_secret_version_check,secret_version > 0"`
	PreviousSecretCiphertext string     `gorm:"type:text;not null;default:''"`
	PreviousSecretValidUntil *time.Time `gorm:"index:idx_webhook_endpoints_previous_secret_expiry"`
	CreatedBy                uint       `gorm:"not null"`
	UpdatedBy                uint       `gorm:"not null"`
	CreatedAt                time.Time  `gorm:"index:idx_webhook_endpoints_organization_created,priority:2"`
	UpdatedAt                time.Time
	DeletedAt                gorm.DeletedAt `gorm:"index"`

	Organization  *organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Subscriptions []SubscriptionPO             `gorm:"foreignKey:EndpointID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (EndpointPO) TableName() string { return "webhook_endpoints" }

// SubscriptionPO is one exact event selection for an endpoint.
type SubscriptionPO struct {
	ID             uint64 `gorm:"primaryKey"`
	EndpointID     uint   `gorm:"not null;uniqueIndex:idx_webhook_subscriptions_endpoint_type,priority:1"`
	OrganizationID uint   `gorm:"not null;index:idx_webhook_subscriptions_organization_type,priority:1"`
	EventType      string `gorm:"size:100;not null;uniqueIndex:idx_webhook_subscriptions_endpoint_type,priority:2;index:idx_webhook_subscriptions_organization_type,priority:2"`
	CreatedAt      time.Time
}

func (SubscriptionPO) TableName() string { return "webhook_subscriptions" }

// EventPO is one minimized durable publication and payload retained for delivery/replay.
type EventPO struct {
	ID              uint64    `gorm:"primaryKey;index:idx_webhook_events_created,priority:2"`
	OrganizationID  uint      `gorm:"not null;index:idx_webhook_events_organization_created,priority:1;uniqueIndex:idx_webhook_events_identity,priority:1"`
	MessageID       string    `gorm:"size:64;not null;uniqueIndex"`
	Source          string    `gorm:"size:64;not null;uniqueIndex:idx_webhook_events_identity,priority:2"`
	ProducerEventID string    `gorm:"column:event_id;size:128;not null;uniqueIndex:idx_webhook_events_identity,priority:3"`
	Fingerprint     string    `gorm:"size:64;not null;check:webhook_events_fingerprint_check,length(fingerprint) = 64"`
	EventType       string    `gorm:"size:100;not null;index"`
	PayloadJSON     string    `gorm:"type:text;not null;check:webhook_events_payload_check,length(payload_json) BETWEEN 2 AND 65536"`
	OccurredAt      time.Time `gorm:"not null"`
	CreatedAt       time.Time `gorm:"index:idx_webhook_events_organization_created,priority:2;index:idx_webhook_events_created,priority:1"`

	Organization *organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (EventPO) TableName() string { return "webhook_events" }

// DeliveryPO stores one event/endpoint execution state.
type DeliveryPO struct {
	ID                uint64     `gorm:"primaryKey;index:idx_webhook_deliveries_due,priority:3;index:idx_webhook_deliveries_organization_created,priority:3;index:idx_webhook_deliveries_organization_endpoint_created,priority:4;index:idx_webhook_deliveries_organization_status_created,priority:4"`
	OrganizationID    uint       `gorm:"not null;index:idx_webhook_deliveries_organization_created,priority:1;index:idx_webhook_deliveries_organization_endpoint_created,priority:1;index:idx_webhook_deliveries_organization_status_created,priority:1"`
	EndpointID        uint       `gorm:"not null;uniqueIndex:idx_webhook_deliveries_event_endpoint,priority:2;index:idx_webhook_deliveries_endpoint;index:idx_webhook_deliveries_organization_endpoint_created,priority:2"`
	EventID           uint64     `gorm:"not null;uniqueIndex:idx_webhook_deliveries_event_endpoint,priority:1"`
	DestinationHash   string     `gorm:"size:64;not null;check:webhook_deliveries_destination_hash_check,length(destination_hash) = 64"`
	Status            string     `gorm:"size:16;not null;index:idx_webhook_deliveries_due,priority:1;index:idx_webhook_deliveries_lease_expiry,priority:1;index:idx_webhook_deliveries_organization_status_created,priority:2;check:webhook_deliveries_status_check,status IN ('pending','processing','delivered','failed','canceled')"`
	AttemptCount      uint16     `gorm:"not null;default:0"`
	CycleAttempt      uint8      `gorm:"not null;default:0"`
	ReplayCount       uint16     `gorm:"not null;default:0"`
	AvailableAt       time.Time  `gorm:"not null;index:idx_webhook_deliveries_due,priority:2"`
	LeaseToken        string     `gorm:"size:64;not null;default:''"`
	LeaseExpiresAt    *time.Time `gorm:"index:idx_webhook_deliveries_lease_expiry,priority:2"`
	HTTPStatus        *int
	FailureCode       string `gorm:"size:64;not null;default:''"`
	ResponseTruncated bool   `gorm:"not null;default:false"`
	DeliveredAt       *time.Time
	CreatedAt         time.Time `gorm:"index:idx_webhook_deliveries_organization_created,priority:2;index:idx_webhook_deliveries_organization_endpoint_created,priority:3;index:idx_webhook_deliveries_organization_status_created,priority:3"`
	UpdatedAt         time.Time

	Endpoint *EndpointPO `gorm:"foreignKey:EndpointID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Event    *EventPO    `gorm:"foreignKey:EventID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (DeliveryPO) TableName() string { return "webhook_deliveries" }

// AttemptPO stores bounded local outcome metadata and never response content.
type AttemptPO struct {
	ID                uint64 `gorm:"primaryKey"`
	OrganizationID    uint   `gorm:"not null;index"`
	DeliveryID        uint64 `gorm:"not null;uniqueIndex:idx_webhook_attempts_delivery_number,priority:1"`
	Number            uint16 `gorm:"not null;uniqueIndex:idx_webhook_attempts_delivery_number,priority:2"`
	Outcome           string `gorm:"size:32;not null"`
	HTTPStatus        *int
	FailureCode       string    `gorm:"size:64;not null;default:''"`
	DurationMS        uint64    `gorm:"not null"`
	ResponseTruncated bool      `gorm:"not null;default:false"`
	StartedAt         time.Time `gorm:"not null"`
	CompletedAt       time.Time `gorm:"not null"`

	Delivery *DeliveryPO `gorm:"foreignKey:DeliveryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (AttemptPO) TableName() string { return "webhook_delivery_attempts" }

func endpointFromPO(po *EndpointPO) *domain.WebhookEndpoint {
	if po == nil {
		return nil
	}
	eventTypes := make([]string, len(po.Subscriptions))
	for index := range po.Subscriptions {
		eventTypes[index] = po.Subscriptions[index].EventType
	}
	slices.Sort(eventTypes)
	return &domain.WebhookEndpoint{
		ID:                   po.ID,
		OrganizationID:       po.OrganizationID,
		Name:                 po.Name,
		URL:                  po.URL,
		EventTypes:           eventTypes,
		Status:               domain.WebhookEndpointStatus(po.Status),
		DisabledReason:       po.DisabledReason,
		ConsecutiveFailures:  po.ConsecutiveFailures,
		Version:              po.Version,
		SecretHint:           po.SecretHint,
		SecretVersion:        po.SecretVersion,
		PreviousSecretExpiry: cloneTime(po.PreviousSecretValidUntil),
		CreatedAt:            po.CreatedAt,
		UpdatedAt:            po.UpdatedAt,
	}
}

func deliveryFromPO(po *DeliveryPO) *domain.WebhookDelivery {
	if po == nil {
		return nil
	}
	result := &domain.WebhookDelivery{
		ID:                po.ID,
		OrganizationID:    po.OrganizationID,
		EndpointID:        po.EndpointID,
		Status:            domain.WebhookDeliveryStatus(po.Status),
		AttemptCount:      po.AttemptCount,
		ReplayCount:       po.ReplayCount,
		HTTPStatus:        cloneInt(po.HTTPStatus),
		FailureCode:       po.FailureCode,
		ResponseTruncated: po.ResponseTruncated,
		AvailableAt:       po.AvailableAt,
		DeliveredAt:       cloneTime(po.DeliveredAt),
		CreatedAt:         po.CreatedAt,
		UpdatedAt:         po.UpdatedAt,
	}
	if po.Event != nil {
		result.MessageID = po.Event.MessageID
		result.EventType = po.Event.EventType
	}
	return result
}

func attemptFromPO(po *AttemptPO) *domain.WebhookAttempt {
	if po == nil {
		return nil
	}
	return &domain.WebhookAttempt{
		ID:                po.ID,
		DeliveryID:        po.DeliveryID,
		Number:            po.Number,
		Outcome:           po.Outcome,
		HTTPStatus:        cloneInt(po.HTTPStatus),
		FailureCode:       po.FailureCode,
		DurationMS:        po.DurationMS,
		ResponseTruncated: po.ResponseTruncated,
		StartedAt:         po.StartedAt,
		CompletedAt:       po.CompletedAt,
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

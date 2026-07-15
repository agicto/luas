package domain

import (
	"context"
	"encoding/json"
	"time"
)

// WebhookEndpointStatus is the durable administration state of one outbound target.
type WebhookEndpointStatus string

const (
	WebhookEndpointStatusActive   WebhookEndpointStatus = "active"
	WebhookEndpointStatusDisabled WebhookEndpointStatus = "disabled"
)

// WebhookDeliveryStatus is the durable execution state of one event/endpoint pair.
type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending    WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusProcessing WebhookDeliveryStatus = "processing"
	WebhookDeliveryStatusDelivered  WebhookDeliveryStatus = "delivered"
	WebhookDeliveryStatusFailed     WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusCanceled   WebhookDeliveryStatus = "canceled"
)

// IsTerminal reports whether automatic delivery processing has ended.
func (s WebhookDeliveryStatus) IsTerminal() bool {
	return s == WebhookDeliveryStatusDelivered ||
		s == WebhookDeliveryStatusFailed ||
		s == WebhookDeliveryStatusCanceled
}

// WebhookEvent is one trusted, catalog-validated publication command.
type WebhookEvent struct {
	OrganizationID uint
	Source         string
	EventID        string
	Type           string
	OccurredAt     time.Time
	Data           json.RawMessage
}

// WebhookReceipt is the durable identity returned to a trusted publisher.
type WebhookReceipt struct {
	ID             uint64
	MessageID      string
	OrganizationID uint
	Type           string
	DeliveryCount  int
	Created        bool
	OccurredAt     time.Time
	CreatedAt      time.Time
}

// WebhookEndpoint is one organization-owned outbound subscription.
type WebhookEndpoint struct {
	ID                   uint
	OrganizationID       uint
	Name                 string
	URL                  string
	EventTypes           []string
	Status               WebhookEndpointStatus
	DisabledReason       string
	ConsecutiveFailures  uint8
	Version              uint64
	SecretHint           string
	SecretVersion        uint64
	PreviousSecretExpiry *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// WebhookEndpointSecret returns plaintext only at creation or rotation.
type WebhookEndpointSecret struct {
	Endpoint             *WebhookEndpoint
	SigningSecret        string
	PreviousSecretExpiry *time.Time
}

// WebhookDelivery is the privacy-minimized delivery summary exposed to managers and operators.
type WebhookDelivery struct {
	ID                uint64
	OrganizationID    uint
	EndpointID        uint
	MessageID         string
	EventType         string
	Status            WebhookDeliveryStatus
	AttemptCount      uint16
	ReplayCount       uint16
	HTTPStatus        *int
	FailureCode       string
	ResponseTruncated bool
	AvailableAt       time.Time
	DeliveredAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// WebhookAttempt is one minimized network execution record.
type WebhookAttempt struct {
	ID                uint64
	DeliveryID        uint64
	Number            uint16
	Outcome           string
	HTTPStatus        *int
	FailureCode       string
	DurationMS        uint64
	ResponseTruncated bool
	StartedAt         time.Time
	CompletedAt       time.Time
}

// WebhookPruneResult reports terminal records removed by retention maintenance.
type WebhookPruneResult struct {
	Attempts   int64
	Deliveries int64
	Events     int64
	Secrets    int64
}

// WebhookPublisher is the application-facing durable outbox seam.
type WebhookPublisher interface {
	PublishWebhook(context.Context, WebhookEvent) (*WebhookReceipt, error)
}

// WebhookDispatcher processes due delivery leases outside request paths.
type WebhookDispatcher interface {
	DispatchWebhooks(context.Context, int) (int, error)
}

// WebhookTester queues only the fixed starter-owned test event for one endpoint.
type WebhookTester interface {
	PublishWebhookTest(context.Context, uint, uint, uint, string) (*WebhookDelivery, error)
}

// WebhookMaintainer owns bounded replay and retention operations.
type WebhookMaintainer interface {
	ReplayWebhookDelivery(context.Context, uint, uint64, uint) (*WebhookDelivery, error)
	PruneWebhookHistory(context.Context, time.Time) (*WebhookPruneResult, error)
}

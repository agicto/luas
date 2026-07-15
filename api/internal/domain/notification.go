package domain

import (
	"context"
	"time"
)

// NotificationChannel identifies one supported delivery boundary.
type NotificationChannel string

const (
	NotificationChannelInApp NotificationChannel = "in_app"
	NotificationChannelEmail NotificationChannel = "email"
)

// Notification is one immutable user-facing application event plus mutable read state.
type Notification struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	IdempotencyKey string     `json:"-"`
	Kind           string     `json:"kind"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	ActionURL      string     `json:"action_url,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// IsRead reports whether the recipient has acknowledged the in-app notification.
func (n *Notification) IsRead() bool {
	return n != nil && n.ReadAt != nil
}

// NotificationPreference controls future non-required channel selection for one user.
type NotificationPreference struct {
	UserID       uint      `json:"user_id"`
	InAppEnabled bool      `json:"in_app_enabled"`
	EmailEnabled bool      `json:"email_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NotificationPublication is the reviewed internal command accepted by the optional starter.
type NotificationPublication struct {
	UserID           uint
	IdempotencyKey   string
	Kind             string
	Title            string
	Body             string
	ActionURL        string
	Channels         []NotificationChannel
	RequiredChannels []NotificationChannel
}

// NotificationPublisher is the application-facing seam used by downstream business modules.
type NotificationPublisher interface {
	Publish(ctx context.Context, publication NotificationPublication) (*Notification, error)
}

// NotificationDispatcher processes durable channel deliveries outside request paths.
type NotificationDispatcher interface {
	DispatchDue(ctx context.Context, limit int) (int, error)
}

const EventNotificationCreated = "notification.created"

// NotificationCreatedEvent exposes identifiers without copying user-visible content.
type NotificationCreatedEvent struct {
	NotificationID uint
	UserID         uint
	Kind           string
	occurredAt     time.Time
}

// NewNotificationCreatedEvent creates the post-commit notification event.
func NewNotificationCreatedEvent(notification *Notification) NotificationCreatedEvent {
	event := NotificationCreatedEvent{occurredAt: time.Now()}
	if notification != nil {
		event.NotificationID = notification.ID
		event.UserID = notification.UserID
		event.Kind = notification.Kind
	}
	return event
}

func (e NotificationCreatedEvent) EventName() string     { return EventNotificationCreated }
func (e NotificationCreatedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e NotificationCreatedEvent) Data() any {
	return map[string]any{
		"notification_id": e.NotificationID,
		"user_id":         e.UserID,
		"kind":            e.Kind,
	}
}

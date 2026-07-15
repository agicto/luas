package notification

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

type replaceNotificationReadStateRequest struct {
	IsRead *bool `json:"is_read" binding:"required"`
}

type replaceNotificationReadStateThroughRequest struct {
	ThroughID uint `json:"through_id" binding:"required"`
}

type replaceNotificationPreferenceRequest struct {
	InAppEnabled *bool `json:"in_app_enabled" binding:"required"`
	EmailEnabled *bool `json:"email_enabled" binding:"required"`
}

type NotificationResponse struct {
	ID        uint       `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ActionURL string     `json:"action_url,omitempty"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type NotificationStatusResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

type NotificationReadStateResponse struct {
	UpdatedCount int64 `json:"updated_count"`
	UnreadCount  int64 `json:"unread_count"`
}

type NotificationPreferenceResponse struct {
	InAppEnabled bool `json:"in_app_enabled"`
	EmailEnabled bool `json:"email_enabled"`
}

func toNotificationResponse(notification *domain.Notification) *NotificationResponse {
	if notification == nil {
		return nil
	}
	return &NotificationResponse{
		ID:        notification.ID,
		Kind:      notification.Kind,
		Title:     notification.Title,
		Body:      notification.Body,
		ActionURL: notification.ActionURL,
		IsRead:    notification.IsRead(),
		ReadAt:    cloneTime(notification.ReadAt),
		CreatedAt: notification.CreatedAt,
	}
}

func toNotificationPreferenceResponse(
	preference *domain.NotificationPreference,
) *NotificationPreferenceResponse {
	if preference == nil {
		return nil
	}
	return &NotificationPreferenceResponse{
		InAppEnabled: preference.InAppEnabled,
		EmailEnabled: preference.EmailEnabled,
	}
}

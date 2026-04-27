package domain

import (
	"context"
	"time"
)

const (
	AuditActorAnonymous = "anonymous"
	AuditActorUser      = "user"
	AuditActorAPIKey    = "api_key"
	AuditActorSystem    = "system"
)

// AuditLog captures a durable record of a request-side action in the scaffold.
type AuditLog struct {
	ID         uint                        `json:"id"`
	UserID     *uint                       `json:"user_id,omitempty"`
	ActorType  string                      `json:"actor_type"`
	ActorID    *uint                       `json:"actor_id,omitempty"`
	APIKeyID   *uint                       `json:"api_key_id,omitempty"`
	Action     string                      `json:"action"`
	Resource   string                      `json:"resource"`
	TargetType string                      `json:"target_type,omitempty"`
	TargetID   string                      `json:"target_id,omitempty"`
	Result     string                      `json:"result,omitempty"`
	Method     string                      `json:"method"`
	Path       string                      `json:"path"`
	RouteName  string                      `json:"route_name,omitempty"`
	StatusCode int                         `json:"status_code"`
	RequestID  string                      `json:"request_id,omitempty"`
	IPAddress  string                      `json:"ip_address,omitempty"`
	UserAgent  string                      `json:"user_agent,omitempty"`
	Changes    map[string]AuditValueChange `json:"changes,omitempty"`
	Metadata   map[string]any              `json:"metadata,omitempty"`
	CreatedAt  time.Time                   `json:"created_at"`
	UpdatedAt  time.Time                   `json:"updated_at"`
}

const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

// AuditValueChange captures a before/after transition for a domain field.
type AuditValueChange struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

// AuditLogFilter narrows audit log queries for read APIs.
type AuditLogFilter struct {
	Action     string
	Resource   string
	Method     string
	RequestID  string
	StatusCode int
}

// AuditLogRepository defines persistence for audit log records.
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	FindByUserID(ctx context.Context, userID uint, filter AuditLogFilter, page, pageSize int) ([]*AuditLog, int64, error)
}

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

// AuditLog captures a durable record of an audited action in the scaffold.
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
	// UserID, From, and To are used only by the platform-wide query; zero values mean unset.
	UserID *uint
	From   time.Time
	To     time.Time
}

// MaxAuditQueryRange bounds one platform-wide audit query so scans stay on the time index.
const MaxAuditQueryRange = 92 * 24 * time.Hour

// AuditLogQuery is the platform-wide audit read seam for platform operators. Results are newest
// first; From is inclusive and To is exclusive.
type AuditLogQuery interface {
	ListAuditLogsAfter(ctx context.Context, filter AuditLogFilter, after *AuditLogCursor, limit int) ([]*AuditLog, *AuditLogCursor, error)
}

// AuditLogCursor marks the last record of a keyset page. The next page holds records strictly
// older than it: by ID for one user's history, by (CreatedAt, ID) for the platform-wide list.
type AuditLogCursor struct {
	CreatedAt time.Time
	ID        uint
}

// AuditLogRepository defines persistence for audit log records.
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	CreateBatch(ctx context.Context, logs []*AuditLog) error
	// FindByUserIDAfter and FindAllAfter return up to limit records older than after (nil starts at
	// the newest) and the cursor for the next page, which is nil when no older record remains.
	FindByUserIDAfter(ctx context.Context, userID uint, filter AuditLogFilter, after *AuditLogCursor, limit int) ([]*AuditLog, *AuditLogCursor, error)
	FindAllAfter(ctx context.Context, filter AuditLogFilter, after *AuditLogCursor, limit int) ([]*AuditLog, *AuditLogCursor, error)
	PruneBefore(ctx context.Context, before time.Time, batch int) (int64, error)
}

// AuditLogRecorder persists a normalized, redacted audit entry.
type AuditLogRecorder interface {
	Record(ctx context.Context, log *AuditLog) error
}

// AuditLogMaintainer owns bounded retention operations for durable audit records.
type AuditLogMaintainer interface {
	PruneAuditLogs(ctx context.Context, before time.Time, batch int) (int64, error)
}

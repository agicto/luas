package audit

import (
	"time"

	"github.com/zgiai/zgo/internal/domain"
)

// AuditLogListRequest filters audit log queries.
type AuditLogListRequest struct {
	Action     string `form:"action"`
	Resource   string `form:"resource"`
	Method     string `form:"method"`
	RequestID  string `form:"request_id"`
	StatusCode int    `form:"status_code"`
}

// AuditLogResponse is the API response shape for audit entries.
type AuditLogResponse struct {
	ID         uint                               `json:"id"`
	UserID     *uint                              `json:"user_id,omitempty"`
	ActorType  string                             `json:"actor_type"`
	ActorID    *uint                              `json:"actor_id,omitempty"`
	APIKeyID   *uint                              `json:"api_key_id,omitempty"`
	Action     string                             `json:"action"`
	Resource   string                             `json:"resource"`
	TargetType string                             `json:"target_type,omitempty"`
	TargetID   string                             `json:"target_id,omitempty"`
	Result     string                             `json:"result,omitempty"`
	Method     string                             `json:"method"`
	Path       string                             `json:"path"`
	RouteName  string                             `json:"route_name,omitempty"`
	StatusCode int                                `json:"status_code"`
	RequestID  string                             `json:"request_id,omitempty"`
	IPAddress  string                             `json:"ip_address,omitempty"`
	UserAgent  string                             `json:"user_agent,omitempty"`
	Changes    map[string]domain.AuditValueChange `json:"changes,omitempty"`
	Metadata   map[string]any                     `json:"metadata,omitempty"`
	CreatedAt  time.Time                          `json:"created_at"`
	UpdatedAt  time.Time                          `json:"updated_at"`
}

func (r *AuditLogListRequest) toFilter() domain.AuditLogFilter {
	if r == nil {
		return domain.AuditLogFilter{}
	}

	return domain.AuditLogFilter{
		Action:     r.Action,
		Resource:   r.Resource,
		Method:     r.Method,
		RequestID:  r.RequestID,
		StatusCode: r.StatusCode,
	}
}

func toResponse(entry *domain.AuditLog) *AuditLogResponse {
	if entry == nil {
		return nil
	}

	return &AuditLogResponse{
		ID:         entry.ID,
		UserID:     cloneUintPointer(entry.UserID),
		ActorType:  entry.ActorType,
		ActorID:    cloneUintPointer(entry.ActorID),
		APIKeyID:   cloneUintPointer(entry.APIKeyID),
		Action:     entry.Action,
		Resource:   entry.Resource,
		TargetType: entry.TargetType,
		TargetID:   entry.TargetID,
		Result:     entry.Result,
		Method:     entry.Method,
		Path:       entry.Path,
		RouteName:  entry.RouteName,
		StatusCode: entry.StatusCode,
		RequestID:  entry.RequestID,
		IPAddress:  entry.IPAddress,
		UserAgent:  entry.UserAgent,
		Changes:    entry.Changes,
		Metadata:   entry.Metadata,
		CreatedAt:  entry.CreatedAt,
		UpdatedAt:  entry.UpdatedAt,
	}
}

func toResponses(items []*domain.AuditLog) []*AuditLogResponse {
	result := make([]*AuditLogResponse, len(items))
	for i, item := range items {
		result[i] = toResponse(item)
	}
	return result
}

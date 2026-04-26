package audit

import (
	"encoding/json"
	"time"

	"github.com/zgiai/zgo/internal/domain"
)

// AuditLogPO is the persistent object for audit log records.
type AuditLogPO struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	UserID     *uint  `gorm:"index"`
	ActorType  string `gorm:"size:20;not null;index"`
	ActorID    *uint  `gorm:"index"`
	APIKeyID   *uint  `gorm:"index"`
	Action     string `gorm:"size:120;not null;index"`
	Resource   string `gorm:"size:180;not null;index"`
	Method     string `gorm:"size:10;not null;index"`
	Path       string `gorm:"size:255;not null"`
	RouteName  string `gorm:"size:180;index"`
	StatusCode int    `gorm:"not null;index"`
	RequestID  string `gorm:"size:80;index"`
	IPAddress  string `gorm:"size:64"`
	UserAgent  string `gorm:"size:512"`
	Metadata   string `gorm:"type:text"`
}

func (AuditLogPO) TableName() string {
	return "audit_logs"
}

func (po *AuditLogPO) toDomain() *domain.AuditLog {
	if po == nil {
		return nil
	}

	return &domain.AuditLog{
		ID:         po.ID,
		UserID:     cloneUintPointer(po.UserID),
		ActorType:  po.ActorType,
		ActorID:    cloneUintPointer(po.ActorID),
		APIKeyID:   cloneUintPointer(po.APIKeyID),
		Action:     po.Action,
		Resource:   po.Resource,
		Method:     po.Method,
		Path:       po.Path,
		RouteName:  po.RouteName,
		StatusCode: po.StatusCode,
		RequestID:  po.RequestID,
		IPAddress:  po.IPAddress,
		UserAgent:  po.UserAgent,
		Metadata:   decodeMetadata(po.Metadata),
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func newAuditLogPO(entry *domain.AuditLog) *AuditLogPO {
	if entry == nil {
		return nil
	}

	return &AuditLogPO{
		ID:         entry.ID,
		CreatedAt:  entry.CreatedAt,
		UpdatedAt:  entry.UpdatedAt,
		UserID:     cloneUintPointer(entry.UserID),
		ActorType:  entry.ActorType,
		ActorID:    cloneUintPointer(entry.ActorID),
		APIKeyID:   cloneUintPointer(entry.APIKeyID),
		Action:     entry.Action,
		Resource:   entry.Resource,
		Method:     entry.Method,
		Path:       entry.Path,
		RouteName:  entry.RouteName,
		StatusCode: entry.StatusCode,
		RequestID:  entry.RequestID,
		IPAddress:  entry.IPAddress,
		UserAgent:  entry.UserAgent,
		Metadata:   encodeMetadata(entry.Metadata),
	}
}

func cloneUintPointer(value *uint) *uint {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func encodeMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}
	return string(payload)
}

func decodeMetadata(value string) map[string]any {
	if value == "" {
		return nil
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		return nil
	}
	return metadata
}

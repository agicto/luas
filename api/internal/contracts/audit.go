package contracts

import (
	"context"
	"strings"
	"sync"

	"github.com/zgiai/luas/api/internal/domain"
)

type auditChangeCollectorKey struct{}

// AuditChange describes business-level audit metadata captured during request handling.
type AuditChange struct {
	Action     string
	Resource   string
	TargetType string
	TargetID   string
	Result     string
	Changes    map[string]domain.AuditValueChange
	Metadata   map[string]any
}

type auditChangeCollector struct {
	mu     sync.Mutex
	change *AuditChange
}

// WithAuditChangeCollector installs the per-request audit change collector.
func WithAuditChangeCollector(ctx context.Context) context.Context {
	return context.WithValue(ctx, auditChangeCollectorKey{}, &auditChangeCollector{})
}

// RecordAuditChange enriches the current request's audit entry with business-level semantics.
func RecordAuditChange(ctx context.Context, change AuditChange) {
	collector, ok := ctx.Value(auditChangeCollectorKey{}).(*auditChangeCollector)
	if !ok || collector == nil {
		return
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()

	normalized := normalizeAuditChange(change)
	if collector.change == nil {
		collector.change = &normalized
		return
	}

	current := collector.change
	if normalized.Action != "" {
		current.Action = normalized.Action
	}
	if normalized.Resource != "" {
		current.Resource = normalized.Resource
	}
	if normalized.TargetType != "" {
		current.TargetType = normalized.TargetType
	}
	if normalized.TargetID != "" {
		current.TargetID = normalized.TargetID
	}
	if normalized.Result != "" {
		current.Result = normalized.Result
	}
	if len(normalized.Changes) > 0 {
		if current.Changes == nil {
			current.Changes = make(map[string]domain.AuditValueChange, len(normalized.Changes))
		}
		for field, value := range normalized.Changes {
			current.Changes[field] = value
		}
	}
	if len(normalized.Metadata) > 0 {
		if current.Metadata == nil {
			current.Metadata = make(map[string]any, len(normalized.Metadata))
		}
		for key, value := range normalized.Metadata {
			current.Metadata[key] = value
		}
	}
}

// AuditChangeFromContext returns the merged audit change for the current request.
func AuditChangeFromContext(ctx context.Context) *AuditChange {
	collector, ok := ctx.Value(auditChangeCollectorKey{}).(*auditChangeCollector)
	if !ok || collector == nil {
		return nil
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()
	if collector.change == nil {
		return nil
	}

	cloned := *collector.change
	cloned.Changes = cloneAuditChanges(collector.change.Changes)
	cloned.Metadata = cloneAuditMetadata(collector.change.Metadata)
	return &cloned
}

func normalizeAuditChange(change AuditChange) AuditChange {
	change.Action = strings.TrimSpace(change.Action)
	change.Resource = strings.TrimSpace(change.Resource)
	change.TargetType = strings.TrimSpace(change.TargetType)
	change.TargetID = strings.TrimSpace(change.TargetID)
	change.Result = strings.TrimSpace(change.Result)
	change.Changes = cloneAuditChanges(change.Changes)
	change.Metadata = cloneAuditMetadata(change.Metadata)
	return change
}

func cloneAuditChanges(changes map[string]domain.AuditValueChange) map[string]domain.AuditValueChange {
	if len(changes) == 0 {
		return nil
	}
	dup := make(map[string]domain.AuditValueChange, len(changes))
	for key, value := range changes {
		dup[key] = value
	}
	return dup
}

func cloneAuditMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	dup := make(map[string]any, len(metadata))
	for key, value := range metadata {
		dup[key] = value
	}
	return dup
}

package audit

import (
	"context"
	"strings"
	"sync"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/redact"
)

type changeCollectorKey struct{}

// Change describes business-level audit metadata captured during request handling.
type Change struct {
	Action     string
	Resource   string
	TargetType string
	TargetID   string
	Result     string
	Changes    map[string]domain.AuditValueChange
	Metadata   map[string]any
}

type changeCollector struct {
	mu     sync.Mutex
	change *Change
}

func withChangeCollector(ctx context.Context) context.Context {
	return context.WithValue(ctx, changeCollectorKey{}, &changeCollector{})
}

// RecordChange enriches the current request's audit entry with business-level semantics.
func RecordChange(ctx context.Context, change Change) {
	collector, ok := ctx.Value(changeCollectorKey{}).(*changeCollector)
	if !ok || collector == nil {
		return
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()

	normalized := normalizeChange(change)
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

func changeFromContext(ctx context.Context) *Change {
	collector, ok := ctx.Value(changeCollectorKey{}).(*changeCollector)
	if !ok || collector == nil {
		return nil
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()
	if collector.change == nil {
		return nil
	}

	cloned := *collector.change
	cloned.Changes = cloneChanges(collector.change.Changes)
	cloned.Metadata = cloneMetadata(collector.change.Metadata)
	return &cloned
}

func normalizeChange(change Change) Change {
	change.Action = strings.TrimSpace(change.Action)
	change.Resource = strings.TrimSpace(change.Resource)
	change.TargetType = strings.TrimSpace(change.TargetType)
	change.TargetID = strings.TrimSpace(change.TargetID)
	change.Result = strings.TrimSpace(change.Result)
	change.Changes = redactChanges(change.Changes)
	change.Metadata = redact.Map(change.Metadata)
	return change
}

func redactChanges(changes map[string]domain.AuditValueChange) map[string]domain.AuditValueChange {
	if len(changes) == 0 {
		return nil
	}
	result := make(map[string]domain.AuditValueChange, len(changes))
	for field, value := range changes {
		if redact.IsSensitiveKey(field) {
			result[field] = domain.AuditValueChange{
				Before: redactAuditValue(value.Before),
				After:  redactAuditValue(value.After),
			}
			continue
		}
		result[field] = domain.AuditValueChange{
			Before: redact.Value(value.Before),
			After:  redact.Value(value.After),
		}
	}
	return result
}

func redactAuditValue(value any) any {
	if value == nil {
		return nil
	}
	return redact.Placeholder
}

func cloneChanges(changes map[string]domain.AuditValueChange) map[string]domain.AuditValueChange {
	if len(changes) == 0 {
		return nil
	}
	dup := make(map[string]domain.AuditValueChange, len(changes))
	for key, value := range changes {
		dup[key] = value
	}
	return dup
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	dup := make(map[string]any, len(metadata))
	for key, value := range metadata {
		dup[key] = value
	}
	return dup
}

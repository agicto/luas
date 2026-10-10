package audit

import (
	"context"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/redact"
)

// Service defines audit logging operations.
type Service interface {
	Record(ctx context.Context, entry *domain.AuditLog) error
	RecordRequest(ctx context.Context, entry *domain.AuditLog) error
	Shutdown(ctx context.Context) error
	ListForUser(ctx context.Context, userID uint, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error)
	ListForUserAfter(ctx context.Context, userID uint, filter domain.AuditLogFilter, after *domain.AuditLogCursor, limit int) ([]*domain.AuditLog, *domain.AuditLogCursor, error)
	PruneAuditLogs(ctx context.Context, before time.Time, batch int) (int64, error)
}

type service struct {
	repo domain.AuditLogRepository
	// writer, when set, takes request audit records off the response path (AUDIT_WRITE_MODE=async).
	writer *asyncWriter
}

var (
	_ Service                   = (*service)(nil)
	_ domain.AuditLogRecorder   = (*service)(nil)
	_ domain.AuditLogMaintainer = (*service)(nil)
)

const maxAuditPruneBatch = 10_000

// NewService creates a new audit service.
func NewService(repo domain.AuditLogRepository) *service {
	return &service{repo: repo}
}

// ProvideService builds the service with the configured request audit write mode.
func ProvideService(repo domain.AuditLogRepository, cfg *config.Config) *service {
	svc := NewService(repo)
	if cfg == nil || cfg.Audit.WriteMode != config.AuditWriteModeSync {
		svc.writer = newAsyncWriter(repo)
	}
	return svc
}

// Record validates, normalizes, and stores one audit record before returning.
func (s *service) Record(ctx context.Context, entry *domain.AuditLog) error {
	if err := s.prepare(ctx, entry); err != nil {
		return err
	}
	return s.repo.Create(ctx, entry)
}

// RecordRequest stores the audit record of a completed HTTP request. Normalization, which reads
// request-scoped changes, finishes before it returns; in async mode the write happens afterwards.
func (s *service) RecordRequest(ctx context.Context, entry *domain.AuditLog) error {
	if err := s.prepare(ctx, entry); err != nil {
		return err
	}
	if s.writer == nil {
		return s.repo.Create(ctx, entry)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC() // the request time, not the later flush time
	}
	return s.writer.Write(ctx, entry)
}

// Shutdown writes every queued request audit record.
func (s *service) Shutdown(ctx context.Context) error {
	if s.writer == nil {
		return nil
	}
	return s.writer.Shutdown(ctx)
}

func (s *service) prepare(ctx context.Context, entry *domain.AuditLog) error {
	if entry == nil {
		return domain.ErrInvalidInput
	}

	mergeBusinessChange(ctx, entry)

	entry.Method = strings.ToUpper(strings.TrimSpace(entry.Method))
	entry.Path = strings.TrimSpace(entry.Path)
	entry.RouteName = strings.TrimSpace(entry.RouteName)
	entry.RequestID = strings.TrimSpace(entry.RequestID)
	entry.IPAddress = strings.TrimSpace(entry.IPAddress)
	entry.UserAgent = strings.TrimSpace(entry.UserAgent)
	entry.TargetType = strings.TrimSpace(entry.TargetType)
	entry.TargetID = strings.TrimSpace(entry.TargetID)
	entry.Result = strings.TrimSpace(entry.Result)
	entry.Changes = redactChanges(entry.Changes)
	entry.Metadata = redact.Map(entry.Metadata)

	if entry.Method == "" || entry.Path == "" {
		return domain.ErrInvalidInput
	}

	entry.ActorType = normalizeActorType(entry.ActorType, entry.UserID, entry.APIKeyID)
	if entry.ActorID == nil {
		switch entry.ActorType {
		case domain.AuditActorUser:
			entry.ActorID = cloneUintPointer(entry.UserID)
		case domain.AuditActorAPIKey:
			entry.ActorID = cloneUintPointer(entry.APIKeyID)
		}
	}

	if entry.Resource == "" || entry.Action == "" {
		resource, action := deriveResourceAction(entry.RouteName, entry.Method, entry.Path)
		if entry.Resource == "" {
			entry.Resource = resource
		}
		if entry.Action == "" {
			entry.Action = action
		}
	}
	if entry.Result == "" {
		if entry.StatusCode >= 400 {
			entry.Result = domain.AuditResultFailure
		} else {
			entry.Result = domain.AuditResultSuccess
		}
	}
	return nil
}

// PruneAuditLogs removes one bounded batch strictly older than the reviewed cutoff.
func (s *service) PruneAuditLogs(ctx context.Context, before time.Time, batch int) (int64, error) {
	if before.IsZero() || !before.Before(time.Now().UTC()) || batch < 1 || batch > maxAuditPruneBatch {
		return 0, domain.ErrInvalidInput
	}
	return s.repo.PruneBefore(ctx, before.UTC(), batch)
}

func (s *service) ListForUser(ctx context.Context, userID uint, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	if userID == 0 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.repo.FindByUserID(ctx, userID, filter, page, pageSize)
}

// ListAuditLogs returns platform-wide audit logs for platform operators. The time range is required
// to be ordered and at most domain.MaxAuditQueryRange; a missing range covers the last 30 days.
func (s *service) ListAuditLogs(ctx context.Context, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	filter, err := platformAuditRange(filter)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.FindAll(ctx, filter, page, pageSize)
}

// ListForUserAfter returns one keyset page of the user's own history, newest first.
func (s *service) ListForUserAfter(
	ctx context.Context,
	userID uint,
	filter domain.AuditLogFilter,
	after *domain.AuditLogCursor,
	limit int,
) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	if userID == 0 || limit < 1 || limit > 100 {
		return nil, nil, domain.ErrInvalidInput
	}
	return s.repo.FindByUserIDAfter(ctx, userID, filter, after, limit)
}

// ListAuditLogsAfter returns one keyset page of platform-wide history under the same range rules as
// ListAuditLogs, without counting the matching records.
func (s *service) ListAuditLogsAfter(
	ctx context.Context,
	filter domain.AuditLogFilter,
	after *domain.AuditLogCursor,
	limit int,
) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	if limit < 1 || limit > 100 {
		return nil, nil, domain.ErrInvalidInput
	}
	filter, err := platformAuditRange(filter)
	if err != nil {
		return nil, nil, err
	}
	return s.repo.FindAllAfter(ctx, filter, after, limit)
}

// platformAuditRange defaults a missing range to the last 30 days and rejects an unordered range or
// one longer than domain.MaxAuditQueryRange.
func platformAuditRange(filter domain.AuditLogFilter) (domain.AuditLogFilter, error) {
	if filter.To.IsZero() {
		filter.To = time.Now().UTC()
	}
	if filter.From.IsZero() {
		filter.From = filter.To.Add(-30 * 24 * time.Hour)
	}
	if !filter.From.Before(filter.To) || filter.To.Sub(filter.From) > domain.MaxAuditQueryRange {
		return filter, domain.ErrInvalidInput
	}
	return filter, nil
}

func normalizeActorType(actorType string, userID, apiKeyID *uint) string {
	switch strings.TrimSpace(actorType) {
	case domain.AuditActorUser, domain.AuditActorAPIKey, domain.AuditActorAnonymous, domain.AuditActorSystem:
		return actorType
	}

	if apiKeyID != nil {
		return domain.AuditActorAPIKey
	}
	if userID != nil {
		return domain.AuditActorUser
	}
	return domain.AuditActorAnonymous
}

func deriveResourceAction(routeName, method, path string) (string, string) {
	routeName = strings.TrimSpace(routeName)
	if routeName != "" {
		parts := strings.Split(routeName, ".")
		if len(parts) > 1 {
			return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
		}
		return routeName, actionFromMethod(method)
	}

	return resourceFromPath(path), actionFromMethod(method)
}

func resourceFromPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "v1/")
	if path == "" {
		return "root"
	}

	parts := strings.Split(path, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, ":") {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) == 0 {
		return "root"
	}
	return strings.Join(filtered, ".")
}

func actionFromMethod(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	case "GET":
		return "read"
	default:
		return "request"
	}
}

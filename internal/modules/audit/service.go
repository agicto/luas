package audit

import (
	"context"
	"strings"

	"github.com/zgiai/zgo/internal/domain"
)

// Service defines audit logging operations.
type Service interface {
	Record(ctx context.Context, entry *domain.AuditLog) error
	ListForUser(ctx context.Context, userID uint, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error)
}

type service struct {
	repo domain.AuditLogRepository
}

var _ Service = (*service)(nil)

// NewService creates a new audit service.
func NewService(repo domain.AuditLogRepository) *service {
	return &service{repo: repo}
}

func (s *service) Record(ctx context.Context, entry *domain.AuditLog) error {
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

	return s.repo.Create(ctx, entry)
}

func (s *service) ListForUser(ctx context.Context, userID uint, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	if userID == 0 {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.repo.FindByUserID(ctx, userID, filter, page, pageSize)
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

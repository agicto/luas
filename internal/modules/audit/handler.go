package audit

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/domain"
	infraMiddleware "github.com/zgiai/zgo/internal/infra/middleware"
	"github.com/zgiai/zgo/internal/infra/router"
	httphandler "github.com/zgiai/zgo/pkg/handler"
	"github.com/zgiai/zgo/pkg/logger"
	"github.com/zgiai/zgo/pkg/pagination"
	"github.com/zgiai/zgo/pkg/response"
)

// Handler exposes the audit log routes and global audit middleware.
type Handler struct {
	service Service
}

var (
	_ contracts.Module           = (*Handler)(nil)
	_ contracts.RouteModule      = (*Handler)(nil)
	_ contracts.MiddlewareModule = (*Handler)(nil)
)

// NewHandler creates a new audit handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the module name.
func (h *Handler) Name() string {
	return "audit"
}

// RegisterMiddleware registers the global audit alias.
func (h *Handler) RegisterMiddleware(r *router.Router) {
	r.AliasMiddleware("audit", h.AuditMiddleware())
}

// List returns the current user's audit history.
func (h *Handler) List(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}

	var req AuditLogListRequest
	if !httphandler.BindQuery(c, &req) {
		return
	}

	page := pagination.FromContext(c)
	items, total, err := h.service.ListForUser(c.Request.Context(), userID, req.toFilter(), page.GetPage(), page.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list audit logs", err)
		return
	}

	paginator := pagination.NewPaginator(toResponses(items), total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// AuditMiddleware records mutating API requests without blocking the primary request path.
func (h *Handler) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if !shouldAudit(c) {
			return
		}

		entry := buildAuditEntry(c)
		if entry == nil {
			return
		}

		if err := h.service.Record(c.Request.Context(), entry); err != nil {
			logger.Channel("audit").Warning("failed to write audit log", map[string]any{
				"error":      err.Error(),
				"method":     entry.Method,
				"path":       entry.Path,
				"route_name": entry.RouteName,
			})
		}
	}
}

func shouldAudit(c *gin.Context) bool {
	switch strings.ToUpper(c.Request.Method) {
	case "GET", "HEAD", "OPTIONS":
		return false
	default:
		return true
	}
}

func buildAuditEntry(c *gin.Context) *domain.AuditLog {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	entry := &domain.AuditLog{
		Method:     c.Request.Method,
		Path:       path,
		StatusCode: c.Writer.Status(),
		RequestID:  infraMiddleware.GetRequestID(c),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		RouteName:  c.GetString("route_name"),
	}

	if userID, ok := getUintFromContext(c, "userID"); ok {
		entry.UserID = &userID
	}
	if apiKeyID, ok := getUintFromContext(c, "apiKeyID"); ok {
		entry.APIKeyID = &apiKeyID
	}

	return entry
}

func getUintFromContext(c *gin.Context, key string) (uint, bool) {
	value, ok := c.Get(key)
	if !ok {
		return 0, false
	}

	switch v := value.(type) {
	case uint:
		return v, true
	case int:
		return uint(v), true
	case int64:
		return uint(v), true
	case uint64:
		return uint(v), true
	case float64:
		return uint(v), true
	default:
		return 0, false
	}
}

package operator

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/audit"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

type listAuditLogsQuery struct {
	UserID     uint   `form:"user_id" binding:"omitempty,min=1"`
	Action     string `form:"action" binding:"omitempty,max=120"`
	Resource   string `form:"resource" binding:"omitempty,max=180"`
	Method     string `form:"method" binding:"omitempty,max=10"`
	RequestID  string `form:"request_id" binding:"omitempty,max=80"`
	StatusCode int    `form:"status_code" binding:"omitempty,gte=100,lte=599"`
	From       string `form:"from" binding:"omitempty,max=40"`
	To         string `form:"to" binding:"omitempty,max=40"`
}

func (q listAuditLogsQuery) filter() (domain.AuditLogFilter, bool) {
	filter := domain.AuditLogFilter{
		Action:     q.Action,
		Resource:   q.Resource,
		Method:     q.Method,
		RequestID:  q.RequestID,
		StatusCode: q.StatusCode,
	}
	if q.UserID > 0 {
		userID := q.UserID
		filter.UserID = &userID
	}
	var ok bool
	if filter.From, ok = parseInstant(q.From); !ok {
		return filter, false
	}
	if filter.To, ok = parseInstant(q.To); !ok {
		return filter, false
	}
	return filter, true
}

func parseInstant(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return parsed.UTC(), err == nil
}

// ListAuditLogs serves GET /v1/operator/audit-logs: platform-wide audit history, newest first.
func (h *Handler) ListAuditLogs(c *gin.Context) {
	var query listAuditLogsQuery
	if !handler.BindQuery(c, &query) {
		return
	}
	filter, ok := query.filter()
	if !ok {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "from and to must be RFC 3339 timestamps")
		return
	}
	if h.service.audit == nil {
		writeError(c, "List audit logs", domain.ErrServiceUnavailable)
		return
	}
	after, err := audit.CursorRequest(c.Request.URL.Query())
	if err != nil {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, err.Error())
		return
	}
	perPage := pagination.FromContext(c).GetPerPage()
	items, next, err := h.service.audit.ListAuditLogsAfter(c.Request.Context(), filter, after, perPage)
	if err != nil {
		writeError(c, "List audit logs", err)
		return
	}
	response.SuccessCursorPage(c, audit.Responses(items), perPage, audit.EncodeCursor(next))
}

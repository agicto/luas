package trend

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/contracts"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler exposes the trend sourcing and scoring routes.
type Handler struct {
	service *Service
}

var (
	_ contracts.Module      = (*Handler)(nil)
	_ contracts.RouteModule = (*Handler)(nil)
)

// NewHandler creates a trend HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Name returns the module name.
func (h *Handler) Name() string {
	return "trend"
}

// List returns sourced and scored trend items.
func (h *Handler) List(c *gin.Context) {
	var req TrendListRequest
	if !httphandler.BindQuery(c, &req) {
		return
	}

	page := pagination.FromContext(c)
	items, total, err := h.service.ListTrends(
		c.Request.Context(),
		req.toFilter(page.GetPage(), page.GetPerPage()),
	)
	if err != nil {
		response.HandleError(c, "Failed to list trends", err)
		return
	}

	paginator := pagination.NewPaginator(toTrendResponses(items), total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path).WithQuery(c.Request.URL.Query())
	response.Success(c, paginator)
}

// Stats returns aggregate counters and source polling state for the console.
func (h *Handler) Stats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load trend stats", err)
		return
	}

	source, err := h.service.GetDailyDevSourceStatus(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load trend source status", err)
		return
	}

	response.Success(c, TrendListSummaryResponse{
		Stats:  toTrendStatsResponse(stats),
		Source: toSourceStatusResponse(source),
	})
}

// CreateSyncRun triggers a synchronous daily.dev sync run.
func (h *Handler) CreateSyncRun(c *gin.Context) {
	result, err := h.service.SyncDailyDevHighlights(c.Request.Context(), "")
	if err != nil {
		response.HandleError(c, "Failed to sync daily.dev highlights", err)
		return
	}

	response.Created(c, toSyncRunResponse(result))
}

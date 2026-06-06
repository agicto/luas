package team

import (
	"github.com/gin-gonic/gin"
	"github.com/zgiai/luas/api/internal/contracts"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler handles HTTP requests for Team.
type Handler struct {
	service Service
}

var (
	_ contracts.Module      = (*Handler)(nil)
	_ contracts.RouteModule = (*Handler)(nil)
)

// NewHandler creates a new handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the module name.
func (h *Handler) Name() string {
	return "team"
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}

	req := pagination.FromContext(c)

	items, total, err := h.service.ListForUser(c.Request.Context(), userID, req.GetPage(), req.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list teams", err)
		return
	}

	paginator := pagination.NewPaginator(toTeamResponses(items), total, req.GetPage(), req.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}

	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	item, err := h.service.GetForUser(c.Request.Context(), userID, id)
	if err != nil {
		response.HandleError(c, "Failed to get team", err)
		return
	}

	response.Success(c, toTeamResponse(item))
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}

	var req CreateTeamRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	item, err := h.service.CreateForUser(c.Request.Context(), userID, &req)
	if err != nil {
		response.HandleError(c, "Failed to create team", err)
		return
	}

	response.Created(c, toTeamResponse(item))
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}

	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	var req UpdateTeamRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	item, err := h.service.UpdateForUser(c.Request.Context(), userID, id, &req)
	if err != nil {
		response.HandleError(c, "Failed to update team", err)
		return
	}

	response.Success(c, toTeamResponse(item))
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}

	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteForUser(c.Request.Context(), userID, id); err != nil {
		response.HandleError(c, "Failed to delete team", err)
		return
	}

	response.NoContent(c)
}

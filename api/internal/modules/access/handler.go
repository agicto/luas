package access

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/contracts"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler handles HTTP requests for access control.
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
	return "access"
}

func (h *Handler) Permissions(c *gin.Context) {
	response.Success(c, toPermissionResponses(h.service.PermissionCatalog(c.Request.Context())))
}

func (h *Handler) ListRoles(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	teamID, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	req := pagination.FromContext(c)

	items, total, err := h.service.ListRoles(c.Request.Context(), userID, teamID, req.GetPage(), req.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list roles", err)
		return
	}

	paginator := pagination.NewPaginator(toAccessRoleResponses(items), total, req.GetPage(), req.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

func (h *Handler) GetRole(c *gin.Context) {
	userID, teamID, roleID, ok := parseRoleScope(c)
	if !ok {
		return
	}

	item, err := h.service.GetRole(c.Request.Context(), userID, teamID, roleID)
	if err != nil {
		response.HandleError(c, "Failed to get role", err)
		return
	}

	response.Success(c, toAccessRoleResponse(item))
}

func (h *Handler) CreateRole(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	teamID, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	var req CreateRoleRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	item, err := h.service.CreateRole(c.Request.Context(), userID, teamID, &req)
	if err != nil {
		response.HandleError(c, "Failed to create role", err)
		return
	}

	response.Created(c, toAccessRoleResponse(item))
}

func (h *Handler) UpdateRole(c *gin.Context) {
	userID, teamID, roleID, ok := parseRoleScope(c)
	if !ok {
		return
	}

	var req UpdateRoleRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	item, err := h.service.UpdateRole(c.Request.Context(), userID, teamID, roleID, &req)
	if err != nil {
		response.HandleError(c, "Failed to update role", err)
		return
	}

	response.Success(c, toAccessRoleResponse(item))
}

func (h *Handler) DeleteRole(c *gin.Context) {
	userID, teamID, roleID, ok := parseRoleScope(c)
	if !ok {
		return
	}

	if err := h.service.DeleteRole(c.Request.Context(), userID, teamID, roleID); err != nil {
		response.HandleError(c, "Failed to delete role", err)
		return
	}

	response.NoContent(c)
}

func parseRoleScope(c *gin.Context) (uint, uint, uint, bool) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return 0, 0, 0, false
	}
	teamID, ok := httphandler.ParseID(c, "id")
	if !ok {
		return 0, 0, 0, false
	}
	roleID, ok := httphandler.ParseID(c, "role_id")
	if !ok {
		return 0, 0, 0, false
	}
	return userID, teamID, roleID, true
}

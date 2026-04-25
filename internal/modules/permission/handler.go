package permission

import (
	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/pkg/handler"
	"github.com/zgiai/zgo/pkg/response"
)

// Handler handles permission-related HTTP requests and exposes route capability.
type Handler struct {
	service Service
}

var (
	_ contracts.Module      = (*Handler)(nil)
	_ contracts.RouteModule = (*Handler)(nil)
)

// NewHandler creates a new permission handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the module name.
func (h *Handler) Name() string {
	return "permission"
}

// CreateRole creates a new role.
func (h *Handler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	role, err := h.service.CreateRole(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Failed to create role", err)
		return
	}

	response.Created(c, toRoleResponse(role))
}

// GetRole gets a role by ID.
func (h *Handler) GetRole(c *gin.Context) {
	id, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	role, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, "Role not found", err)
		return
	}

	response.Success(c, toRoleResponse(role))
}

// ListRoles lists all roles.
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to list roles", err)
		return
	}

	response.Success(c, toRoleResponses(roles))
}

// UpdateRole updates a role.
func (h *Handler) UpdateRole(c *gin.Context) {
	id, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	var req UpdateRoleRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	role, err := h.service.UpdateRole(c.Request.Context(), id, &req)
	if err != nil {
		response.HandleError(c, "Failed to update role", err)
		return
	}

	response.Success(c, toRoleResponse(role))
}

// DeleteRole deletes a role.
func (h *Handler) DeleteRole(c *gin.Context) {
	id, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteRole(c.Request.Context(), id); err != nil {
		response.HandleError(c, "Failed to delete role", err)
		return
	}

	response.NoContent(c)
}

// AssignRole assigns a role to a user.
func (h *Handler) AssignRole(c *gin.Context) {
	var req AssignRoleRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	if err := h.service.AssignRoleToUser(c.Request.Context(), req.UserID, req.RoleID); err != nil {
		response.HandleError(c, "Failed to assign role", err)
		return
	}

	response.NoContent(c)
}

// RemoveRole removes a role from a user.
func (h *Handler) RemoveRole(c *gin.Context) {
	var req AssignRoleRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	if err := h.service.RemoveRoleFromUser(c.Request.Context(), req.UserID, req.RoleID); err != nil {
		response.HandleError(c, "Failed to remove role", err)
		return
	}

	response.NoContent(c)
}

// GetUserRoles gets all roles for a user.
func (h *Handler) GetUserRoles(c *gin.Context) {
	userID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	roles, err := h.service.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, "Failed to get user roles", err)
		return
	}

	response.Success(c, newUserRolesResponse(userID, roles))
}

// ListPermissions lists all permissions.
func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to list permissions", err)
		return
	}

	response.Success(c, toPermissionResponses(permissions))
}

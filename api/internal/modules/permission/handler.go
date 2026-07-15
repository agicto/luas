package permission

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler owns permission starter HTTP routes.
type Handler struct {
	service Service
	guard   *Guard
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
)

// NewHandler creates the permission HTTP handler.
func NewHandler(service *service, guard *Guard) *Handler {
	return &Handler{service: service, guard: guard}
}

// Name returns the starter module name.
func (h *Handler) Name() string { return "permission" }

// GetEffective returns the current persisted permission context.
func (h *Handler) GetEffective(c *gin.Context) {
	organization, ok := permissionOrganizationContext(c)
	if !ok {
		return
	}
	effective, err := h.service.Effective(c.Request.Context(), organization)
	if err != nil {
		response.HandleError(c, "Failed to resolve permission context", err)
		return
	}
	response.Success(c, effective)
}

// ListPermissions returns the immutable code-owned catalog.
func (h *Handler) ListPermissions(c *gin.Context) {
	organization, ok := permissionOrganizationContext(c)
	if !ok {
		return
	}
	permissions, err := h.service.Catalog(c.Request.Context(), organization)
	if err != nil {
		response.HandleError(c, "Failed to list permissions", err)
		return
	}
	response.Success(c, &PermissionCatalogResponse{Permissions: permissions})
}

// ListRoles returns paginated access roles for the active organization.
func (h *Handler) ListRoles(c *gin.Context) {
	organization, ok := permissionOrganizationContext(c)
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	roles, total, err := h.service.ListRoles(c.Request.Context(), organization, page.GetPage(), page.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list access roles", err)
		return
	}
	paginator := pagination.NewPaginator(roles, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// GetRole returns one active-organization access role.
func (h *Handler) GetRole(c *gin.Context) {
	organization, roleID, ok := permissionContextAndID(c, "id")
	if !ok {
		return
	}
	role, err := h.service.GetRole(c.Request.Context(), organization, roleID)
	if err != nil {
		response.HandleError(c, "Access role not found", err)
		return
	}
	response.Success(c, role)
}

// CreateRole creates one access role.
func (h *Handler) CreateRole(c *gin.Context) {
	organization, ok := permissionOrganizationContext(c)
	if !ok {
		return
	}
	var request CreateAccessRoleRequest
	if !handler.BindJSON(c, &request) {
		return
	}
	if errors := request.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}
	role, err := h.service.CreateRole(c.Request.Context(), organization, &request)
	if err != nil {
		response.HandleError(c, "Failed to create access role", err)
		return
	}
	response.Created(c, role)
}

// UpdateRole changes one access role without changing its slug.
func (h *Handler) UpdateRole(c *gin.Context) {
	organization, roleID, ok := permissionContextAndID(c, "id")
	if !ok {
		return
	}
	var request UpdateAccessRoleRequest
	if !handler.BindJSON(c, &request) {
		return
	}
	if errors := request.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}
	role, err := h.service.UpdateRole(c.Request.Context(), organization, roleID, &request)
	if err != nil {
		response.HandleError(c, "Failed to update access role", err)
		return
	}
	response.Success(c, role)
}

// DeleteRole removes a role and its assignments.
func (h *Handler) DeleteRole(c *gin.Context) {
	organization, roleID, ok := permissionContextAndID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteRole(c.Request.Context(), organization, roleID); err != nil {
		response.HandleError(c, "Failed to delete access role", err)
		return
	}
	response.NoContent(c)
}

// GetMemberRoles returns one member's access-role IDs.
func (h *Handler) GetMemberRoles(c *gin.Context) {
	organization, memberID, ok := permissionContextAndID(c, "member_id")
	if !ok {
		return
	}
	assignment, err := h.service.MemberRoles(c.Request.Context(), organization, memberID)
	if err != nil {
		response.HandleError(c, "Failed to read member access roles", err)
		return
	}
	response.Success(c, assignment)
}

// ReplaceMemberRoles atomically replaces one member's access-role IDs.
func (h *Handler) ReplaceMemberRoles(c *gin.Context) {
	organization, memberID, ok := permissionContextAndID(c, "member_id")
	if !ok {
		return
	}
	var request ReplaceMemberAccessRolesRequest
	if !handler.BindJSON(c, &request) {
		return
	}
	if errors := request.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}
	assignment, err := h.service.ReplaceMemberRoles(c.Request.Context(), organization, memberID, &request)
	if err != nil {
		response.HandleError(c, "Failed to replace member access roles", err)
		return
	}
	response.Success(c, assignment)
}

func permissionOrganizationContext(c *gin.Context) (domain.OrganizationContext, bool) {
	organization, ok := domain.OrganizationContextFromContext(c.Request.Context())
	if !ok {
		response.HandleError(c, "Organization context required", domain.ErrOrganizationContextRequired)
		return domain.OrganizationContext{}, false
	}
	return organization, true
}

func permissionContextAndID(c *gin.Context, parameter string) (domain.OrganizationContext, uint, bool) {
	organization, ok := permissionOrganizationContext(c)
	if !ok {
		return domain.OrganizationContext{}, 0, false
	}
	id, ok := handler.ParseID(c, parameter)
	if !ok || id == 0 {
		return domain.OrganizationContext{}, 0, false
	}
	return organization, id, true
}

package organization

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler handles organization HTTP requests and active-starter hooks.
type Handler struct {
	service        Service
	deletionGuard  user.AccountDeletionGuard
	deletionPolicy *user.AccountDeletionPolicy
}

var (
	_ assembly.Module           = (*Handler)(nil)
	_ assembly.RouteModule      = (*Handler)(nil)
	_ assembly.ActivationModule = (*Handler)(nil)
)

// NewHandler creates an organization handler.
func NewHandler(service *service, deletionPolicy *user.AccountDeletionPolicy) *Handler {
	return &Handler{
		service:        service,
		deletionGuard:  service,
		deletionPolicy: deletionPolicy,
	}
}

// Name returns the starter module name.
func (h *Handler) Name() string {
	return "organization"
}

// Activate installs ownership protection only when this optional starter is selected.
func (h *Handler) Activate() error {
	return h.deletionPolicy.Register(h.deletionGuard)
}

// Create creates an organization owned by the authenticated user.
func (h *Handler) Create(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	var req CreateOrganizationRequest
	if !handler.BindJSON(c, &req) {
		return
	}
	if errors := req.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}

	membership, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		response.HandleError(c, "Failed to create organization", err)
		return
	}
	response.Created(c, toResponse(membership))
}

// List returns organizations visible through the caller's memberships.
func (h *Handler) List(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	page := pagination.FromContext(c)
	memberships, total, err := h.service.List(c.Request.Context(), userID, page.GetPage(), page.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list organizations", err)
		return
	}

	items := make([]*OrganizationResponse, len(memberships))
	for i, membership := range memberships {
		items[i] = toResponse(membership)
	}
	paginator := pagination.NewPaginator(items, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// Get returns one membership-scoped organization.
func (h *Handler) Get(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	membership, err := h.service.Get(c.Request.Context(), userID, organizationID)
	if err != nil {
		response.HandleError(c, "Organization not found", err)
		return
	}
	response.Success(c, toResponse(membership))
}

// Update changes mutable organization settings.
func (h *Handler) Update(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}
	organizationID, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	var req UpdateOrganizationRequest
	if !handler.BindJSON(c, &req) {
		return
	}
	if errors := req.validationErrors(); len(errors) > 0 {
		response.ValidationFailed(c, errors)
		return
	}

	membership, err := h.service.Update(c.Request.Context(), userID, organizationID, &req)
	if err != nil {
		response.HandleError(c, "Failed to update organization", err)
		return
	}
	response.Success(c, toResponse(membership))
}

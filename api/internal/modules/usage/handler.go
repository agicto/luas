package usage

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler owns private current-user and active-organization usage summaries.
type Handler struct {
	service        Service
	cleaner        user.AccountDeletionCleaner
	deletionPolicy *user.AccountDeletionPolicy
}

var (
	_ assembly.Module           = (*Handler)(nil)
	_ assembly.RouteModule      = (*Handler)(nil)
	_ assembly.ActivationModule = (*Handler)(nil)
)

// NewHandler creates the optional usage HTTP boundary.
func NewHandler(service *service, deletionPolicy *user.AccountDeletionPolicy) *Handler {
	return &Handler{
		service:        service,
		cleaner:        service,
		deletionPolicy: deletionPolicy,
	}
}

func (h *Handler) Name() string { return "usage" }

// Activate installs user-usage cleanup only when the starter is selected.
func (h *Handler) Activate() error {
	return h.deletionPolicy.RegisterCleaner(h.cleaner)
}

// UserList returns the current user's finite current-period usage catalog.
func (h *Handler) UserList(c *gin.Context) {
	setPrivateUsageResponse(c, false)
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	h.list(c, domain.UsageTarget{Scope: domain.UsageScopeUser, SubjectID: userID})
}

// OrganizationList returns owner/admin usage for the verified active organization.
func (h *Handler) OrganizationList(c *gin.Context) {
	setPrivateUsageResponse(c, true)
	organization, ok := domain.OrganizationContextFromContext(c.Request.Context())
	if !ok {
		response.HandleError(c, "Organization context required", domain.ErrOrganizationContextRequired)
		return
	}
	if !organization.Role.CanManageOrganization() {
		response.HandleError(c, "Organization usage access forbidden", domain.ErrPermissionDenied)
		return
	}
	h.list(c, domain.UsageTarget{
		Scope:     domain.UsageScopeOrganization,
		SubjectID: organization.OrganizationID,
	})
}

// luas:bounded-list max=64 reason=finite-code-owned-catalog
func (h *Handler) list(c *gin.Context, target domain.UsageTarget) {
	values, err := h.service.ListUsage(c.Request.Context(), target)
	if err != nil {
		response.HandleError(c, "Failed to load usage", err)
		return
	}
	response.Success(c, toUsageSummaryResponses(values))
}

func setPrivateUsageResponse(c *gin.Context, organization bool) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization")
	if organization {
		c.Header("Vary", "Authorization, Organization-Id")
	}
}

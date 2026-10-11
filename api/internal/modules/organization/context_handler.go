package organization

import (
	"github.com/gin-gonic/gin"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/pkg/response"
)

// GetActiveContext returns the membership resolved by active-context middleware.
func (h *Handler) GetActiveContext(c *gin.Context) {
	resolved, ok := domain.OrganizationContextFromContext(c.Request.Context())
	if !ok {
		response.HandleError(c, "Failed to resolve organization context", domain.ErrServiceUnavailable)
		return
	}
	response.Success(c, toContextResponse(resolved))
}

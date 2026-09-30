package usage

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrUsageMetricNotFound, http.StatusNotFound, domain.CodeUsageMetricNotFound)
	mapper.Register(domain.ErrUsageIdempotencyConflict, http.StatusConflict, domain.CodeUsageIdempotencyConflict)
	mapper.Register(domain.ErrUsageQuotaVersionConflict, http.StatusPreconditionFailed, domain.CodeUsageQuotaVersionConflict)
	mapper.Register(domain.ErrUsageInvalidEvent, http.StatusUnprocessableEntity, domain.CodeUsageInvalidEvent)
	mapper.Register(domain.ErrUsageEventOutsideWindow, http.StatusUnprocessableEntity, domain.CodeUsageEventOutsideWindow)
	mapper.Register(domain.ErrUsagePreconditionRequired, http.StatusPreconditionRequired, domain.CodeUsagePreconditionRequired)
	mapper.Register(domain.ErrUsageQuotaExceeded, http.StatusTooManyRequests, domain.CodeUsageQuotaExceeded)
}

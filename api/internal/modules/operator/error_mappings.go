package operator

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors for callers outside its own handlers.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrOperatorForbidden, http.StatusForbidden, domain.CodeOperatorForbidden)
	mapper.Register(domain.ErrOperatorTargetProtected, http.StatusConflict, domain.CodeOperatorTargetProtected)
}

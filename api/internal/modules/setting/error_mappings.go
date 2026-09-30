package setting

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrSettingNotFound, http.StatusNotFound, domain.CodeSettingNotFound)
	mapper.Register(domain.ErrSettingVersionConflict, http.StatusPreconditionFailed, domain.CodeSettingVersionConflict)
	mapper.Register(domain.ErrSettingInvalidValue, http.StatusUnprocessableEntity, domain.CodeSettingInvalidValue)
	mapper.Register(domain.ErrSettingPreconditionRequired, http.StatusPreconditionRequired, domain.CodeSettingPreconditionRequired)
}

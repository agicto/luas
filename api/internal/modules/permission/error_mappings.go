package permission

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrAccessRoleNotFound, http.StatusNotFound, domain.CodeAccessRoleNotFound)
	mapper.Register(domain.ErrAccessRoleSlugAlreadyExists, http.StatusConflict, domain.CodeAccessRoleSlugAlreadyExists)
	mapper.Register(domain.ErrPermissionUnknown, http.StatusUnprocessableEntity, domain.CodePermissionUnknown)
}

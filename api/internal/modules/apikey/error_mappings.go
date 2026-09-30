package apikey

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrAPIKeyNotFound, http.StatusNotFound, domain.CodeAPIKeyNotFound)
	mapper.Register(domain.ErrAPIKeyInvalid, http.StatusUnauthorized, domain.CodeAPIKeyInvalid)
	mapper.Register(domain.ErrAPIKeyExpired, http.StatusUnauthorized, domain.CodeAPIKeyExpired)
	mapper.Register(domain.ErrAPIKeyRevoked, http.StatusUnauthorized, domain.CodeAPIKeyRevoked)
}

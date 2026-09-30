package notification

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrNotificationNotFound, http.StatusNotFound, domain.CodeNotificationNotFound)
	mapper.Register(domain.ErrNotificationIdempotencyConflict, http.StatusConflict, domain.CodeNotificationIdempotencyConflict)
	mapper.Register(domain.ErrNotificationInvalidChannel, http.StatusUnprocessableEntity, domain.CodeNotificationInvalidChannel)
}

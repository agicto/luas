package webhook

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrWebhookEndpointNotFound, http.StatusNotFound, domain.CodeWebhookEndpointNotFound)
	mapper.Register(domain.ErrWebhookDeliveryNotFound, http.StatusNotFound, domain.CodeWebhookDeliveryNotFound)
	mapper.Register(domain.ErrWebhookIdempotencyConflict, http.StatusConflict, domain.CodeWebhookIdempotencyConflict)
	mapper.Register(domain.ErrWebhookEndpointVersionConflict, http.StatusConflict, domain.CodeWebhookEndpointVersionConflict)
	mapper.Register(domain.ErrWebhookReplayNotAllowed, http.StatusConflict, domain.CodeWebhookReplayNotAllowed)
	mapper.Register(domain.ErrWebhookInvalidEventType, http.StatusUnprocessableEntity, domain.CodeWebhookInvalidEventType)
	mapper.Register(domain.ErrWebhookInvalidTarget, http.StatusUnprocessableEntity, domain.CodeWebhookInvalidTarget)
	mapper.Register(domain.ErrWebhookPreconditionRequired, http.StatusPreconditionRequired, domain.CodeWebhookPreconditionRequired)
}

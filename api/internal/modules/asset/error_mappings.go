package asset

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrAssetNotFound, http.StatusNotFound, domain.CodeAssetNotFound)
	mapper.Register(domain.ErrAssetNotReady, http.StatusConflict, domain.CodeAssetNotReady)
	mapper.Register(domain.ErrAssetIdempotencyConflict, http.StatusConflict, domain.CodeAssetIdempotencyConflict)
	mapper.Register(domain.ErrAssetCleanupRequired, http.StatusConflict, domain.CodeAssetCleanupRequired)
	mapper.Register(domain.ErrAssetUploadExpired, http.StatusGone, domain.CodeAssetUploadExpired)
	mapper.Register(domain.ErrAssetSizeExceeded, http.StatusRequestEntityTooLarge, domain.CodeAssetSizeExceeded)
	mapper.Register(domain.ErrAssetInvalidMediaType, http.StatusUnprocessableEntity, domain.CodeAssetInvalidMediaType)
}

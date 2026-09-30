package user

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrUserNotFound, http.StatusNotFound, domain.CodeUserNotFound)
	mapper.Register(domain.ErrInvalidCredentials, http.StatusUnauthorized, domain.CodeInvalidCredentials)
	mapper.Register(domain.ErrPasswordResetTokenInvalid, http.StatusUnauthorized, domain.CodePasswordResetTokenInvalid)
	mapper.Register(domain.ErrPasswordResetTokenExpired, http.StatusUnauthorized, domain.CodePasswordResetTokenExpired)
	mapper.Register(domain.ErrAccountDisabled, http.StatusForbidden, domain.CodeAccountDisabled)
	mapper.Register(domain.ErrEmailAlreadyExists, http.StatusConflict, domain.CodeEmailAlreadyExists)
	mapper.Register(domain.ErrUsernameAlreadyExists, http.StatusConflict, domain.CodeUsernameAlreadyExists)
}

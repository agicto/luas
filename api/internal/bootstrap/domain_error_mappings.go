package bootstrap

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter"
	"github.com/zgiai/luas/api/pkg/response"
)

// registerDomainErrorMappings installs the shared domain errors, then lets each active starter map
// its own errors. Adding a starter never requires editing this file.
func registerDomainErrorMappings(mapper *response.ErrorMapper, starters *starter.Registry) {
	if mapper == nil {
		return
	}
	registerCoreErrorMappings(mapper)
	starters.RegisterErrorMappings(mapper)
}

// registerCoreErrorMappings maps errors shared by several starters or by core runtime code.
func registerCoreErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrNotFound, http.StatusNotFound, domain.CodeNotFound)
	mapper.Register(domain.ErrRoleNotFound, http.StatusNotFound, domain.CodeRoleNotFound)
	mapper.Register(domain.ErrAuthenticationRequired, http.StatusUnauthorized, response.ErrorCodeUnauthorized)
	mapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
	mapper.Register(domain.ErrConflict, http.StatusConflict, domain.CodeConflict)
	mapper.Register(domain.ErrInvalidInput, http.StatusBadRequest, domain.CodeInvalidInput)
	mapper.Register(domain.ErrServiceUnavailable, http.StatusServiceUnavailable, domain.CodeServiceUnavailable)
}

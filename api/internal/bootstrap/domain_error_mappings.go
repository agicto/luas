package bootstrap

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

func registerDefaultDomainErrorMappings() {
	registerDomainErrorMappings(response.DefaultErrorMapper)
}

func registerDomainErrorMappings(mapper *response.ErrorMapper) {
	if mapper == nil {
		return
	}

	mapper.Register(domain.ErrNotFound, http.StatusNotFound, domain.CodeNotFound)
	mapper.Register(domain.ErrUserNotFound, http.StatusNotFound, domain.CodeUserNotFound)
	mapper.Register(domain.ErrRoleNotFound, http.StatusNotFound, domain.CodeRoleNotFound)
	mapper.Register(domain.ErrAPIKeyNotFound, http.StatusNotFound, domain.CodeAPIKeyNotFound)
	mapper.Register(domain.ErrOrganizationNotFound, http.StatusNotFound, domain.CodeOrganizationNotFound)
	mapper.Register(domain.ErrOrganizationInvitationNotFound, http.StatusNotFound, domain.CodeOrganizationInvitationNotFound)
	mapper.Register(domain.ErrOrganizationInvitationInvalid, http.StatusNotFound, domain.CodeOrganizationInvitationInvalid)

	mapper.Register(domain.ErrInvalidCredentials, http.StatusUnauthorized, domain.CodeInvalidCredentials)
	mapper.Register(domain.ErrAPIKeyInvalid, http.StatusUnauthorized, domain.CodeAPIKeyInvalid)
	mapper.Register(domain.ErrAPIKeyExpired, http.StatusUnauthorized, domain.CodeAPIKeyExpired)
	mapper.Register(domain.ErrAPIKeyRevoked, http.StatusUnauthorized, domain.CodeAPIKeyRevoked)
	mapper.Register(domain.ErrPasswordResetTokenInvalid, http.StatusUnauthorized, domain.CodePasswordResetTokenInvalid)
	mapper.Register(domain.ErrPasswordResetTokenExpired, http.StatusUnauthorized, domain.CodePasswordResetTokenExpired)

	mapper.Register(domain.ErrAccountDisabled, http.StatusForbidden, domain.CodeAccountDisabled)
	mapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
	mapper.Register(domain.ErrOrganizationInvitationEmailMismatch, http.StatusForbidden, domain.CodeOrganizationInvitationEmailMismatch)

	mapper.Register(domain.ErrEmailAlreadyExists, http.StatusConflict, domain.CodeEmailAlreadyExists)
	mapper.Register(domain.ErrUsernameAlreadyExists, http.StatusConflict, domain.CodeUsernameAlreadyExists)
	mapper.Register(domain.ErrConflict, http.StatusConflict, domain.CodeConflict)
	mapper.Register(domain.ErrOrganizationSlugAlreadyExists, http.StatusConflict, domain.CodeOrganizationSlugAlreadyExists)
	mapper.Register(domain.ErrOrganizationOwnershipTransferRequired, http.StatusConflict, domain.CodeOrganizationOwnershipTransferRequired)
	mapper.Register(domain.ErrOrganizationInvitationAlreadyPending, http.StatusConflict, domain.CodeOrganizationInvitationAlreadyPending)
	mapper.Register(domain.ErrOrganizationMemberAlreadyExists, http.StatusConflict, domain.CodeOrganizationMemberAlreadyExists)

	mapper.Register(domain.ErrOrganizationInvitationExpired, http.StatusGone, domain.CodeOrganizationInvitationExpired)

	mapper.Register(domain.ErrInvalidInput, http.StatusUnprocessableEntity, domain.CodeInvalidInput)
	mapper.Register(domain.ErrServiceUnavailable, http.StatusServiceUnavailable, domain.CodeServiceUnavailable)
}

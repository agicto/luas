package organization

import (
	"net/http"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// RegisterErrorMappings maps this starter's domain errors to public status codes and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(domain.ErrOrganizationNotFound, http.StatusNotFound, domain.CodeOrganizationNotFound)
	mapper.Register(domain.ErrOrganizationMemberNotFound, http.StatusNotFound, domain.CodeOrganizationMemberNotFound)
	mapper.Register(domain.ErrOrganizationInvitationNotFound, http.StatusNotFound, domain.CodeOrganizationInvitationNotFound)
	mapper.Register(domain.ErrOrganizationInvitationInvalid, http.StatusNotFound, domain.CodeOrganizationInvitationInvalid)
	mapper.Register(domain.ErrOrganizationContextRequired, http.StatusBadRequest, domain.CodeOrganizationContextRequired)
	mapper.Register(domain.ErrOrganizationContextInvalid, http.StatusBadRequest, domain.CodeOrganizationContextInvalid)
	mapper.Register(domain.ErrOrganizationInvitationEmailMismatch, http.StatusForbidden, domain.CodeOrganizationInvitationEmailMismatch)
	mapper.Register(domain.ErrOrganizationSlugAlreadyExists, http.StatusConflict, domain.CodeOrganizationSlugAlreadyExists)
	mapper.Register(domain.ErrOrganizationOwnershipTransferRequired, http.StatusConflict, domain.CodeOrganizationOwnershipTransferRequired)
	mapper.Register(domain.ErrOrganizationOwnershipTransferTargetInvalid, http.StatusConflict, domain.CodeOrganizationOwnershipTransferTargetInvalid)
	mapper.Register(domain.ErrOrganizationMembershipExitRequired, http.StatusConflict, domain.CodeOrganizationMembershipExitRequired)
	mapper.Register(domain.ErrOrganizationInvitationAlreadyPending, http.StatusConflict, domain.CodeOrganizationInvitationAlreadyPending)
	mapper.Register(domain.ErrOrganizationMemberAlreadyExists, http.StatusConflict, domain.CodeOrganizationMemberAlreadyExists)
	mapper.Register(domain.ErrOrganizationInvitationExpired, http.StatusGone, domain.CodeOrganizationInvitationExpired)
}

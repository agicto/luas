package bootstrap

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestRegisterDomainErrorMappings(t *testing.T) {
	mapper := &response.ErrorMapper{}
	registerDomainErrorMappings(mapper)

	tests := []struct {
		name       string
		err        error
		statusCode int
		errorCode  string
	}{
		{
			name:       "wrapped user not found",
			err:        fmt.Errorf("load profile: %w", domain.ErrUserNotFound),
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeUserNotFound,
		},
		{
			name:       "invalid credentials",
			err:        domain.ErrInvalidCredentials,
			statusCode: http.StatusUnauthorized,
			errorCode:  domain.CodeInvalidCredentials,
		},
		{
			name:       "permission denied",
			err:        domain.ErrPermissionDenied,
			statusCode: http.StatusForbidden,
			errorCode:  domain.CodePermissionDenied,
		},
		{
			name:       "access role not found",
			err:        domain.ErrAccessRoleNotFound,
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeAccessRoleNotFound,
		},
		{
			name:       "access role slug already exists",
			err:        domain.ErrAccessRoleSlugAlreadyExists,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeAccessRoleSlugAlreadyExists,
		},
		{
			name:       "permission is unknown",
			err:        domain.ErrPermissionUnknown,
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  domain.CodePermissionUnknown,
		},
		{
			name:       "username already exists",
			err:        domain.ErrUsernameAlreadyExists,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeUsernameAlreadyExists,
		},
		{
			name:       "invalid input",
			err:        domain.ErrInvalidInput,
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  domain.CodeInvalidInput,
		},
		{
			name:       "organization is not visible",
			err:        fmt.Errorf("find membership: %w", domain.ErrOrganizationNotFound),
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeOrganizationNotFound,
		},
		{
			name:       "organization ownership blocks account deletion",
			err:        domain.ErrOrganizationOwnershipTransferRequired,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeOrganizationOwnershipTransferRequired,
		},
		{
			name:       "organization ownership target is invalid",
			err:        domain.ErrOrganizationOwnershipTransferTargetInvalid,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeOrganizationOwnershipTransferTargetInvalid,
		},
		{
			name:       "organization context is required",
			err:        domain.ErrOrganizationContextRequired,
			statusCode: http.StatusBadRequest,
			errorCode:  domain.CodeOrganizationContextRequired,
		},
		{
			name:       "organization context is invalid",
			err:        domain.ErrOrganizationContextInvalid,
			statusCode: http.StatusBadRequest,
			errorCode:  domain.CodeOrganizationContextInvalid,
		},
		{
			name:       "organization memberships block account deletion",
			err:        domain.ErrOrganizationMembershipExitRequired,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeOrganizationMembershipExitRequired,
		},
		{
			name:       "organization member is not visible",
			err:        domain.ErrOrganizationMemberNotFound,
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeOrganizationMemberNotFound,
		},
		{
			name:       "organization invitation is not visible",
			err:        domain.ErrOrganizationInvitationNotFound,
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeOrganizationInvitationNotFound,
		},
		{
			name:       "organization invitation token is invalid",
			err:        domain.ErrOrganizationInvitationInvalid,
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeOrganizationInvitationInvalid,
		},
		{
			name:       "organization invitation expired",
			err:        domain.ErrOrganizationInvitationExpired,
			statusCode: http.StatusGone,
			errorCode:  domain.CodeOrganizationInvitationExpired,
		},
		{
			name:       "organization invitation email mismatch",
			err:        domain.ErrOrganizationInvitationEmailMismatch,
			statusCode: http.StatusForbidden,
			errorCode:  domain.CodeOrganizationInvitationEmailMismatch,
		},
		{
			name:       "organization invitation already pending",
			err:        domain.ErrOrganizationInvitationAlreadyPending,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeOrganizationInvitationAlreadyPending,
		},
		{
			name:       "organization member already exists",
			err:        domain.ErrOrganizationMemberAlreadyExists,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeOrganizationMemberAlreadyExists,
		},
		{
			name:       "wrapped service unavailable",
			err:        fmt.Errorf("load dependency: %w", domain.ErrServiceUnavailable),
			statusCode: http.StatusServiceUnavailable,
			errorCode:  domain.CodeServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			descriptor := mapper.Resolve(tt.err)
			if descriptor.StatusCode != tt.statusCode {
				t.Fatalf("status = %d, want %d", descriptor.StatusCode, tt.statusCode)
			}
			if descriptor.ErrorCode != tt.errorCode {
				t.Fatalf("error_code = %q, want %q", descriptor.ErrorCode, tt.errorCode)
			}
		})
	}
}

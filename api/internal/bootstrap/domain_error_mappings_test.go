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
			name:       "notification is not visible",
			err:        fmt.Errorf("replace read state: %w", domain.ErrNotificationNotFound),
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeNotificationNotFound,
		},
		{
			name:       "notification idempotency conflicts",
			err:        domain.ErrNotificationIdempotencyConflict,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeNotificationIdempotencyConflict,
		},
		{
			name:       "notification channel is invalid",
			err:        domain.ErrNotificationInvalidChannel,
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  domain.CodeNotificationInvalidChannel,
		},
		{
			name:       "asset is not visible",
			err:        fmt.Errorf("load asset: %w", domain.ErrAssetNotFound),
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeAssetNotFound,
		},
		{
			name:       "asset is not ready",
			err:        domain.ErrAssetNotReady,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeAssetNotReady,
		},
		{
			name:       "asset idempotency conflicts",
			err:        domain.ErrAssetIdempotencyConflict,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeAssetIdempotencyConflict,
		},
		{
			name:       "asset cleanup blocks account deletion",
			err:        domain.ErrAssetCleanupRequired,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeAssetCleanupRequired,
		},
		{
			name:       "asset upload expired",
			err:        domain.ErrAssetUploadExpired,
			statusCode: http.StatusGone,
			errorCode:  domain.CodeAssetUploadExpired,
		},
		{
			name:       "asset size exceeded",
			err:        domain.ErrAssetSizeExceeded,
			statusCode: http.StatusRequestEntityTooLarge,
			errorCode:  domain.CodeAssetSizeExceeded,
		},
		{
			name:       "asset media type is invalid",
			err:        domain.ErrAssetInvalidMediaType,
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  domain.CodeAssetInvalidMediaType,
		},
		{
			name:       "setting is not registered",
			err:        domain.ErrSettingNotFound,
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeSettingNotFound,
		},
		{
			name:       "setting value is invalid",
			err:        domain.ErrSettingInvalidValue,
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  domain.CodeSettingInvalidValue,
		},
		{
			name:       "setting version is stale",
			err:        domain.ErrSettingVersionConflict,
			statusCode: http.StatusPreconditionFailed,
			errorCode:  domain.CodeSettingVersionConflict,
		},
		{
			name:       "setting version is required",
			err:        domain.ErrSettingPreconditionRequired,
			statusCode: http.StatusPreconditionRequired,
			errorCode:  domain.CodeSettingPreconditionRequired,
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

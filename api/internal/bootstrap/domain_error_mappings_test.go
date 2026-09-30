package bootstrap

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/apikey"
	"github.com/zgiai/luas/api/internal/modules/asset"
	"github.com/zgiai/luas/api/internal/modules/audit"
	"github.com/zgiai/luas/api/internal/modules/notification"
	"github.com/zgiai/luas/api/internal/modules/operator"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/permission"
	"github.com/zgiai/luas/api/internal/modules/setting"
	"github.com/zgiai/luas/api/internal/modules/usage"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/modules/webhook"
	"github.com/zgiai/luas/api/internal/starter"
	"github.com/zgiai/luas/api/pkg/response"
)

func allStartersRegistry() *starter.Registry {
	registry := starter.NewRegistry()
	for _, module := range []interface{ Name() string }{
		&audit.Handler{}, &apikey.Handler{}, &user.Handler{}, &organization.Handler{},
		&permission.Handler{}, &notification.Handler{}, &asset.Handler{}, &setting.Handler{},
		&usage.Handler{}, &webhook.Handler{}, &operator.Handler{},
	} {
		registry.RegisterModule(module)
	}
	return registry
}

// TestDomainErrorMappingsAreUnchangedWhenOwnedByStarters pins every mapping that used to live in one
// central bootstrap list. Each starter now registers its own errors; the public contract must not move.
func TestDomainErrorMappingsAreUnchangedWhenOwnedByStarters(t *testing.T) {
	mapper := &response.ErrorMapper{}
	registerDomainErrorMappings(mapper, allStartersRegistry())

	tests := []struct {
		err        error
		statusCode int
		errorCode  string
	}{
		{domain.ErrNotFound, http.StatusNotFound, domain.CodeNotFound},
		{domain.ErrUserNotFound, http.StatusNotFound, domain.CodeUserNotFound},
		{domain.ErrRoleNotFound, http.StatusNotFound, domain.CodeRoleNotFound},
		{domain.ErrAccessRoleNotFound, http.StatusNotFound, domain.CodeAccessRoleNotFound},
		{domain.ErrAPIKeyNotFound, http.StatusNotFound, domain.CodeAPIKeyNotFound},
		{domain.ErrOrganizationNotFound, http.StatusNotFound, domain.CodeOrganizationNotFound},
		{domain.ErrOrganizationMemberNotFound, http.StatusNotFound, domain.CodeOrganizationMemberNotFound},
		{domain.ErrOrganizationInvitationNotFound, http.StatusNotFound, domain.CodeOrganizationInvitationNotFound},
		{domain.ErrOrganizationInvitationInvalid, http.StatusNotFound, domain.CodeOrganizationInvitationInvalid},
		{domain.ErrNotificationNotFound, http.StatusNotFound, domain.CodeNotificationNotFound},
		{domain.ErrAssetNotFound, http.StatusNotFound, domain.CodeAssetNotFound},
		{domain.ErrSettingNotFound, http.StatusNotFound, domain.CodeSettingNotFound},
		{domain.ErrUsageMetricNotFound, http.StatusNotFound, domain.CodeUsageMetricNotFound},
		{domain.ErrWebhookEndpointNotFound, http.StatusNotFound, domain.CodeWebhookEndpointNotFound},
		{domain.ErrWebhookDeliveryNotFound, http.StatusNotFound, domain.CodeWebhookDeliveryNotFound},
		{domain.ErrOrganizationContextRequired, http.StatusBadRequest, domain.CodeOrganizationContextRequired},
		{domain.ErrOrganizationContextInvalid, http.StatusBadRequest, domain.CodeOrganizationContextInvalid},
		{domain.ErrInvalidCredentials, http.StatusUnauthorized, domain.CodeInvalidCredentials},
		{domain.ErrAuthenticationRequired, http.StatusUnauthorized, response.ErrorCodeUnauthorized},
		{domain.ErrAPIKeyInvalid, http.StatusUnauthorized, domain.CodeAPIKeyInvalid},
		{domain.ErrAPIKeyExpired, http.StatusUnauthorized, domain.CodeAPIKeyExpired},
		{domain.ErrAPIKeyRevoked, http.StatusUnauthorized, domain.CodeAPIKeyRevoked},
		{domain.ErrPasswordResetTokenInvalid, http.StatusUnauthorized, domain.CodePasswordResetTokenInvalid},
		{domain.ErrPasswordResetTokenExpired, http.StatusUnauthorized, domain.CodePasswordResetTokenExpired},
		{domain.ErrAccountDisabled, http.StatusForbidden, domain.CodeAccountDisabled},
		{domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied},
		{domain.ErrOrganizationInvitationEmailMismatch, http.StatusForbidden, domain.CodeOrganizationInvitationEmailMismatch},
		{domain.ErrEmailAlreadyExists, http.StatusConflict, domain.CodeEmailAlreadyExists},
		{domain.ErrUsernameAlreadyExists, http.StatusConflict, domain.CodeUsernameAlreadyExists},
		{domain.ErrConflict, http.StatusConflict, domain.CodeConflict},
		{domain.ErrOrganizationSlugAlreadyExists, http.StatusConflict, domain.CodeOrganizationSlugAlreadyExists},
		{domain.ErrOrganizationOwnershipTransferRequired, http.StatusConflict, domain.CodeOrganizationOwnershipTransferRequired},
		{domain.ErrOrganizationOwnershipTransferTargetInvalid, http.StatusConflict, domain.CodeOrganizationOwnershipTransferTargetInvalid},
		{domain.ErrOrganizationMembershipExitRequired, http.StatusConflict, domain.CodeOrganizationMembershipExitRequired},
		{domain.ErrOrganizationInvitationAlreadyPending, http.StatusConflict, domain.CodeOrganizationInvitationAlreadyPending},
		{domain.ErrOrganizationMemberAlreadyExists, http.StatusConflict, domain.CodeOrganizationMemberAlreadyExists},
		{domain.ErrAccessRoleSlugAlreadyExists, http.StatusConflict, domain.CodeAccessRoleSlugAlreadyExists},
		{domain.ErrNotificationIdempotencyConflict, http.StatusConflict, domain.CodeNotificationIdempotencyConflict},
		{domain.ErrAssetNotReady, http.StatusConflict, domain.CodeAssetNotReady},
		{domain.ErrAssetIdempotencyConflict, http.StatusConflict, domain.CodeAssetIdempotencyConflict},
		{domain.ErrAssetCleanupRequired, http.StatusConflict, domain.CodeAssetCleanupRequired},
		{domain.ErrUsageIdempotencyConflict, http.StatusConflict, domain.CodeUsageIdempotencyConflict},
		{domain.ErrWebhookIdempotencyConflict, http.StatusConflict, domain.CodeWebhookIdempotencyConflict},
		{domain.ErrWebhookEndpointVersionConflict, http.StatusConflict, domain.CodeWebhookEndpointVersionConflict},
		{domain.ErrWebhookReplayNotAllowed, http.StatusConflict, domain.CodeWebhookReplayNotAllowed},
		{domain.ErrSettingVersionConflict, http.StatusPreconditionFailed, domain.CodeSettingVersionConflict},
		{domain.ErrUsageQuotaVersionConflict, http.StatusPreconditionFailed, domain.CodeUsageQuotaVersionConflict},
		{domain.ErrOrganizationInvitationExpired, http.StatusGone, domain.CodeOrganizationInvitationExpired},
		{domain.ErrAssetUploadExpired, http.StatusGone, domain.CodeAssetUploadExpired},
		{domain.ErrAssetSizeExceeded, http.StatusRequestEntityTooLarge, domain.CodeAssetSizeExceeded},
		{domain.ErrInvalidInput, http.StatusUnprocessableEntity, domain.CodeInvalidInput},
		{domain.ErrPermissionUnknown, http.StatusUnprocessableEntity, domain.CodePermissionUnknown},
		{domain.ErrNotificationInvalidChannel, http.StatusUnprocessableEntity, domain.CodeNotificationInvalidChannel},
		{domain.ErrAssetInvalidMediaType, http.StatusUnprocessableEntity, domain.CodeAssetInvalidMediaType},
		{domain.ErrSettingInvalidValue, http.StatusUnprocessableEntity, domain.CodeSettingInvalidValue},
		{domain.ErrUsageInvalidEvent, http.StatusUnprocessableEntity, domain.CodeUsageInvalidEvent},
		{domain.ErrUsageEventOutsideWindow, http.StatusUnprocessableEntity, domain.CodeUsageEventOutsideWindow},
		{domain.ErrWebhookInvalidEventType, http.StatusUnprocessableEntity, domain.CodeWebhookInvalidEventType},
		{domain.ErrWebhookInvalidTarget, http.StatusUnprocessableEntity, domain.CodeWebhookInvalidTarget},
		{domain.ErrSettingPreconditionRequired, http.StatusPreconditionRequired, domain.CodeSettingPreconditionRequired},
		{domain.ErrUsagePreconditionRequired, http.StatusPreconditionRequired, domain.CodeUsagePreconditionRequired},
		{domain.ErrWebhookPreconditionRequired, http.StatusPreconditionRequired, domain.CodeWebhookPreconditionRequired},
		{domain.ErrUsageQuotaExceeded, http.StatusTooManyRequests, domain.CodeUsageQuotaExceeded},
		{domain.ErrServiceUnavailable, http.StatusServiceUnavailable, domain.CodeServiceUnavailable},
	}
	for _, test := range tests {
		wrapped := fmt.Errorf("context: %w", test.err)
		descriptor := mapper.Resolve(wrapped)
		if descriptor.StatusCode != test.statusCode || descriptor.ErrorCode != test.errorCode {
			t.Errorf("%v resolved to %d %s, want %d %s", test.err, descriptor.StatusCode, descriptor.ErrorCode, test.statusCode, test.errorCode)
		}
	}
}

func TestCoreErrorMappingsDoNotDependOnOptionalStarters(t *testing.T) {
	mapper := &response.ErrorMapper{}
	registerDomainErrorMappings(mapper, nil)

	if got := mapper.Resolve(domain.ErrPermissionDenied); got.StatusCode != http.StatusForbidden {
		t.Fatalf("permission denied must stay mapped without the permission starter, got %d", got.StatusCode)
	}
	if got := mapper.Resolve(domain.ErrServiceUnavailable); got.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("service unavailable must stay a core mapping, got %d", got.StatusCode)
	}
}

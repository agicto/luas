package domain

// Stable machine-readable error codes used by the API layer.
//
// Naming rules:
// - UPPERCASE segments separated by dots
// - first segment = scope/domain
// - remaining segment(s) = specific reason
//
// Examples:
// - USER.NOT_FOUND
// - AUTH.INVALID_CREDENTIALS
// - API_KEY.REVOKED
const (
	CodeInternal           = "COMMON.INTERNAL"
	CodeNotFound           = "COMMON.NOT_FOUND"
	CodeConflict           = "COMMON.CONFLICT"
	CodeInvalidInput       = "COMMON.INVALID_INPUT"
	CodeValidationFailed   = "COMMON.VALIDATION_FAILED"
	CodeRateLimited        = "COMMON.RATE_LIMITED"
	CodeRequestTooLarge    = "COMMON.REQUEST_TOO_LARGE"
	CodeTimeout            = "COMMON.TIMEOUT"
	CodeServiceUnavailable = "COMMON.SERVICE_UNAVAILABLE"

	CodeUserNotFound              = "USER.NOT_FOUND"
	CodeEmailAlreadyExists        = "USER.EMAIL_ALREADY_EXISTS"
	CodeUsernameAlreadyExists     = "USER.USERNAME_ALREADY_EXISTS"
	CodeInvalidCredentials        = "AUTH.INVALID_CREDENTIALS"
	CodeAccountDisabled           = "AUTH.ACCOUNT_DISABLED"
	CodePasswordResetTokenInvalid = "AUTH.PASSWORD_RESET_TOKEN_INVALID"
	CodePasswordResetTokenExpired = "AUTH.PASSWORD_RESET_TOKEN_EXPIRED"

	CodePermissionDenied            = "PERMISSION.DENIED"
	CodePermissionUnknown           = "PERMISSION.UNKNOWN"
	CodeAccessRoleNotFound          = "PERMISSION.ROLE_NOT_FOUND"
	CodeAccessRoleSlugAlreadyExists = "PERMISSION.ROLE_SLUG_ALREADY_EXISTS"
	CodeRoleNotFound                = "ROLE.NOT_FOUND" // Deprecated: use CodeAccessRoleNotFound.

	CodeAPIKeyNotFound = "API_KEY.NOT_FOUND"
	CodeAPIKeyInvalid  = "API_KEY.INVALID"
	CodeAPIKeyExpired  = "API_KEY.EXPIRED"
	CodeAPIKeyRevoked  = "API_KEY.REVOKED"

	CodeOrganizationNotFound                       = "ORGANIZATION.NOT_FOUND"
	CodeOrganizationContextRequired                = "ORGANIZATION.CONTEXT_REQUIRED"
	CodeOrganizationContextInvalid                 = "ORGANIZATION.CONTEXT_INVALID"
	CodeOrganizationSlugAlreadyExists              = "ORGANIZATION.SLUG_ALREADY_EXISTS"
	CodeOrganizationOwnershipTransferRequired      = "ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED"
	CodeOrganizationOwnershipTransferTargetInvalid = "ORGANIZATION.OWNERSHIP_TRANSFER_TARGET_INVALID"
	CodeOrganizationMembershipExitRequired         = "ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED"
	CodeOrganizationMemberNotFound                 = "ORGANIZATION.MEMBER_NOT_FOUND"
	CodeOrganizationInvitationNotFound             = "ORGANIZATION.INVITATION.NOT_FOUND"
	CodeOrganizationInvitationInvalid              = "ORGANIZATION.INVITATION.INVALID"
	CodeOrganizationInvitationExpired              = "ORGANIZATION.INVITATION.EXPIRED"
	CodeOrganizationInvitationEmailMismatch        = "ORGANIZATION.INVITATION.EMAIL_MISMATCH"
	CodeOrganizationInvitationAlreadyPending       = "ORGANIZATION.INVITATION.ALREADY_PENDING"
	CodeOrganizationMemberAlreadyExists            = "ORGANIZATION.MEMBER_ALREADY_EXISTS"

	CodeNotificationNotFound            = "NOTIFICATION.NOT_FOUND"
	CodeNotificationIdempotencyConflict = "NOTIFICATION.IDEMPOTENCY_CONFLICT"
	CodeNotificationInvalidChannel      = "NOTIFICATION.INVALID_CHANNEL"

	CodeAssetNotFound            = "ASSET.NOT_FOUND"
	CodeAssetNotReady            = "ASSET.NOT_READY"
	CodeAssetIdempotencyConflict = "ASSET.IDEMPOTENCY_CONFLICT"
	CodeAssetCleanupRequired     = "ASSET.CLEANUP_REQUIRED"
	CodeAssetUploadExpired       = "ASSET.UPLOAD_EXPIRED"
	CodeAssetSizeExceeded        = "ASSET.SIZE_EXCEEDED"
	CodeAssetInvalidMediaType    = "ASSET.INVALID_MEDIA_TYPE"

	CodeSettingNotFound             = "SETTING.NOT_FOUND"
	CodeSettingInvalidValue         = "SETTING.INVALID_VALUE"
	CodeSettingVersionConflict      = "SETTING.VERSION_CONFLICT"
	CodeSettingPreconditionRequired = "SETTING.PRECONDITION_REQUIRED"
)

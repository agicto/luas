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
	CodeNotFound     = "COMMON.NOT_FOUND"
	CodeConflict     = "COMMON.CONFLICT"
	CodeInvalidInput = "COMMON.INVALID_INPUT"

	CodeUserNotFound              = "USER.NOT_FOUND"
	CodeEmailAlreadyExists        = "USER.EMAIL_ALREADY_EXISTS"
	CodeUsernameAlreadyExists     = "USER.USERNAME_ALREADY_EXISTS"
	CodeInvalidCredentials        = "AUTH.INVALID_CREDENTIALS"
	CodeAccountDisabled           = "AUTH.ACCOUNT_DISABLED"
	CodePasswordResetTokenInvalid = "AUTH.PASSWORD_RESET_TOKEN_INVALID"
	CodePasswordResetTokenExpired = "AUTH.PASSWORD_RESET_TOKEN_EXPIRED"

	CodePermissionDenied = "PERMISSION.DENIED"
	CodeRoleNotFound     = "ROLE.NOT_FOUND"

	CodeAPIKeyNotFound = "API_KEY.NOT_FOUND"
	CodeAPIKeyInvalid  = "API_KEY.INVALID"
	CodeAPIKeyExpired  = "API_KEY.EXPIRED"
	CodeAPIKeyRevoked  = "API_KEY.REVOKED"
)

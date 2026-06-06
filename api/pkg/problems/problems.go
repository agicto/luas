package problems

import "errors"

// Stable machine-readable error codes used by the API layer.
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

	CodeTeamNotFound          = "TEAM.NOT_FOUND"
	CodeTeamSlugAlreadyExists = "TEAM.SLUG_ALREADY_EXISTS"

	CodeAccessRoleNotFound          = "ACCESS_ROLE.NOT_FOUND"
	CodeAccessRoleSlugAlreadyExists = "ACCESS_ROLE.SLUG_ALREADY_EXISTS"
)

// Stable sentinel errors that can be mapped by transport packages without importing internal domain packages.
var (
	ErrUserNotFound              = errors.New("user not found")
	ErrEmailAlreadyExists        = errors.New("email already registered")
	ErrUsernameAlreadyExists     = errors.New("username already registered")
	ErrInvalidCredentials        = errors.New("invalid username or password")
	ErrAccountDisabled           = errors.New("account is disabled")
	ErrPasswordResetTokenInvalid = errors.New("password reset token is invalid")
	ErrPasswordResetTokenExpired = errors.New("password reset token is expired")

	ErrPermissionDenied = errors.New("permission denied")
	ErrRoleNotFound     = errors.New("role not found")

	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyInvalid  = errors.New("api key is invalid")
	ErrAPIKeyExpired  = errors.New("api key is expired")
	ErrAPIKeyRevoked  = errors.New("api key is revoked")

	ErrTeamNotFound          = errors.New("team not found")
	ErrTeamSlugAlreadyExists = errors.New("team slug already exists")

	ErrAccessRoleNotFound          = errors.New("access role not found")
	ErrAccessRoleSlugAlreadyExists = errors.New("access role slug already exists")

	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource already exists")
	ErrInvalidInput = errors.New("invalid input")
)

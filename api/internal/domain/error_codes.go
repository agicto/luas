package domain

import "github.com/zgiai/luas/api/pkg/problems"

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
	CodeNotFound     = problems.CodeNotFound
	CodeConflict     = problems.CodeConflict
	CodeInvalidInput = problems.CodeInvalidInput

	CodeUserNotFound              = problems.CodeUserNotFound
	CodeEmailAlreadyExists        = problems.CodeEmailAlreadyExists
	CodeUsernameAlreadyExists     = problems.CodeUsernameAlreadyExists
	CodeInvalidCredentials        = problems.CodeInvalidCredentials
	CodeAccountDisabled           = problems.CodeAccountDisabled
	CodePasswordResetTokenInvalid = problems.CodePasswordResetTokenInvalid
	CodePasswordResetTokenExpired = problems.CodePasswordResetTokenExpired

	CodePermissionDenied = problems.CodePermissionDenied
	CodeRoleNotFound     = problems.CodeRoleNotFound

	CodeAPIKeyNotFound = problems.CodeAPIKeyNotFound
	CodeAPIKeyInvalid  = problems.CodeAPIKeyInvalid
	CodeAPIKeyExpired  = problems.CodeAPIKeyExpired
	CodeAPIKeyRevoked  = problems.CodeAPIKeyRevoked
)

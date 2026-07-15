package domain

import (
	"errors"
	"time"
)

// Domain-specific errors
// These are business errors that can be returned by any layer
var (
	// User errors
	ErrUserNotFound              = errors.New("user not found")
	ErrEmailAlreadyExists        = errors.New("email already registered")
	ErrUsernameAlreadyExists     = errors.New("username already registered")
	ErrInvalidCredentials        = errors.New("invalid username or password")
	ErrAccountDisabled           = errors.New("account is disabled")
	ErrPasswordResetTokenInvalid = errors.New("password reset token is invalid")
	ErrPasswordResetTokenExpired = errors.New("password reset token is expired")

	// Permission errors
	ErrPermissionDenied            = errors.New("permission denied")
	ErrPermissionUnknown           = errors.New("permission is not registered")
	ErrAccessRoleNotFound          = errors.New("access role not found")
	ErrAccessRoleSlugAlreadyExists = errors.New("access role slug already exists")
	ErrRoleNotFound                = errors.New("role not found") // Deprecated: use ErrAccessRoleNotFound.

	// API key errors
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyInvalid  = errors.New("api key is invalid")
	ErrAPIKeyExpired  = errors.New("api key is expired")
	ErrAPIKeyRevoked  = errors.New("api key is revoked")

	// Organization errors
	ErrOrganizationNotFound                       = errors.New("organization not found")
	ErrOrganizationContextRequired                = errors.New("organization context is required")
	ErrOrganizationContextInvalid                 = errors.New("organization context is invalid")
	ErrOrganizationSlugAlreadyExists              = errors.New("organization slug already exists")
	ErrOrganizationOwnershipTransferRequired      = errors.New("organization ownership must be transferred first")
	ErrOrganizationOwnershipTransferTargetInvalid = errors.New("organization ownership transfer target is invalid")
	ErrOrganizationMembershipExitRequired         = errors.New("organization memberships must be exited before account deletion")
	ErrOrganizationMemberNotFound                 = errors.New("organization member not found")
	ErrOrganizationInvitationNotFound             = errors.New("organization invitation not found")
	ErrOrganizationInvitationInvalid              = errors.New("organization invitation is invalid")
	ErrOrganizationInvitationExpired              = errors.New("organization invitation is expired")
	ErrOrganizationInvitationEmailMismatch        = errors.New("organization invitation belongs to another account")
	ErrOrganizationInvitationAlreadyPending       = errors.New("organization invitation is already pending")
	ErrOrganizationMemberAlreadyExists            = errors.New("organization member already exists")

	// Notification errors
	ErrNotificationNotFound            = errors.New("notification not found")
	ErrNotificationIdempotencyConflict = errors.New("notification idempotency key conflicts with existing content")
	ErrNotificationInvalidChannel      = errors.New("notification channel is invalid")

	// Asset errors
	ErrAssetNotFound            = errors.New("asset not found")
	ErrAssetNotReady            = errors.New("asset is not ready")
	ErrAssetIdempotencyConflict = errors.New("asset idempotency key conflicts with existing metadata")
	ErrAssetCleanupRequired     = errors.New("active assets must be deleted before account deletion")
	ErrAssetUploadExpired       = errors.New("asset upload expired")
	ErrAssetSizeExceeded        = errors.New("asset size exceeds policy")
	ErrAssetInvalidMediaType    = errors.New("asset media type is invalid")

	// Setting errors
	ErrSettingNotFound             = errors.New("setting not found")
	ErrSettingInvalidValue         = errors.New("setting value is invalid")
	ErrSettingVersionConflict      = errors.New("setting version conflict")
	ErrSettingPreconditionRequired = errors.New("setting version precondition is required")

	// Generic errors
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrServiceUnavailable = errors.New("required service unavailable")
)

// Events
const (
	EventUserCreated = "user.created"
)

// UserCreatedEvent is triggered when a new user registers
type UserCreatedEvent struct {
	User       *User
	occurredAt time.Time
}

func NewUserCreatedEvent(user *User) UserCreatedEvent {
	return UserCreatedEvent{
		User:       user,
		occurredAt: time.Now(),
	}
}

func (e UserCreatedEvent) EventName() string {
	return EventUserCreated
}

func (e UserCreatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e UserCreatedEvent) Data() any {
	return e.User
}

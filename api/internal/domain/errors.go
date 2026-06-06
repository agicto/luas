package domain

import (
	"time"

	"github.com/zgiai/luas/api/pkg/problems"
)

// Domain-specific errors
// These are business errors that can be returned by any layer
var (
	// User errors
	ErrUserNotFound              = problems.ErrUserNotFound
	ErrEmailAlreadyExists        = problems.ErrEmailAlreadyExists
	ErrUsernameAlreadyExists     = problems.ErrUsernameAlreadyExists
	ErrInvalidCredentials        = problems.ErrInvalidCredentials
	ErrAccountDisabled           = problems.ErrAccountDisabled
	ErrPasswordResetTokenInvalid = problems.ErrPasswordResetTokenInvalid
	ErrPasswordResetTokenExpired = problems.ErrPasswordResetTokenExpired

	// Permission errors
	ErrPermissionDenied = problems.ErrPermissionDenied
	ErrRoleNotFound     = problems.ErrRoleNotFound

	// API key errors
	ErrAPIKeyNotFound = problems.ErrAPIKeyNotFound
	ErrAPIKeyInvalid  = problems.ErrAPIKeyInvalid
	ErrAPIKeyExpired  = problems.ErrAPIKeyExpired
	ErrAPIKeyRevoked  = problems.ErrAPIKeyRevoked

	// Team errors
	ErrTeamNotFound          = problems.ErrTeamNotFound
	ErrTeamSlugAlreadyExists = problems.ErrTeamSlugAlreadyExists

	// Access errors
	ErrAccessRoleNotFound          = problems.ErrAccessRoleNotFound
	ErrAccessRoleSlugAlreadyExists = problems.ErrAccessRoleSlugAlreadyExists

	// Generic errors
	ErrNotFound     = problems.ErrNotFound
	ErrConflict     = problems.ErrConflict
	ErrInvalidInput = problems.ErrInvalidInput
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

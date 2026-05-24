package user

import (
	"context"

	"github.com/zgiai/zgo/internal/domain"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/pkg/logger"
)

// handleUserCreated sends a welcome email when a user is created.
func (h *Handler) handleUserCreated(ctx context.Context, e events.Event) error {
	var user *domain.User

	// Try to get the underlying domain event
	// The infra layer wraps simple domain events in WrappedEvent
	var underlying any = e
	if wrapped, ok := e.(events.WrappedEvent); ok {
		underlying = wrapped.Event
	}

	// Double check the type
	if userEvent, ok := underlying.(domain.UserCreatedEvent); ok {
		user = userEvent.User
	}

	if user == nil {
		return nil
	}

	if h.mailer == nil {
		return nil
	}

	if err := h.mailer.SendWelcomeEmail(user.Email, user.Username); err != nil {
		logger.Error("failed to send welcome email", map[string]any{
			"error": err,
			"user":  user.Username,
		})
		return err
	}

	return nil
}

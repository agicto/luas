package user

import (
	"context"
	"log/slog"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/events"
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

	if h.mailer == nil || !h.mailer.IsConfigured() {
		return nil
	}

	if err := h.mailer.SendWelcomeEmail(ctx, user.Email, user.Username); err != nil {
		slog.ErrorContext(ctx, "user.welcome_email_delivery_failed",
			"user_id", user.ID,
			"err", err,
		)
		return err
	}

	return nil
}

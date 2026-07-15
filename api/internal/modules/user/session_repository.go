package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

type authenticationSessionRow struct {
	SessionID     string     `gorm:"column:session_id"`
	UserID        uint       `gorm:"column:user_id"`
	Username      string     `gorm:"column:username"`
	UserStatus    int        `gorm:"column:user_status"`
	UserDeletedAt *time.Time `gorm:"column:user_deleted_at"`
	ExpiresAt     time.Time  `gorm:"column:expires_at"`
	IdleExpiresAt time.Time  `gorm:"column:idle_expires_at"`
	LastSeenAt    time.Time  `gorm:"column:last_seen_at"`
	RevokedAt     *time.Time `gorm:"column:revoked_at"`
}

func (r *repository) createAuthenticationSession(ctx context.Context, session *AuthenticationSessionPO) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrInvalidInput
	}
	if err := db.Create(session).Error; err != nil {
		return authenticationPersistenceError("create", err)
	}
	return nil
}

func (r *repository) authenticateSession(
	ctx context.Context,
	tokenHash string,
	now time.Time,
	touchInterval time.Duration,
	idleTimeout time.Duration,
) (*domain.AuthenticationIdentity, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}

	var row authenticationSessionRow
	err = db.Table("authentication_sessions AS sessions").
		Select([]string{
			"sessions.id AS session_id",
			"sessions.user_id",
			"sessions.expires_at",
			"sessions.idle_expires_at",
			"sessions.last_seen_at",
			"sessions.revoked_at",
			"users.username",
			"users.status AS user_status",
			"users.deleted_at AS user_deleted_at",
		}).
		Joins("LEFT JOIN users ON users.id = sessions.user_id").
		Where("sessions.token_hash = ?", tokenHash).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrAuthenticationRequired
	}
	if err != nil {
		return nil, authenticationPersistenceError("resolve", err)
	}

	if row.RevokedAt != nil {
		return nil, domain.ErrAuthenticationRequired
	}
	if row.UserID == 0 || row.UserDeletedAt != nil || row.Username == "" {
		if revokeErr := r.revokeAuthenticationSessionByID(ctx, row.SessionID, now, sessionRevocationAccountDeleted); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, domain.ErrAuthenticationRequired
	}
	if row.UserStatus != 1 {
		if revokeErr := r.revokeUserAuthenticationSessions(ctx, row.UserID, now, sessionRevocationAccountDisabled); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, domain.ErrAccountDisabled
	}
	if !now.Before(row.ExpiresAt) {
		if revokeErr := r.revokeAuthenticationSessionByID(ctx, row.SessionID, now, sessionRevocationExpired); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, domain.ErrAuthenticationRequired
	}
	if !now.Before(row.IdleExpiresAt) {
		if revokeErr := r.revokeAuthenticationSessionByID(ctx, row.SessionID, now, sessionRevocationIdleTimeout); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, domain.ErrAuthenticationRequired
	}

	if !now.Before(row.LastSeenAt.Add(touchInterval)) {
		nextIdleExpiry := minTime(now.Add(idleTimeout), row.ExpiresAt)
		result := db.Model(&AuthenticationSessionPO{}).
			Where("id = ? AND revoked_at IS NULL AND last_seen_at <= ?", row.SessionID, now.Add(-touchInterval)).
			Updates(map[string]any{
				"last_seen_at":    now,
				"idle_expires_at": nextIdleExpiry,
				"updated_at":      now,
			})
		if result.Error != nil {
			return nil, authenticationPersistenceError("touch", result.Error)
		}
	}

	return &domain.AuthenticationIdentity{
		UserID:    row.UserID,
		Username:  row.Username,
		SessionID: row.SessionID,
	}, nil
}

func (r *repository) revokeAuthenticationSessionByTokenHash(
	ctx context.Context,
	tokenHash string,
	now time.Time,
	reason string,
) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	result := db.Model(&AuthenticationSessionPO{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Updates(sessionRevocationUpdates(now, reason))
	if result.Error != nil {
		return authenticationPersistenceError("revoke credential", result.Error)
	}
	return nil
}

func (r *repository) revokeAuthenticationSessionByID(
	ctx context.Context,
	sessionID string,
	now time.Time,
	reason string,
) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	result := db.Model(&AuthenticationSessionPO{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Updates(sessionRevocationUpdates(now, reason))
	if result.Error != nil {
		return authenticationPersistenceError("revoke", result.Error)
	}
	return nil
}

func (r *repository) revokeUserAuthenticationSessions(
	ctx context.Context,
	userID uint,
	now time.Time,
	reason string,
) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return revokeUserSessionsDB(db, userID, now, reason)
}

func (r *repository) pruneAuthenticationSessions(ctx context.Context, cutoff time.Time, batch int) (int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return 0, err
	}

	var ids []string
	err = db.Model(&AuthenticationSessionPO{}).
		Where(
			"(revoked_at IS NOT NULL AND revoked_at <= ?) OR expires_at <= ? OR idle_expires_at <= ?",
			cutoff,
			cutoff,
			cutoff,
		).
		Order("updated_at ASC").
		Limit(batch).
		Pluck("id", &ids).Error
	if err != nil {
		return 0, authenticationPersistenceError("select prune batch", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	result := db.Where("id IN ?", ids).Delete(&AuthenticationSessionPO{})
	if result.Error != nil {
		return 0, authenticationPersistenceError("prune", result.Error)
	}
	return result.RowsAffected, nil
}

func revokeUserSessionsDB(db *gorm.DB, userID uint, now time.Time, reason string) error {
	if userID == 0 {
		return domain.ErrInvalidInput
	}
	result := db.Model(&AuthenticationSessionPO{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(sessionRevocationUpdates(now, reason))
	if result.Error != nil {
		return authenticationPersistenceError("revoke user sessions", result.Error)
	}
	return nil
}

func sessionRevocationUpdates(now time.Time, reason string) map[string]any {
	return map[string]any{
		"revoked_at":        now,
		"revocation_reason": normalizeSessionRevocationReason(reason),
		"updated_at":        now,
	}
}

func authenticationPersistenceError(operation string, err error) error {
	return fmt.Errorf("%w: authentication session %s: %v", domain.ErrServiceUnavailable, operation, err)
}

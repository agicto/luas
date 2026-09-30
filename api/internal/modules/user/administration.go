package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

const (
	maxUserQueryLength = 100
	userStatusActive   = 1
	userStatusDisabled = 0
)

var _ domain.UserAdministrator = (*service)(nil)

// ListUsers returns one bounded page of accounts, newest first.
func (s *service) ListUsers(
	ctx context.Context,
	filter domain.UserListFilter,
	page int,
	pageSize int,
) ([]*domain.User, int64, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	if len(filter.Query) > maxUserQueryLength || page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	if s.admin == nil {
		return nil, 0, domain.ErrServiceUnavailable
	}
	return s.admin.listUsers(ctx, filter, page, pageSize)
}

// GetUser returns one account or ErrUserNotFound.
func (s *service) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	return user, err
}

// SetUserActive enables or disables an account. Disabling revokes every session in the same
// transaction; repeating a transition is a no-op that returns the current account.
func (s *service) SetUserActive(ctx context.Context, id uint, active bool) (*domain.User, error) {
	status := userStatusDisabled
	if active {
		status = userStatusActive
	}
	if s.admin == nil {
		return nil, domain.ErrServiceUnavailable
	}
	if err := s.admin.setUserStatus(ctx, id, status, time.Now()); err != nil {
		return nil, err
	}
	return s.GetUser(ctx, id)
}

// RevokeUserSessions ends every active session of an account without changing its status.
func (s *service) RevokeUserSessions(ctx context.Context, id uint) error {
	if s.admin == nil {
		return domain.ErrServiceUnavailable
	}
	if _, err := s.GetUser(ctx, id); err != nil {
		return err
	}
	return s.admin.revokeUserSessions(ctx, id, time.Now())
}

// userAdministrationStore is the persistence the administration seam needs beyond UserRepository.
type userAdministrationStore interface {
	listUsers(ctx context.Context, filter domain.UserListFilter, page, pageSize int) ([]*domain.User, int64, error)
	setUserStatus(ctx context.Context, id uint, status int, now time.Time) error
	revokeUserSessions(ctx context.Context, id uint, now time.Time) error
}

func (r *repository) listUsers(
	ctx context.Context,
	filter domain.UserListFilter,
	page int,
	pageSize int,
) ([]*domain.User, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	query := db.Model(&UserPO{})
	if filter.Query != "" {
		pattern := "%" + escapeLike(strings.ToLower(filter.Query)) + "%"
		query = query.Where("(LOWER(users.username) LIKE ? ESCAPE '\\' OR LOWER(users.email) LIKE ? ESCAPE '\\')", pattern, pattern)
	}
	if filter.Active != nil {
		status := userStatusDisabled
		if *filter.Active {
			status = userStatusActive
		}
		query = query.Where("users.status = ?", status)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []UserPO
	if err := query.Order("users.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	users := make([]*domain.User, len(rows))
	for index := range rows {
		users[index] = rows[index].toDomain()
	}
	return users, total, nil
}

func (r *repository) setUserStatus(ctx context.Context, id uint, status int, now time.Time) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&UserPO{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrUserNotFound
		}
		if status == userStatusDisabled {
			return revokeUserSessionsDB(tx, id, now, sessionRevocationAccountDisabled)
		}
		return nil
	})
}

func (r *repository) revokeUserSessions(ctx context.Context, id uint, now time.Time) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return revokeUserSessionsDB(db, id, now, sessionRevocationOperator)
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

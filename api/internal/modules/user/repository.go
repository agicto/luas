package user

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	infradatabase "github.com/zgiai/luas/api/internal/infra/database"
)

// repository implements domain.UserRepository
// It uses UserPO internally for database operations and converts to domain.User
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance that implements domain.UserRepository
func NewRepository(db *gorm.DB) *repository {
	return &repository{
		db: db,
	}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return infradatabase.ResolveContextDB(ctx, r.db), nil
}

// Create adds a new user
func (r *repository) Create(ctx context.Context, user *domain.User) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	po := newUserPO(user)
	if err := db.Create(po).Error; err != nil {
		return err
	}
	// Update the domain user with generated ID
	user.ID = po.ID
	user.CreatedAt = po.CreatedAt
	user.UpdatedAt = po.UpdatedAt
	return nil
}

// Update modifies an existing user
func (r *repository) Update(ctx context.Context, user *domain.User) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	po := newUserPO(user)
	if err := db.Save(po).Error; err != nil {
		return err
	}
	user.UpdatedAt = po.UpdatedAt
	return nil
}

// Delete removes a user by ID
func (r *repository) Delete(ctx context.Context, id uint) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return db.Delete(&UserPO{}, id).Error
}

// DeleteAccount locks the undeleted user row, runs starter guards and cleanup, and soft-deletes
// in one transaction shared through the callback context.
func (r *repository) DeleteAccount(ctx context.Context, id uint, check func(context.Context) error) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	if id == 0 {
		return domain.ErrInvalidInput
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var po UserPO
		query := tx
		if tx.Dialector != nil && tx.Name() != "sqlite" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if findErr := query.First(&po, id).Error; findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return findErr
		}

		transactionContext := infradatabase.ContextWithTransaction(ctx, tx)
		if check != nil {
			if checkErr := check(transactionContext); checkErr != nil {
				return checkErr
			}
		}
		if revokeErr := revokeUserSessionsDB(
			tx.WithContext(transactionContext),
			id,
			time.Now().UTC(),
			sessionRevocationAccountDeleted,
		); revokeErr != nil {
			return revokeErr
		}

		result := tx.WithContext(transactionContext).Delete(&po)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrUserNotFound
		}
		return nil
	})
}

// UpdatePasswordAndRevokeSessions changes the password and invalidates every active session atomically.
func (r *repository) UpdatePasswordAndRevokeSessions(
	ctx context.Context,
	userID uint,
	passwordHash string,
	now time.Time,
) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&UserPO{}).
			Where("id = ?", userID).
			Updates(map[string]any{
				"password":   passwordHash,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrUserNotFound
		}
		return revokeUserSessionsDB(tx, userID, now, sessionRevocationPasswordChange)
	})
}

// FindByID retrieves a user by ID
func (r *repository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var po UserPO
	if err := db.First(&po, id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// FindAll retrieves users with pagination
func (r *repository) FindAll(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	var poList []*UserPO
	var total int64

	offset := (page - 1) * pageSize
	if err := db.Model(&UserPO{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&poList).Error; err != nil {
		return nil, 0, err
	}

	return toDomainList(poList), total, nil
}

// FindByUsername retrieves a user by username
func (r *repository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var po UserPO
	if err := db.Where("username = ?", username).First(&po).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// FindByLoginIdentifier resolves username or email in one query. Username
// keeps precedence to preserve the starter's historical login behavior when
// identifiers collide across fields.
func (r *repository) FindByLoginIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var po UserPO
	err = db.
		Where("username = ? OR email = ?", identifier, identifier).
		Order(clause.Expr{
			SQL:  "CASE WHEN username = ? THEN 0 ELSE 1 END",
			Vars: []any{identifier},
		}).
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// FindByEmail retrieves a user by email
func (r *repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var po UserPO
	if err := db.Where("email = ?", email).First(&po).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// StorePasswordResetToken stores a one-time password reset token hash.
func (r *repository) StorePasswordResetToken(ctx context.Context, userID uint, tokenHash string, expiresAt time.Time) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND used_at IS NULL", userID).Delete(&PasswordResetTokenPO{}).Error; err != nil {
			return err
		}

		token := &PasswordResetTokenPO{
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: expiresAt,
		}
		return tx.Create(token).Error
	})
}

// ResetPasswordWithToken validates a reset token, updates the password, and consumes the token atomically.
func (r *repository) ResetPasswordWithToken(ctx context.Context, tokenHash string, passwordHash string, now time.Time) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var token PasswordResetTokenPO
		if err := tx.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrPasswordResetTokenInvalid
			}
			return err
		}

		if token.UsedAt != nil {
			return domain.ErrPasswordResetTokenInvalid
		}
		if now.After(token.ExpiresAt) {
			return domain.ErrPasswordResetTokenExpired
		}

		if err := tx.Model(&UserPO{}).Where("id = ?", token.UserID).Update("password", passwordHash).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return err
		}
		if err := revokeUserSessionsDB(tx, token.UserID, now, sessionRevocationPasswordReset); err != nil {
			return err
		}

		result := tx.Model(&PasswordResetTokenPO{}).
			Where("id = ? AND used_at IS NULL", token.ID).
			Updates(map[string]any{
				"used_at":    now,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrPasswordResetTokenInvalid
		}

		return nil
	})
}

package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/email"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/jwt"
)

type fakeRepo struct {
	createFn         func(context.Context, *domain.User) error
	updateFn         func(context.Context, *domain.User) error
	deleteFn         func(context.Context, uint) error
	findByIDFn       func(context.Context, uint) (*domain.User, error)
	findByEmailFn    func(context.Context, string) (*domain.User, error)
	findByUsernameFn func(context.Context, string) (*domain.User, error)
	findByLoginFn    func(context.Context, string) (*domain.User, error)
	findAllFn        func(context.Context, int, int) ([]*domain.User, int64, error)
	storeResetFn     func(context.Context, uint, string, time.Time) error
	resetByTokenFn   func(context.Context, string, string, time.Time) error
}

type fakeUserMailer struct {
	passwordResetErr error
}

func (m *fakeUserMailer) SendPasswordResetEmail(string, string) error {
	return m.passwordResetErr
}

func (m *fakeUserMailer) SendWelcomeEmail(string, string) error {
	return nil
}

func (r *fakeRepo) Create(ctx context.Context, user *domain.User) error {
	if r.createFn != nil {
		return r.createFn(ctx, user)
	}
	return nil
}

func (r *fakeRepo) Update(ctx context.Context, user *domain.User) error {
	if r.updateFn != nil {
		return r.updateFn(ctx, user)
	}
	return nil
}

func (r *fakeRepo) Delete(ctx context.Context, id uint) error {
	if r.deleteFn != nil {
		return r.deleteFn(ctx, id)
	}
	return nil
}

func (r *fakeRepo) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	if r.findByIDFn != nil {
		return r.findByIDFn(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if r.findByEmailFn != nil {
		return r.findByEmailFn(ctx, email)
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	if r.findByUsernameFn != nil {
		return r.findByUsernameFn(ctx, username)
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) FindByLoginIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	if r.findByLoginFn != nil {
		return r.findByLoginFn(ctx, identifier)
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) FindAll(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	if r.findAllFn != nil {
		return r.findAllFn(ctx, page, pageSize)
	}
	return nil, 0, nil
}

func (r *fakeRepo) StorePasswordResetToken(ctx context.Context, userID uint, tokenHash string, expiresAt time.Time) error {
	if r.storeResetFn != nil {
		return r.storeResetFn(ctx, userID, tokenHash, expiresAt)
	}
	return nil
}

func (r *fakeRepo) ResetPasswordWithToken(ctx context.Context, tokenHash string, passwordHash string, now time.Time) error {
	if r.resetByTokenFn != nil {
		return r.resetByTokenFn(ctx, tokenHash, passwordHash, now)
	}
	return nil
}

func newTestService(repo userRepository) *service {
	cfg := &config.Config{}
	return NewService(repo, repo.(passwordResetStore), jwt.NewTestService(), events.NewEventBus(), email.NewService(cfg))
}

func mustHashTestPassword(t *testing.T) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func TestServiceRegisterSuccess(t *testing.T) {
	var created *domain.User

	svc := newTestService(&fakeRepo{
		findByEmailFn: func(context.Context, string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
		createFn: func(_ context.Context, user *domain.User) error {
			created = user
			return nil
		},
	})

	user, err := svc.Register(context.Background(), &UserRegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
		Nickname: "Alice",
		Phone:    "123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, created)
	assert.Equal(t, "alice", created.Username)
	assert.Equal(t, "alice@example.com", created.Email)
	assert.Equal(t, "Alice", created.Nickname)
	assert.NotEqual(t, "password123", created.Password)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.Password), []byte("password123")))
}

func TestServiceRegisterReturnsDomainErrorWhenEmailExists(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByUsernameFn: func(context.Context, string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
		findByEmailFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: "alice@example.com"}, nil
		},
	})

	user, err := svc.Register(context.Background(), &UserRegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})

	assert.Nil(t, user)
	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}

func TestServiceRegisterReturnsDomainErrorWhenUsernameExists(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByUsernameFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: 1, Username: "alice"}, nil
		},
	})

	user, err := svc.Register(context.Background(), &UserRegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})

	assert.Nil(t, user)
	assert.ErrorIs(t, err, domain.ErrUsernameAlreadyExists)
}

func TestServiceRegisterFailsFastOnLookupError(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByUsernameFn: func(context.Context, string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
		findByEmailFn: func(context.Context, string) (*domain.User, error) {
			return nil, errors.New("db unavailable")
		},
	})

	user, err := svc.Register(context.Background(), &UserRegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})

	assert.Nil(t, user)
	assert.EqualError(t, err, "failed to check existing email: db unavailable")
}

func TestServiceLoginFindsEmailIdentifierAndUpdatesLastLogin(t *testing.T) {
	var updated *domain.User
	hashedPassword := mustHashTestPassword(t)

	svc := newTestService(&fakeRepo{
		findByLoginFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{
				ID:       42,
				Username: "alice",
				Email:    "alice@example.com",
				Password: hashedPassword,
				Status:   1,
			}, nil
		},
		updateFn: func(_ context.Context, user *domain.User) error {
			updated = user
			return nil
		},
	})

	resp, err := svc.Login(context.Background(), &UserLoginRequest{
		Username: "alice@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, uint(42), resp.User.ID)
	assert.NotNil(t, updated)
	assert.NotNil(t, updated.LastLogin)
}

func TestServiceLoginReturnsInvalidCredentialsOnWrongPassword(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByLoginFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{
				ID:       1,
				Username: "alice",
				Email:    "alice@example.com",
				Password: mustHashTestPassword(t),
				Status:   1,
			}, nil
		},
	})

	resp, err := svc.Login(context.Background(), &UserLoginRequest{
		Username: "alice",
		Password: "wrong-password",
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestServiceLoginFailsFastOnIdentifierLookupError(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByLoginFn: func(context.Context, string) (*domain.User, error) {
			return nil, errors.New("db unavailable")
		},
	})

	resp, err := svc.Login(context.Background(), &UserLoginRequest{
		Username: "alice",
		Password: "password123",
	})

	assert.Nil(t, resp)
	assert.EqualError(t, err, "failed to lookup login identifier: db unavailable")
}

func TestServiceLoginUsesDummyHashForUnknownIdentifier(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByLoginFn: func(context.Context, string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
	})

	var comparedHash string
	svc.verifyPassword = func(hash, _ []byte) error {
		comparedHash = string(hash)
		return bcrypt.ErrMismatchedHashAndPassword
	}

	resp, err := svc.Login(context.Background(), &UserLoginRequest{
		Username: "missing@example.com",
		Password: "password123",
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Equal(t, dummyPasswordHash, comparedHash)
}

func TestServiceLoginDoesNotRevealDisabledAccount(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByLoginFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{
				ID:       1,
				Username: "disabled-user",
				Password: mustHashTestPassword(t),
				Status:   0,
			}, nil
		},
	})

	resp, err := svc.Login(context.Background(), &UserLoginRequest{
		Username: "disabled-user",
		Password: "password123",
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.NotErrorIs(t, err, domain.ErrAccountDisabled)
}

func TestServiceChangePasswordReturnsInvalidCredentialsOnWrongOldPassword(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByIDFn: func(context.Context, uint) (*domain.User, error) {
			return &domain.User{
				ID:       7,
				Password: mustHashTestPassword(t),
				Status:   1,
			}, nil
		},
	})

	err := svc.ChangePassword(context.Background(), 7, &UserChangePasswordRequest{
		OldPassword: "wrong-password",
		NewPassword: "new-password-123",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestServiceRequestPasswordResetDoesNotEnumerateUnknownEmail(t *testing.T) {
	svc := newTestService(&fakeRepo{
		findByEmailFn: func(context.Context, string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
	})

	err := svc.RequestPasswordReset(context.Background(), &UserPasswordResetRequest{
		Email: "missing@example.com",
	})

	assert.NoError(t, err)
}

func TestServiceRequestPasswordResetStoresHashedToken(t *testing.T) {
	var storedUserID uint
	var storedTokenHash string
	var storedExpiresAt time.Time

	svc := newTestService(&fakeRepo{
		findByEmailFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: 42, Email: "alice@example.com"}, nil
		},
		storeResetFn: func(_ context.Context, userID uint, tokenHash string, expiresAt time.Time) error {
			storedUserID = userID
			storedTokenHash = tokenHash
			storedExpiresAt = expiresAt
			return nil
		},
	})

	err := svc.RequestPasswordReset(context.Background(), &UserPasswordResetRequest{
		Email: "alice@example.com",
	})

	assert.NoError(t, err)
	assert.Equal(t, uint(42), storedUserID)
	assert.Len(t, storedTokenHash, 64)
	assert.WithinDuration(t, time.Now().Add(30*time.Minute), storedExpiresAt, 5*time.Second)
}

func TestServiceRequestPasswordResetDoesNotExposeAccountSpecificProcessingFailure(t *testing.T) {
	tests := []struct {
		name     string
		storeErr error
		mailErr  error
	}{
		{name: "token store failure", storeErr: errors.New("store unavailable")},
		{name: "mail delivery failure", mailErr: errors.New("mail unavailable")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				findByEmailFn: func(context.Context, string) (*domain.User, error) {
					return &domain.User{ID: 42, Email: "alice@example.com"}, nil
				},
				storeResetFn: func(context.Context, uint, string, time.Time) error {
					return tt.storeErr
				},
			}
			svc := NewService(
				repo,
				repo,
				jwt.NewTestService(),
				events.NewEventBus(),
				&fakeUserMailer{passwordResetErr: tt.mailErr},
			)

			err := svc.RequestPasswordReset(context.Background(), &UserPasswordResetRequest{
				Email: "alice@example.com",
			})

			assert.NoError(t, err)
		})
	}
}

func TestServiceConfirmPasswordResetHashesNewPassword(t *testing.T) {
	var seenTokenHash string
	var seenPasswordHash string

	svc := newTestService(&fakeRepo{
		resetByTokenFn: func(_ context.Context, tokenHash string, passwordHash string, _ time.Time) error {
			seenTokenHash = tokenHash
			seenPasswordHash = passwordHash
			return nil
		},
	})

	err := svc.ConfirmPasswordReset(context.Background(), &UserPasswordResetConfirmRequest{
		Token:       "reset-token-value",
		NewPassword: "new-password-123",
	})

	assert.NoError(t, err)
	assert.Len(t, seenTokenHash, 64)
	assert.NotEqual(t, "new-password-123", seenPasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(seenPasswordHash), []byte("new-password-123")))
}

func TestServiceConfirmPasswordResetReturnsDomainErrorFromResetStore(t *testing.T) {
	svc := newTestService(&fakeRepo{
		resetByTokenFn: func(context.Context, string, string, time.Time) error {
			return domain.ErrPasswordResetTokenExpired
		},
	})

	err := svc.ConfirmPasswordReset(context.Background(), &UserPasswordResetConfirmRequest{
		Token:       "expired-token",
		NewPassword: "new-password-123",
	})

	assert.ErrorIs(t, err, domain.ErrPasswordResetTokenExpired)
}

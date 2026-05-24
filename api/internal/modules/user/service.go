package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/zgiai/zgo/internal/capabilities/crypto"
	"github.com/zgiai/zgo/internal/capabilities/idgen"
	"github.com/zgiai/zgo/internal/domain"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/jwt"
	auditstarter "github.com/zgiai/zgo/internal/modules/audit"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService defines the authentication and public account flows.
type AuthService interface {
	Register(ctx context.Context, req *UserRegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req *UserLoginRequest) (*UserLoginResponse, error)
	RequestPasswordReset(ctx context.Context, req *UserPasswordResetRequest) error
	ConfirmPasswordReset(ctx context.Context, req *UserPasswordResetConfirmRequest) error
}

// ProfileService defines authenticated profile management flows.
type ProfileService interface {
	GetProfile(ctx context.Context, userID uint) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID uint, req *UserUpdateRequest) (*domain.User, error)
	ChangePassword(ctx context.Context, userID uint, req *UserChangePasswordRequest) error
	DeleteAccount(ctx context.Context, userID uint) error
}

// UserQueryService defines read-only user lookup flows.
type UserQueryService interface {
	GetByID(ctx context.Context, id uint) (*domain.User, error)
	List(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error)
}

// Service is the full user service surface kept for compatibility.
// Narrow seams should prefer AuthService, ProfileService, or UserQueryService.
type Service interface {
	AuthService
	ProfileService
	UserQueryService
}

// UserMailer captures the email seam used by the user starter.
// Email is a true external dependency, so this seam is worth naming explicitly.
type UserMailer interface {
	SendPasswordResetEmail(to string, resetToken string) error
	SendWelcomeEmail(to string, username string) error
}

type passwordResetStore interface {
	StorePasswordResetToken(ctx context.Context, userID uint, tokenHash string, expiresAt time.Time) error
	ResetPasswordWithToken(ctx context.Context, tokenHash string, passwordHash string, now time.Time) error
}

// service implements the user service interfaces.
type service struct {
	repo           domain.UserRepository
	passwordResets passwordResetStore
	jwtService     *jwt.Service
	eventBus       *events.EventBus
	mailer         UserMailer
}

var (
	_ Service          = (*service)(nil)
	_ AuthService      = (*service)(nil)
	_ ProfileService   = (*service)(nil)
	_ UserQueryService = (*service)(nil)
)

// NewService creates a new service instance
func NewService(
	repo domain.UserRepository,
	passwordResets passwordResetStore,
	jwtService *jwt.Service,
	eventBus *events.EventBus,
	mailer UserMailer,
) *service {
	return &service{
		repo:           repo,
		passwordResets: passwordResets,
		jwtService:     jwtService,
		eventBus:       eventBus,
		mailer:         mailer,
	}
}

// ============================================================================
// Authentication
// ============================================================================

// Register handles user registration
func (s *service) Register(ctx context.Context, req *UserRegisterRequest) (*domain.User, error) {
	existingByUsername, err := s.repo.FindByUsername(ctx, req.Username)
	if err == nil && existingByUsername != nil {
		return nil, domain.ErrUsernameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing username: %w", err)
	}

	// Check if email already exists
	existingByEmail, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil && existingByEmail != nil {
		return nil, domain.ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Status:   1,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Publish UserCreated event (fully decoupled side effects)
	s.eventBus.PublishAsync(ctx, domain.NewUserCreatedEvent(user))

	return user, nil
}

// Login handles user login
func (s *service) Login(ctx context.Context, req *UserLoginRequest) (*UserLoginResponse, error) {
	// Try username first, then email
	user, err := s.repo.FindByUsername(ctx, req.Username)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to lookup username: %w", err)
		}

		user, err = s.repo.FindByEmail(ctx, req.Username)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, domain.ErrInvalidCredentials
			}
			return nil, fmt.Errorf("failed to lookup email: %w", err)
		}
	}

	if !user.IsActive() {
		return nil, domain.ErrAccountDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := s.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Update last login. Auth has already succeeded; a write failure
	// here shouldn't fail the login, but it shouldn't be invisible either.
	now := time.Now()
	user.LastLogin = &now
	if err := s.repo.Update(ctx, user); err != nil {
		log.Printf("user: failed to update LastLogin for user %d: %v", user.ID, err)
	}

	return &UserLoginResponse{
		AccessToken: token,
		User:        user, // Domain直接输出
	}, nil
}

// ============================================================================
// Profile (Authenticated User)
// ============================================================================

// GetProfile retrieves user profile
func (s *service) GetProfile(ctx context.Context, userID uint) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile updates user profile
func (s *service) UpdateProfile(ctx context.Context, userID uint, req *UserUpdateRequest) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	changes := make(map[string]domain.AuditValueChange)

	// Only update non-empty fields
	if req.Nickname != "" && req.Nickname != user.Nickname {
		changes["nickname"] = domain.AuditValueChange{Before: user.Nickname, After: req.Nickname}
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" && req.Avatar != user.Avatar {
		changes["avatar"] = domain.AuditValueChange{Before: user.Avatar, After: req.Avatar}
		user.Avatar = req.Avatar
	}
	if req.Phone != "" && req.Phone != user.Phone {
		changes["phone"] = domain.AuditValueChange{Before: user.Phone, After: req.Phone}
		user.Phone = req.Phone
	}
	if req.Bio != "" && req.Bio != user.Bio {
		changes["bio"] = domain.AuditValueChange{Before: user.Bio, After: req.Bio}
		user.Bio = req.Bio
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	if len(changes) > 0 {
		auditstarter.RecordChange(ctx, auditstarter.Change{
			TargetType: "user",
			TargetID:   strconv.FormatUint(uint64(userID), 10),
			Result:     domain.AuditResultSuccess,
			Changes:    changes,
		})
	}

	return user, nil
}

// ChangePassword changes user password
func (s *service) ChangePassword(ctx context.Context, userID uint, req *UserChangePasswordRequest) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = string(hashedPassword)
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		TargetType: "user",
		TargetID:   strconv.FormatUint(uint64(userID), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"password": {Before: "[redacted]", After: "[redacted]"},
		},
		Metadata: map[string]any{
			"credential": "password",
		},
	})
	return nil
}

// DeleteAccount deletes user account
func (s *service) DeleteAccount(ctx context.Context, userID uint) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		return err
	}

	auditstarter.RecordChange(ctx, auditstarter.Change{
		TargetType: "user",
		TargetID:   strconv.FormatUint(uint64(userID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"operation": "delete_account",
		},
	})
	return nil
}

// ============================================================================
// Public
// ============================================================================

// RequestPasswordReset creates a one-time reset token and emails it to the user.
func (s *service) RequestPasswordReset(ctx context.Context, req *UserPasswordResetRequest) error {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to lookup reset user: %w", err)
	}

	secret, err := crypto.GenerateKeyHex(24)
	if err != nil {
		return fmt.Errorf("failed to generate password reset token: %w", err)
	}
	resetToken := "zrp_" + strings.ToLower(idgen.ShortID()) + "." + secret
	expiresAt := time.Now().Add(30 * time.Minute)

	if err := s.passwordResets.StorePasswordResetToken(ctx, user.ID, crypto.SHA256Hex(resetToken), expiresAt); err != nil {
		return fmt.Errorf("failed to store password reset token: %w", err)
	}

	if err := s.mailer.SendPasswordResetEmail(user.Email, resetToken); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	return nil
}

// ConfirmPasswordReset consumes a one-time token and writes the new password hash.
func (s *service) ConfirmPasswordReset(ctx context.Context, req *UserPasswordResetConfirmRequest) error {
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return domain.ErrInvalidInput
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.passwordResets.ResetPasswordWithToken(ctx, crypto.SHA256Hex(token), string(hashedPassword), time.Now()); err != nil {
		return err
	}
	return nil
}

// ============================================================================
// Admin/Query
// ============================================================================

// GetByID retrieves a user by ID
func (s *service) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	return s.repo.FindByID(ctx, id)
}

// List retrieves a paginated list of users
func (s *service) List(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	return s.repo.FindAll(ctx, page, pageSize)
}

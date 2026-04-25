package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zgiai/zgo/internal/domain"
	"github.com/zgiai/zgo/internal/infra/email"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/jwt"
	"github.com/zgiai/zgo/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService defines the authentication and public account flows.
type AuthService interface {
	Register(ctx context.Context, req *UserRegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req *UserLoginRequest) (*UserLoginResponse, error)
	ResetPassword(ctx context.Context, req *UserPasswordResetRequest) error
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

// service implements the user service interfaces.
type service struct {
	repo       domain.UserRepository
	jwtService *jwt.Service
	eventBus   *events.EventBus
}

var (
	_ Service          = (*service)(nil)
	_ AuthService      = (*service)(nil)
	_ ProfileService   = (*service)(nil)
	_ UserQueryService = (*service)(nil)
)

// NewService creates a new service instance
func NewService(repo domain.UserRepository, jwtService *jwt.Service, eventBus *events.EventBus) *service {
	return &service{
		repo:       repo,
		jwtService: jwtService,
		eventBus:   eventBus,
	}
}

// ============================================================================
// Authentication
// ============================================================================

// Register handles user registration
func (s *service) Register(ctx context.Context, req *UserRegisterRequest) (*domain.User, error) {
	// Check if email already exists
	exists, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil && exists != nil {
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
		user, err = s.repo.FindByEmail(ctx, req.Username)
		if err != nil {
			return nil, domain.ErrInvalidCredentials
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

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	_ = s.repo.Update(ctx, user)

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

	// Only update non-empty fields
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
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
	return s.repo.Update(ctx, user)
}

// DeleteAccount deletes user account
func (s *service) DeleteAccount(ctx context.Context, userID uint) error {
	return s.repo.Delete(ctx, userID)
}

// ============================================================================
// Public
// ============================================================================

// ResetPassword resets user password via email
func (s *service) ResetPassword(ctx context.Context, req *UserPasswordResetRequest) error {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to lookup reset user: %w", err)
	}

	newPassword := utils.GenerateRandomString(12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = string(hashedPassword)
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return email.SendPasswordResetEmail(user.Email, newPassword)
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

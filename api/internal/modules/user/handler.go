package user

import (
	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/domain"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/jwt"
	"github.com/zgiai/zgo/pkg/handler"
	"github.com/zgiai/zgo/pkg/pagination"
	"github.com/zgiai/zgo/pkg/response"
)

// Handler handles user-related HTTP requests and exposes route, middleware, and event capabilities.
type Handler struct {
	auth       AuthService
	profile    ProfileService
	query      UserQueryService
	jwtService *jwt.Service
	mailer     UserMailer
}

var (
	_ contracts.Module           = (*Handler)(nil)
	_ contracts.RouteModule      = (*Handler)(nil)
	_ contracts.MiddlewareModule = (*Handler)(nil)
	_ contracts.EventModule      = (*Handler)(nil)
)

// NewHandler creates a new Handler instance.
func NewHandler(auth AuthService, profile ProfileService, query UserQueryService, jwtService *jwt.Service, mailer UserMailer) *Handler {
	return &Handler{
		auth:       auth,
		profile:    profile,
		query:      query,
		jwtService: jwtService,
		mailer:     mailer,
	}
}

// Name returns the module name
func (h *Handler) Name() string {
	return "user"
}

// RegisterEvents registers user module event listeners
func (h *Handler) RegisterEvents(bus *events.EventBus) {
	bus.Subscribe(domain.EventUserCreated, h.handleUserCreated, events.WithAsync())
}

// ============================================================================
// Authentication
// ============================================================================

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req UserRegisterRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	user, err := h.auth.Register(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Registration failed", err)
		return
	}

	c.Set("userID", user.ID)
	response.Created(c, user)
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req UserLoginRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Login failed", err)
		return
	}

	if resp.User != nil {
		c.Set("userID", resp.User.ID)
	}
	response.Success(c, resp)
}

// ============================================================================
// Profile (Authenticated User)
// ============================================================================

// GetProfile gets current user's profile
func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	user, err := h.profile.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, "Failed to get profile", err)
		return
	}

	response.Success(c, user)
}

// UpdateProfile updates current user's profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	var req UserUpdateRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	user, err := h.profile.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		response.HandleError(c, "Failed to update profile", err)
		return
	}

	response.Success(c, user)
}

// ChangePassword changes current user's password
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	var req UserChangePasswordRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	if err := h.profile.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		response.HandleError(c, "Failed to change password", err)
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

// DeleteAccount deletes current user's account
func (h *Handler) DeleteAccount(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	if err := h.profile.DeleteAccount(c.Request.Context(), userID); err != nil {
		response.HandleError(c, "Failed to delete account", err)
		return
	}

	response.NoContent(c)
}

// ============================================================================
// Public
// ============================================================================

// RequestPasswordReset initiates password reset.
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req UserPasswordResetRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	if err := h.auth.RequestPasswordReset(c.Request.Context(), &req); err != nil {
		response.HandleError(c, "Failed to reset password", err)
		return
	}

	response.Success(c, gin.H{"message": "If the account exists, password reset instructions have been sent"})
}

// ConfirmPasswordReset consumes a reset token and writes a new password.
func (h *Handler) ConfirmPasswordReset(c *gin.Context) {
	var req UserPasswordResetConfirmRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	if err := h.auth.ConfirmPasswordReset(c.Request.Context(), &req); err != nil {
		response.HandleError(c, "Failed to confirm password reset", err)
		return
	}

	response.Success(c, gin.H{"message": "Password has been reset successfully"})
}

// ============================================================================
// Admin/Query
// ============================================================================

// Get gets user by ID
func (h *Handler) Get(c *gin.Context) {
	id, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	user, err := h.query.GetByID(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, "User not found", err)
		return
	}

	response.Success(c, user)
}

// List gets paginated user list
func (h *Handler) List(c *gin.Context) {
	req := pagination.FromContext(c)

	users, total, err := h.query.List(c.Request.Context(), req.GetPage(), req.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to get user list", err)
		return
	}

	paginator := pagination.NewPaginator(users, total, req.GetPage(), req.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)

	response.Success(c, paginator)
}

// GetUserInfo gets detailed user info by ID (alias for Get)
func (h *Handler) GetUserInfo(c *gin.Context) {
	h.Get(c)
}

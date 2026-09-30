package operator

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// managedUserResponse is the operator view of an account: identity, status, and activity only.
type managedUserResponse struct {
	ID         uint       `json:"id"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	Nickname   string     `json:"nickname"`
	Status     string     `json:"status"`
	IsOperator bool       `json:"is_operator"`
	CreatedAt  time.Time  `json:"created_at"`
	LastLogin  *time.Time `json:"last_login"`
}

func newManagedUserResponse(user *domain.User, isOperator bool) managedUserResponse {
	status := "disabled"
	if user.IsActive() {
		status = "active"
	}
	return managedUserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Nickname:   user.Nickname,
		Status:     status,
		IsOperator: isOperator,
		CreatedAt:  user.CreatedAt,
		LastLogin:  user.LastLogin,
	}
}

type listUsersQuery struct {
	Query  string `form:"q" binding:"max=100"`
	Status string `form:"status" binding:"omitempty,oneof=active disabled all"`
}

// listUsers returns a page of accounts with each account's operator flag.
func (s *service) listUsers(
	ctx context.Context,
	filter domain.UserListFilter,
	page int,
	perPage int,
) ([]managedUserResponse, int64, error) {
	if s.admin == nil {
		return nil, 0, domain.ErrServiceUnavailable
	}
	users, total, err := s.admin.ListUsers(ctx, filter, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint, len(users))
	for index, user := range users {
		ids[index] = user.ID
	}
	operators, err := s.grants.operatorIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	items := make([]managedUserResponse, len(users))
	for index, user := range users {
		items[index] = newManagedUserResponse(user, operators[user.ID])
	}
	return items, total, nil
}

func (s *service) getUser(ctx context.Context, id uint) (managedUserResponse, error) {
	if s.admin == nil {
		return managedUserResponse{}, domain.ErrServiceUnavailable
	}
	user, err := s.admin.GetUser(ctx, id)
	if err != nil {
		return managedUserResponse{}, err
	}
	isOperator, err := s.grants.isOperator(ctx, id)
	if err != nil {
		return managedUserResponse{}, err
	}
	return newManagedUserResponse(user, isOperator), nil
}

// requireUnprotectedTarget loads the target and rejects accounts that hold an operator grant,
// including the caller's own account. Removing an operator is a CLI decision.
func (s *service) requireUnprotectedTarget(ctx context.Context, id uint) error {
	if s.admin == nil {
		return domain.ErrServiceUnavailable
	}
	if _, err := s.admin.GetUser(ctx, id); err != nil {
		return err
	}
	isOperator, err := s.grants.isOperator(ctx, id)
	if err != nil {
		return err
	}
	if isOperator {
		return domain.ErrOperatorTargetProtected
	}
	return nil
}

func (s *service) setUserActive(ctx context.Context, id uint, active bool) (managedUserResponse, error) {
	if err := s.requireUnprotectedTarget(ctx, id); err != nil {
		return managedUserResponse{}, err
	}
	user, err := s.admin.SetUserActive(ctx, id, active)
	if err != nil {
		return managedUserResponse{}, err
	}
	return newManagedUserResponse(user, false), nil
}

func (s *service) revokeUserSessions(ctx context.Context, id uint) error {
	if err := s.requireUnprotectedTarget(ctx, id); err != nil {
		return err
	}
	return s.admin.RevokeUserSessions(ctx, id)
}

// ListUsers serves GET /v1/operator/users.
func (h *Handler) ListUsers(c *gin.Context) {
	var query listUsersQuery
	if !handler.BindQuery(c, &query) {
		return
	}
	filter := domain.UserListFilter{Query: query.Query}
	switch query.Status {
	case "active":
		active := true
		filter.Active = &active
	case "disabled":
		active := false
		filter.Active = &active
	}
	page := pagination.FromContext(c)
	items, total, err := h.service.listUsers(c.Request.Context(), filter, page.GetPage(), page.GetPerPage())
	if err != nil {
		writeError(c, "List users", err)
		return
	}
	paginator := pagination.NewPaginator(items, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// GetUser serves GET /v1/operator/users/:id.
func (h *Handler) GetUser(c *gin.Context) {
	id, ok := userIDParam(c)
	if !ok {
		return
	}
	user, err := h.service.getUser(c.Request.Context(), id)
	if err != nil {
		writeError(c, "Get user", err)
		return
	}
	response.Success(c, user)
}

// DisableUser serves POST /v1/operator/users/:id/disable.
func (h *Handler) DisableUser(c *gin.Context) {
	h.setUserActive(c, false)
}

// EnableUser serves POST /v1/operator/users/:id/enable.
func (h *Handler) EnableUser(c *gin.Context) {
	h.setUserActive(c, true)
}

func (h *Handler) setUserActive(c *gin.Context, active bool) {
	id, ok := userIDParam(c)
	if !ok {
		return
	}
	user, err := h.service.setUserActive(c.Request.Context(), id, active)
	if err != nil {
		writeError(c, "Change user status", err)
		return
	}
	response.Success(c, user)
}

// RevokeUserSessions serves POST /v1/operator/users/:id/sessions/revoke.
func (h *Handler) RevokeUserSessions(c *gin.Context) {
	id, ok := userIDParam(c)
	if !ok {
		return
	}
	if err := h.service.revokeUserSessions(c.Request.Context(), id); err != nil {
		writeError(c, "Revoke user sessions", err)
		return
	}
	response.NoContent(c)
}

func userIDParam(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid user identifier")
		return 0, false
	}
	return uint(value), true
}

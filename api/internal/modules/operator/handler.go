package operator

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/modules/notification"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/setting"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/modules/webhook"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/response"
)

const (
	contextCredential = "operator_session_credential"
	contextOperator   = "operator_account"
)

// Surfaces are the operator-facing handlers owned by other optional starters. Each owning starter
// keeps its behavior and data; this starter owns authorization and mounts a surface only when its
// starter is selected.
type Surfaces struct {
	Settings      *setting.Handler
	Organizations *organization.OperatorHandler
	Webhooks      *webhook.OperatorHandler
	Notifications *notification.OperatorHandler
}

// Handler serves the operator browser session and operator routes.
type Handler struct {
	service       *service
	session       *browserSession
	guard         *user.AuthAbuseGuard
	settings      *setting.Handler
	organizations *organization.OperatorHandler
	webhooks      *webhook.OperatorHandler
	notifications *notification.OperatorHandler
	cfg           *config.Config
}

// NewHandler creates the operator HTTP handler and keeps only the surfaces of selected starters.
func NewHandler(
	service *service,
	guard *user.AuthAbuseGuard,
	cfg *config.Config,
	surfaces Surfaces,
) *Handler {
	handler := &Handler{service: service, session: newBrowserSession(cfg), guard: guard, cfg: cfg}
	if cfg == nil {
		return handler
	}
	if cfg.Starters.Selected(config.StarterSetting) {
		handler.settings = surfaces.Settings
	}
	if cfg.Starters.Selected(config.StarterOrganization) {
		handler.organizations = surfaces.Organizations
	}
	// Webhook routes nest under an organization, so they need the organization surface as well.
	if cfg.Starters.Selected(config.StarterWebhook) && handler.organizations != nil {
		handler.webhooks = surfaces.Webhooks
	}
	if cfg.Starters.Selected(config.StarterNotification) {
		handler.notifications = surfaces.Notifications
	}
	return handler
}

// Name returns the module name.
func (h *Handler) Name() string {
	return config.StarterOperator
}

type signInRequest struct {
	Identifier string `json:"identifier" binding:"required,max=100"`
	Password   string `json:"password" binding:"required,max=128"`
}

type operatorResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

type sessionResponse struct {
	Operator  operatorResponse `json:"operator"`
	CSRFToken string           `json:"csrf_token"`
}

func newSessionResponse(account *domain.User, credential string) sessionResponse {
	return sessionResponse{
		Operator: operatorResponse{
			ID:       account.ID,
			Username: account.Username,
			Email:    account.Email,
			Nickname: account.Nickname,
		},
		CSRFToken: csrfToken(credential),
	}
}

// SignIn verifies credentials for a current operator and stores the session in an HttpOnly cookie.
func (h *Handler) SignIn(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if !h.session.originAllowed(c) {
		response.AbortWithCode(c, http.StatusForbidden, domain.CodeOperatorOriginRejected, "Origin is not allowed")
		return
	}
	var req signInRequest
	if !handler.BindJSON(c, &req) {
		return
	}
	if h.guard != nil && !h.guard.AllowLoginSubject(c, req.Identifier) {
		return
	}

	issued, err := h.service.SignIn(c.Request.Context(), req.Identifier, req.Password)
	if err != nil {
		writeError(c, "Operator sign-in failed", err)
		return
	}
	h.session.store(c, issued.Credential, issued.ExpiresAt)
	c.Set("userID", issued.User.ID)
	response.Success(c, newSessionResponse(issued.User, issued.Credential))
}

// Current returns the signed-in operator and the CSRF token for unsafe requests.
func (h *Handler) Current(c *gin.Context) {
	account, credential, ok := currentOperator(c)
	if !ok {
		response.AbortWithCode(c, http.StatusUnauthorized, response.ErrorCodeUnauthorized, "Authentication required")
		return
	}
	response.Success(c, newSessionResponse(account, credential))
}

// SignOut revokes the presented session and expires the cookie. It succeeds when no session exists.
func (h *Handler) SignOut(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	credential, ok := h.session.credential(c)
	if !ok {
		h.session.clear(c)
		response.NoContent(c)
		return
	}
	if !h.session.originAllowed(c) {
		response.AbortWithCode(c, http.StatusForbidden, domain.CodeOperatorOriginRejected, "Origin is not allowed")
		return
	}
	if !csrfValid(credential, c.GetHeader(csrfHeader)) {
		response.AbortWithCode(c, http.StatusForbidden, domain.CodeOperatorCSRFRejected, "CSRF token is missing or invalid")
		return
	}
	// Attribute the audit record to the operator while the session is still valid; an already expired
	// or revoked session still signs out.
	if account, err := h.service.Authenticate(c.Request.Context(), credential); err == nil {
		c.Set("userID", account.ID)
	}
	if err := h.service.SignOut(c.Request.Context(), credential); err != nil {
		writeError(c, "Operator sign-out failed", err)
		return
	}
	h.session.clear(c)
	response.NoContent(c)
}

// requireOperator authenticates the cookie session, requires a current grant, and for unsafe methods
// requires an allowed Origin and a valid CSRF token.
func (h *Handler) requireOperator(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	credential, ok := h.session.credential(c)
	if !ok {
		response.AbortWithCode(c, http.StatusUnauthorized, response.ErrorCodeUnauthorized, "Authentication required")
		return
	}

	account, err := h.service.Authenticate(c.Request.Context(), credential)
	if err != nil {
		writeError(c, "Operator authentication failed", err)
		return
	}

	if unsafeMethod(c.Request.Method) {
		if !h.session.originAllowed(c) {
			response.AbortWithCode(c, http.StatusForbidden, domain.CodeOperatorOriginRejected, "Origin is not allowed")
			return
		}
		if !csrfValid(credential, c.GetHeader(csrfHeader)) {
			response.AbortWithCode(c, http.StatusForbidden, domain.CodeOperatorCSRFRejected, "CSRF token is missing or invalid")
			return
		}
	}

	c.Set("userID", account.ID)
	c.Set("username", account.Username)
	c.Set(contextCredential, credential)
	c.Set(contextOperator, account)
	c.Next()
}

// writeError maps operator-starter failures to their public status and error code. Unexpected
// failures fall through to the shared mapper; persistence faults become a retryable 503.
func writeError(c *gin.Context, message string, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		response.AbortWithCode(c, http.StatusUnauthorized, domain.CodeInvalidCredentials, "Invalid credentials")
	case errors.Is(err, domain.ErrAuthenticationRequired):
		response.AbortWithCode(c, http.StatusUnauthorized, response.ErrorCodeUnauthorized, "Authentication required")
	case errors.Is(err, domain.ErrAccountDisabled):
		response.AbortWithCode(c, http.StatusForbidden, domain.CodeAccountDisabled, "Account access is disabled")
	case errors.Is(err, domain.ErrOperatorForbidden):
		response.AbortWithCode(c, http.StatusForbidden, domain.CodeOperatorForbidden, "Operator access is required")
	case errors.Is(err, domain.ErrOperatorTargetProtected):
		response.AbortWithCode(c, http.StatusConflict, domain.CodeOperatorTargetProtected, "Operator accounts cannot be changed here")
	case errors.Is(err, domain.ErrInvalidInput):
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid input")
	case errors.Is(err, domain.ErrUserNotFound):
		response.AbortWithCode(c, http.StatusNotFound, domain.CodeUserNotFound, "User not found")
	default:
		response.AbortWithCode(c, http.StatusServiceUnavailable, response.ErrorCodeServiceUnavailable, "Operator service unavailable")
		slog.ErrorContext(c.Request.Context(), "operator.request_failed", "operation", message, "err", err)
	}
}

func currentOperator(c *gin.Context) (*domain.User, string, bool) {
	accountValue, ok := c.Get(contextOperator)
	if !ok {
		return nil, "", false
	}
	account, ok := accountValue.(*domain.User)
	if !ok || account == nil {
		return nil, "", false
	}
	credential := c.GetString(contextCredential)
	return account, credential, credential != ""
}

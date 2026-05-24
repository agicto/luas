package apikey

import (
	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/middleware"
	"github.com/zgiai/zgo/internal/infra/router"
	"github.com/zgiai/zgo/pkg/handler"
	"github.com/zgiai/zgo/pkg/pagination"
	"github.com/zgiai/zgo/pkg/response"
)

// Handler handles API key HTTP requests.
type Handler struct {
	service Service
}

var (
	_ contracts.Module           = (*Handler)(nil)
	_ contracts.RouteModule      = (*Handler)(nil)
	_ contracts.MiddlewareModule = (*Handler)(nil)
)

// NewHandler creates a new API key handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the module name.
func (h *Handler) Name() string {
	return "apikey"
}

// RegisterMiddleware registers the api_key middleware group and alias.
func (h *Handler) RegisterMiddleware(r *router.Router) {
	keyAuth := middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:    "header:X-API-Key",
		ContextKey:   "apiKey",
		ErrorMessage: "Invalid or missing API key",
		ValidatorWithContext: func(c *gin.Context, key string) (*middleware.KeyAuthResult, error) {
			apiKey, err := h.service.Validate(c.Request.Context(), key)
			if err != nil {
				return nil, err
			}

			return &middleware.KeyAuthResult{
				Key: key,
				Values: map[string]any{
					"apiKeyID":     apiKey.ID,
					"apiKeyUserID": apiKey.UserID,
					"apiKeyScopes": append([]string(nil), apiKey.Scopes...),
					"userID":       apiKey.UserID,
				},
			}, nil
		},
	})

	r.MiddlewareGroup("api_key", keyAuth)
	r.AliasMiddleware("key", keyAuth)
}

// Create creates a new API key for the authenticated user.
func (h *Handler) Create(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	var req APIKeyCreateRequest
	if !handler.BindJSON(c, &req) {
		return
	}

	result, err := h.service.CreateForUser(c.Request.Context(), userID, &req)
	if err != nil {
		response.HandleError(c, "Failed to create API key", err)
		return
	}

	response.Created(c, toCreateResponse(result))
}

// List returns the authenticated user's API keys.
func (h *Handler) List(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	req := pagination.FromContext(c)
	items, total, err := h.service.ListForUser(c.Request.Context(), userID, req.GetPage(), req.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list API keys", err)
		return
	}

	responses := make([]*APIKeyResponse, len(items))
	for i, item := range items {
		responses[i] = toResponse(item)
	}

	paginator := pagination.NewPaginator(responses, total, req.GetPage(), req.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// Revoke revokes one of the authenticated user's API keys.
func (h *Handler) Revoke(c *gin.Context) {
	userID, ok := handler.GetUserID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "id")
	if !ok {
		return
	}

	if err := h.service.RevokeForUser(c.Request.Context(), userID, id); err != nil {
		response.HandleError(c, "Failed to revoke API key", err)
		return
	}

	response.NoContent(c)
}

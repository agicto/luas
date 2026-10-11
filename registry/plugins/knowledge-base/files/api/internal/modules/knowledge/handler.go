package knowledge

import (
	"encoding/base64"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/starter/assembly"
	httphandler "github.com/fastagi/fastbuild/api/pkg/handler"
	"github.com/fastagi/fastbuild/api/pkg/pagination"
	"github.com/fastagi/fastbuild/api/pkg/response"
)

var articleETagPattern = regexp.MustCompile(`^"knowledge-article-v([1-9][0-9]{0,18})"$`)

// Handler serves the knowledge routes.
type Handler struct {
	service Service
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
	_ assembly.ErrorModule = (*Handler)(nil)
)

// NewHandler returns the knowledge handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the starter name.
func (h *Handler) Name() string { return config.StarterKnowledge }

type articleResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Version   uint64    `json:"version"`
	CreatedBy *uint     `json:"created_by"`
	UpdatedBy *uint     `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type articleSummary struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Version   uint64    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type searchHitResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Excerpt   string    `json:"excerpt"`
	Version   uint64    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createArticleRequest struct {
	Title string `json:"title" binding:"required,max=200"`
	Body  string `json:"body" binding:"max=200000"`
}

type updateArticleRequest struct {
	Title *string `json:"title" binding:"omitempty,max=200"`
	Body  *string `json:"body" binding:"omitempty,max=200000"`
}

type searchRequest struct {
	Q string `form:"q" binding:"required,max=200"`
}

// List returns keyset pages of the organization's articles, newest first.
func (h *Handler) List(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	after, err := decodeCursor(c.Query("cursor"))
	if err != nil || len(c.QueryArray("cursor")) > 1 || c.Query("page") != "" {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid cursor")
		return
	}
	perPage := pagination.FromContext(c).GetPerPage()
	articles, next, err := h.service.List(c.Request.Context(), organization.OrganizationID, after, perPage)
	if err != nil {
		response.HandleError(c, "Failed to list knowledge articles", err)
		return
	}
	items := make([]articleSummary, len(articles))
	for index, article := range articles {
		items[index] = articleSummary{article.ID, article.Title, article.Version, article.UpdatedAt}
	}
	response.SuccessCursorPage(c, items, perPage, encodeCursor(next))
}

// Get returns one article.
func (h *Handler) Get(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	article, err := h.service.Get(c.Request.Context(), organization.OrganizationID, id)
	if err != nil {
		response.HandleError(c, "Failed to load knowledge article", err)
		return
	}
	c.Header("ETag", articleETag(article.Version))
	response.Success(c, toArticleResponse(article))
}

// Create adds an article; owners and admins only.
func (h *Handler) Create(c *gin.Context) {
	organization, actorID, ok := managerContext(c)
	if !ok {
		return
	}
	var request createArticleRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	article, err := h.service.Create(c.Request.Context(), organization.OrganizationID, actorID, request.Title, request.Body)
	if err != nil {
		response.HandleError(c, "Failed to create knowledge article", err)
		return
	}
	c.Header("ETag", articleETag(article.Version))
	response.Created(c, toArticleResponse(article))
}

// Update changes the title or body; it requires If-Match with the version that was read.
func (h *Handler) Update(c *gin.Context) {
	organization, actorID, ok := managerContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	version, ok := expectedVersion(c)
	if !ok {
		return
	}
	var request updateArticleRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	article, err := h.service.Update(c.Request.Context(), organization.OrganizationID, actorID, id, version,
		ArticleChanges(request))
	if err != nil {
		response.HandleError(c, "Failed to update knowledge article", err)
		return
	}
	c.Header("ETag", articleETag(article.Version))
	response.Success(c, toArticleResponse(article))
}

// Delete removes an article; it requires If-Match with the version that was read.
func (h *Handler) Delete(c *gin.Context) {
	organization, _, ok := managerContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	version, ok := expectedVersion(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), organization.OrganizationID, id, version); err != nil {
		response.HandleError(c, "Failed to delete knowledge article", err)
		return
	}
	response.NoContent(c)
}

// Search returns up to 20 ranked articles matching words or a substring.
func (h *Handler) Search(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	var request searchRequest
	if !httphandler.BindQuery(c, &request) {
		return
	}
	hits, err := h.service.Search(c.Request.Context(), organization.OrganizationID, request.Q)
	if err != nil {
		response.HandleError(c, "Failed to search knowledge articles", err)
		return
	}
	items := make([]searchHitResponse, len(hits))
	for index, hit := range hits {
		items[index] = searchHitResponse(hit)
	}
	response.Success(c, items)
}

func memberContext(c *gin.Context) (domain.OrganizationContext, bool) {
	setPrivateResponse(c)
	actorID, ok := httphandler.GetUserID(c)
	if !ok {
		return domain.OrganizationContext{}, false
	}
	organization, ok := domain.OrganizationContextFromContext(c.Request.Context())
	if !ok || organization.UserID != actorID {
		response.HandleError(c, "Organization context required", domain.ErrOrganizationContextRequired)
		return domain.OrganizationContext{}, false
	}
	return organization, true
}

func managerContext(c *gin.Context) (domain.OrganizationContext, uint, bool) {
	organization, ok := memberContext(c)
	if !ok {
		return domain.OrganizationContext{}, 0, false
	}
	if !organization.Role.CanManageOrganization() {
		response.HandleError(c, "Knowledge management forbidden", domain.ErrPermissionDenied)
		return domain.OrganizationContext{}, 0, false
	}
	return organization, organization.UserID, true
}

func expectedVersion(c *gin.Context) (uint64, bool) {
	values := c.Request.Header.Values("If-Match")
	if len(values) == 0 {
		response.HandleError(c, "Knowledge article version required", ErrPreconditionRequired)
		return 0, false
	}
	match := articleETagPattern.FindStringSubmatch(values[0])
	if len(values) != 1 || match == nil {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	version, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	return version, true
}

func articleETag(version uint64) string {
	return `"knowledge-article-v` + strconv.FormatUint(version, 10) + `"`
}

func toArticleResponse(article *Article) articleResponse {
	return articleResponse{
		ID: article.ID, Title: article.Title, Body: article.Body, Version: article.Version,
		CreatedBy: article.CreatedBy, UpdatedBy: article.UpdatedBy,
		CreatedAt: article.CreatedAt, UpdatedAt: article.UpdatedAt,
	}
}

func setPrivateResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization, Organization-Id")
}

// Cursors are opaque to clients; the layout is not part of the contract.
func encodeCursor(cursor *ArticleCursor) string {
	if cursor == nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(cursor.ID), 10)))
}

func decodeCursor(token string) (*ArticleCursor, error) {
	if token == "" {
		return nil, nil
	}
	if len(token) > 32 {
		return nil, domain.ErrInvalidInput
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || strings.TrimSpace(string(raw)) != string(raw) {
		return nil, domain.ErrInvalidInput
	}
	id, err := strconv.ParseUint(string(raw), 10, 64)
	if err != nil || id == 0 || uint64(uint(id)) != id {
		return nil, domain.ErrInvalidInput
	}
	return &ArticleCursor{ID: uint(id)}, nil
}

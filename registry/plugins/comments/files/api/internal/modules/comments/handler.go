package comments

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

var commentETagPattern = regexp.MustCompile(`^"comment-v([1-9][0-9]{0,18})"$`)

// Handler serves the comments routes.
type Handler struct {
	service Service
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
	_ assembly.ErrorModule = (*Handler)(nil)
)

// NewHandler returns the comments handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the starter name.
func (h *Handler) Name() string { return config.StarterComments }

type commentResponse struct {
	ID           uint       `json:"id"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	ParentID     *uint      `json:"parent_id"`
	AuthorID     *uint      `json:"author_id"`
	Body         *string    `json:"body"`
	Mentions     []uint     `json:"mentions"`
	Version      uint64     `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	EditedAt     *time.Time `json:"edited_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type threadResponse struct {
	Comment        commentResponse   `json:"comment"`
	Replies        []commentResponse `json:"replies"`
	HasMoreReplies bool              `json:"has_more_replies"`
}

type listRequest struct {
	ResourceType string `form:"resource_type" binding:"required,max=40"`
	ResourceID   string `form:"resource_id" binding:"required,max=64"`
}

type createCommentRequest struct {
	ResourceType string `json:"resource_type" binding:"required,max=40"`
	ResourceID   string `json:"resource_id" binding:"required,max=64"`
	ParentID     *uint  `json:"parent_id" binding:"omitempty,min=1"`
	Body         string `json:"body" binding:"required,max=10000"`
	Mentions     []uint `json:"mentions" binding:"omitempty,max=20"`
}

type updateCommentRequest struct {
	Body     *string `json:"body" binding:"omitempty,max=10000"`
	Mentions *[]uint `json:"mentions" binding:"omitempty,max=20"`
}

// List returns keyset pages of a resource's threads, oldest first, each with its first replies.
func (h *Handler) List(c *gin.Context) {
	actor, ok := memberContext(c)
	if !ok {
		return
	}
	after, ok := cursorQuery(c)
	if !ok {
		return
	}
	var request listRequest
	if !httphandler.BindQuery(c, &request) {
		return
	}
	perPage := pagination.FromContext(c).GetPerPage()
	threads, next, err := h.service.List(c.Request.Context(), actor, request.ResourceType, request.ResourceID, after, perPage)
	if err != nil {
		response.HandleError(c, "Failed to list comments", err)
		return
	}
	items := make([]threadResponse, len(threads))
	for index, thread := range threads {
		replies := make([]commentResponse, len(thread.Replies))
		for replyIndex, reply := range thread.Replies {
			replies[replyIndex] = toCommentResponse(reply)
		}
		items[index] = threadResponse{toCommentResponse(thread.Root), replies, thread.HasMoreReplies}
	}
	response.SuccessCursorPage(c, items, perPage, encodeCursor(next))
}

// ListReplies returns keyset pages of one thread's replies, oldest first.
func (h *Handler) ListReplies(c *gin.Context) {
	actor, ok := memberContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	after, ok := cursorQuery(c)
	if !ok {
		return
	}
	perPage := pagination.FromContext(c).GetPerPage()
	replies, next, err := h.service.ListReplies(c.Request.Context(), actor, id, after, perPage)
	if err != nil {
		response.HandleError(c, "Failed to list comment replies", err)
		return
	}
	items := make([]commentResponse, len(replies))
	for index, reply := range replies {
		items[index] = toCommentResponse(reply)
	}
	response.SuccessCursorPage(c, items, perPage, encodeCursor(next))
}

// Create adds a root comment or a reply.
func (h *Handler) Create(c *gin.Context) {
	actor, ok := memberContext(c)
	if !ok {
		return
	}
	var request createCommentRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	comment, err := h.service.Create(c.Request.Context(), actor, NewComment(request))
	if err != nil {
		response.HandleError(c, "Failed to create comment", err)
		return
	}
	c.Header("ETag", commentETag(comment.Version))
	response.Created(c, toCommentResponse(comment))
}

// Update edits the caller's own comment; it requires If-Match with the version that was read.
func (h *Handler) Update(c *gin.Context) {
	actor, ok := memberContext(c)
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
	var request updateCommentRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	comment, err := h.service.Update(c.Request.Context(), actor, id, version, CommentChanges(request))
	if err != nil {
		response.HandleError(c, "Failed to update comment", err)
		return
	}
	c.Header("ETag", commentETag(comment.Version))
	response.Success(c, toCommentResponse(comment))
}

// Delete soft-deletes a comment; the author or an organization owner or admin may delete.
func (h *Handler) Delete(c *gin.Context) {
	actor, ok := memberContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), actor, id); err != nil {
		response.HandleError(c, "Failed to delete comment", err)
		return
	}
	response.NoContent(c)
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

func cursorQuery(c *gin.Context) (*Cursor, bool) {
	after, err := decodeCursor(c.Query("cursor"))
	if err != nil || len(c.QueryArray("cursor")) > 1 || c.Query("page") != "" {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid cursor")
		return nil, false
	}
	return after, true
}

func expectedVersion(c *gin.Context) (uint64, bool) {
	values := c.Request.Header.Values("If-Match")
	if len(values) == 0 {
		response.HandleError(c, "Comment version required", ErrPreconditionRequired)
		return 0, false
	}
	match := commentETagPattern.FindStringSubmatch(values[0])
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

func commentETag(version uint64) string {
	return `"comment-v` + strconv.FormatUint(version, 10) + `"`
}

// toCommentResponse hides the body and mentions of a deleted comment.
func toCommentResponse(comment *Comment) commentResponse {
	out := commentResponse{
		ID: comment.ID, ResourceType: comment.ResourceType, ResourceID: comment.ResourceID,
		ParentID: comment.ParentID, AuthorID: comment.AuthorID, Mentions: []uint{}, Version: comment.Version,
		CreatedAt: comment.CreatedAt, UpdatedAt: comment.UpdatedAt, EditedAt: comment.EditedAt, DeletedAt: comment.DeletedAt,
	}
	if !comment.IsDeleted() {
		body := comment.Body
		out.Body = &body
		if comment.Mentions != nil {
			out.Mentions = comment.Mentions
		}
	}
	return out
}

func setPrivateResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization, Organization-Id")
}

// Cursors are opaque to clients; the layout is not part of the contract.
func encodeCursor(cursor *Cursor) string {
	if cursor == nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(cursor.ID), 10)))
}

func decodeCursor(token string) (*Cursor, error) {
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
	return &Cursor{ID: uint(id)}, nil
}

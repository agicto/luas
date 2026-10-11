package tag

import (
	"encoding/base64"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/starter/assembly"
	httphandler "github.com/fastagi/fastbuild/api/pkg/handler"
	"github.com/fastagi/fastbuild/api/pkg/pagination"
	"github.com/fastagi/fastbuild/api/pkg/response"
)

var tagETagPattern = regexp.MustCompile(`^"tag-v([1-9][0-9]{0,18})"$`)

// Handler serves the tag routes.
type Handler struct {
	service Service
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
	_ assembly.ErrorModule = (*Handler)(nil)
)

// NewHandler returns the tag handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the starter name.
func (h *Handler) Name() string { return config.StarterTag }

type tagResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	Version     uint64    `json:"version"`
	CreatedBy   *uint     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type taggedResourceResponse struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	TaggedAt     time.Time `json:"tagged_at"`
}

type createTagRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Color       string `json:"color" binding:"omitempty,max=16"`
	Description string `json:"description" binding:"max=200"`
}

type updateTagRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=50"`
	Color       *string `json:"color" binding:"omitempty,max=16"`
	Description *string `json:"description" binding:"omitempty,max=200"`
}

type replaceResourceTagsRequest struct {
	TagIDs []uint `json:"tag_ids" binding:"required,max=20,unique,dive,gt=0"`
}

// List returns keyset pages of the organization's tags ordered by name.
func (h *Handler) List(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	after, err := decodeTagCursor(c.Query("cursor"))
	if err != nil || len(c.QueryArray("cursor")) > 1 || c.Query("page") != "" {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid cursor")
		return
	}
	perPage := pagination.FromContext(c).GetPerPage()
	tags, next, err := h.service.List(c.Request.Context(), organization.OrganizationID, after, perPage)
	if err != nil {
		response.HandleError(c, "Failed to list tags", err)
		return
	}
	response.SuccessCursorPage(c, toTagResponses(tags), perPage, encodeTagCursor(next))
}

// Get returns one tag.
func (h *Handler) Get(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	tag, err := h.service.Get(c.Request.Context(), organization.OrganizationID, id)
	if err != nil {
		response.HandleError(c, "Failed to load tag", err)
		return
	}
	c.Header("ETag", tagETag(tag.Version))
	response.Success(c, toTagResponse(tag))
}

// Create adds a tag; owners and admins only.
func (h *Handler) Create(c *gin.Context) {
	organization, actorID, ok := managerContext(c)
	if !ok {
		return
	}
	var request createTagRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	tag, err := h.service.Create(c.Request.Context(), organization.OrganizationID, actorID, TagInput(request))
	if err != nil {
		response.HandleError(c, "Failed to create tag", err)
		return
	}
	c.Header("ETag", tagETag(tag.Version))
	response.Created(c, toTagResponse(tag))
}

// Update changes the name, color, or description; it requires If-Match with the version read.
func (h *Handler) Update(c *gin.Context) {
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
	var request updateTagRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	tag, err := h.service.Update(c.Request.Context(), organization.OrganizationID, id, version, TagChanges(request))
	if err != nil {
		response.HandleError(c, "Failed to update tag", err)
		return
	}
	c.Header("ETag", tagETag(tag.Version))
	response.Success(c, toTagResponse(tag))
}

// Delete removes a tag and every assignment of it; it requires If-Match with the version read.
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
		response.HandleError(c, "Failed to delete tag", err)
		return
	}
	response.NoContent(c)
}

// Resources returns keyset pages of the resources carrying a tag, newest assignment first.
func (h *Handler) Resources(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	after, err := decodeAssignmentCursor(c.Query("cursor"))
	resourceType := c.Query("resource_type")
	if err != nil || len(c.QueryArray("cursor")) > 1 || len(c.QueryArray("resource_type")) > 1 || c.Query("page") != "" ||
		(resourceType != "" && !resourceTypePattern.MatchString(resourceType)) {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid query parameters")
		return
	}
	perPage := pagination.FromContext(c).GetPerPage()
	resources, next, err := h.service.TaggedResources(c.Request.Context(), organization.OrganizationID, id, resourceType, after, perPage)
	if err != nil {
		response.HandleError(c, "Failed to list tagged resources", err)
		return
	}
	items := make([]taggedResourceResponse, len(resources))
	for index, resource := range resources {
		items[index] = taggedResourceResponse{resource.ResourceType, resource.ResourceID, resource.TaggedAt}
	}
	response.SuccessCursorPage(c, items, perPage, encodeAssignmentCursor(next))
}

// ResourceTags returns the tags one resource carries, ordered by name.
//
// fastbuild:bounded-list max=20 reason=bounded-resource-tag-set
func (h *Handler) ResourceTags(c *gin.Context) {
	organization, ok := memberContext(c)
	if !ok {
		return
	}
	ref, ok := resourceRef(c)
	if !ok {
		return
	}
	tags, err := h.service.ResourceTags(c.Request.Context(), organization.OrganizationID, ref)
	if err != nil {
		response.HandleError(c, "Failed to list resource tags", err)
		return
	}
	response.Success(c, toTagResponses(tags))
}

// ReplaceResourceTags makes the request's tag IDs the resource's complete tag set; owners and
// admins only. Repeating the same request changes nothing.
func (h *Handler) ReplaceResourceTags(c *gin.Context) {
	organization, _, ok := managerContext(c)
	if !ok {
		return
	}
	ref, ok := resourceRef(c)
	if !ok {
		return
	}
	var request replaceResourceTagsRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	tags, err := h.service.ReplaceResourceTags(c.Request.Context(), organization.OrganizationID, ref, request.TagIDs)
	if err != nil {
		response.HandleError(c, "Failed to replace resource tags", err)
		return
	}
	response.Success(c, toTagResponses(tags))
}

func resourceRef(c *gin.Context) (ResourceRef, bool) {
	ref := ResourceRef{Type: c.Param("resource_type"), ID: c.Param("resource_id")}
	if !ValidResourceRef(ref) {
		response.AbortWithCode(c, http.StatusBadRequest, response.ErrorCodeInvalidInput, "Invalid resource reference")
		return ResourceRef{}, false
	}
	return ref, true
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
		response.HandleError(c, "Tag management forbidden", domain.ErrPermissionDenied)
		return domain.OrganizationContext{}, 0, false
	}
	return organization, organization.UserID, true
}

func expectedVersion(c *gin.Context) (uint64, bool) {
	values := c.Request.Header.Values("If-Match")
	if len(values) == 0 {
		response.HandleError(c, "Tag version required", ErrPreconditionRequired)
		return 0, false
	}
	match := tagETagPattern.FindStringSubmatch(values[0])
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

func tagETag(version uint64) string {
	return `"tag-v` + strconv.FormatUint(version, 10) + `"`
}

func toTagResponse(tag *Tag) tagResponse {
	return tagResponse{
		ID: tag.ID, Name: tag.Name, Color: tag.Color, Description: tag.Description, Version: tag.Version,
		CreatedBy: tag.CreatedBy, CreatedAt: tag.CreatedAt, UpdatedAt: tag.UpdatedAt,
	}
}

func toTagResponses(tags []*Tag) []tagResponse {
	items := make([]tagResponse, len(tags))
	for index, tag := range tags {
		items[index] = toTagResponse(tag)
	}
	return items
}

func setPrivateResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization, Organization-Id")
}

// Cursors are opaque to clients; their layout is not part of the contract.
func encodeTagCursor(cursor *TagCursor) string {
	if cursor == nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(cursor.SortKey))
}

func decodeTagCursor(token string) (*TagCursor, error) {
	if token == "" {
		return nil, nil
	}
	if len(token) > base64.RawURLEncoding.EncodedLen(maxTagCursorSortKeyLen) {
		return nil, domain.ErrInvalidInput
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) == 0 || !utf8.Valid(raw) || utf8.RuneCount(raw) > maxNameLength {
		return nil, domain.ErrInvalidInput
	}
	return &TagCursor{SortKey: string(raw)}, nil
}

func encodeAssignmentCursor(cursor *AssignmentCursor) string {
	if cursor == nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(cursor.ID), 10)))
}

func decodeAssignmentCursor(token string) (*AssignmentCursor, error) {
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
	return &AssignmentCursor{ID: uint(id)}, nil
}

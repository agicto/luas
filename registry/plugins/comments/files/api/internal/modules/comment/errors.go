package comment

import (
	"errors"
	"net/http"

	"github.com/fastagi/fastbuild/api/pkg/response"
)

// Stable error codes owned by this starter.
const (
	CodeCommentNotFound         = "COMMENT.NOT_FOUND"
	CodeResourceNotFound        = "COMMENT.RESOURCE_NOT_FOUND"
	CodeUnsupportedResourceType = "COMMENT.UNSUPPORTED_RESOURCE_TYPE"
	CodeInvalidComment          = "COMMENT.INVALID"
	CodeInvalidParent           = "COMMENT.INVALID_PARENT"
	CodeInvalidMentions         = "COMMENT.INVALID_MENTIONS"
	CodeVersionConflict         = "COMMENT.VERSION_CONFLICT"
	CodePreconditionRequired    = "COMMENT.PRECONDITION_REQUIRED"
)

var (
	// ErrCommentNotFound also covers deleted comments on writes, comments of other organizations,
	// and comments on resources the caller may not read, so IDs disclose nothing.
	ErrCommentNotFound = errors.New("comment not found")
	// ErrResourceNotFound covers resources that do not exist in the organization and resources the
	// caller may not read.
	ErrResourceNotFound        = errors.New("commented resource not found")
	ErrUnsupportedResourceType = errors.New("comment resource type is not registered")
	ErrInvalidComment          = errors.New("comment is invalid")
	ErrInvalidParent           = errors.New("comment parent is invalid")
	ErrInvalidMentions         = errors.New("comment mentions are invalid")
	ErrVersionConflict         = errors.New("comment changed since it was read")
	ErrPreconditionRequired    = errors.New("comment version precondition is required")
)

// RegisterErrorMappings maps this starter's errors to public status and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(ErrCommentNotFound, http.StatusNotFound, CodeCommentNotFound)
	mapper.Register(ErrResourceNotFound, http.StatusNotFound, CodeResourceNotFound)
	mapper.Register(ErrUnsupportedResourceType, http.StatusUnprocessableEntity, CodeUnsupportedResourceType)
	mapper.Register(ErrInvalidComment, http.StatusUnprocessableEntity, CodeInvalidComment)
	mapper.Register(ErrInvalidParent, http.StatusUnprocessableEntity, CodeInvalidParent)
	mapper.Register(ErrInvalidMentions, http.StatusUnprocessableEntity, CodeInvalidMentions)
	mapper.Register(ErrVersionConflict, http.StatusPreconditionFailed, CodeVersionConflict)
	mapper.Register(ErrPreconditionRequired, http.StatusPreconditionRequired, CodePreconditionRequired)
}

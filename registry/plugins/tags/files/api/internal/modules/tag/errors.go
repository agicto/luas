package tag

import (
	"errors"
	"net/http"

	"github.com/fastagi/fastbuild/api/pkg/response"
)

// Stable error codes owned by this starter.
const (
	CodeNotFound             = "TAG.NOT_FOUND"
	CodeInvalid              = "TAG.INVALID"
	CodeNameTaken            = "TAG.NAME_TAKEN"
	CodeVersionConflict      = "TAG.VERSION_CONFLICT"
	CodePreconditionRequired = "TAG.PRECONDITION_REQUIRED"
	CodeResourceNotFound     = "TAG.RESOURCE_NOT_FOUND"
	CodeInvalidAssignment    = "TAG.INVALID_ASSIGNMENT"
)

var (
	// ErrNotFound also covers tags of other organizations, so IDs disclose nothing.
	ErrNotFound = errors.New("tag not found")
	// ErrInvalid reports a blank or too long name, an unknown color, or an empty update.
	ErrInvalid              = errors.New("tag is invalid")
	ErrNameTaken            = errors.New("tag name is already used in the organization")
	ErrVersionConflict      = errors.New("tag changed since it was read")
	ErrPreconditionRequired = errors.New("tag version precondition is required")
	// ErrResourceNotFound covers an unregistered resource type and a resource that is not in the
	// active organization alike.
	ErrResourceNotFound = errors.New("tagged resource not found")
	// ErrInvalidAssignment reports a tag set that is too large, repeats a tag, or names a tag that
	// is not in the organization.
	ErrInvalidAssignment = errors.New("tag assignment is invalid")
)

// RegisterErrorMappings maps this starter's errors to public status and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(ErrNotFound, http.StatusNotFound, CodeNotFound)
	mapper.Register(ErrInvalid, http.StatusUnprocessableEntity, CodeInvalid)
	mapper.Register(ErrNameTaken, http.StatusConflict, CodeNameTaken)
	mapper.Register(ErrVersionConflict, http.StatusPreconditionFailed, CodeVersionConflict)
	mapper.Register(ErrPreconditionRequired, http.StatusPreconditionRequired, CodePreconditionRequired)
	mapper.Register(ErrResourceNotFound, http.StatusNotFound, CodeResourceNotFound)
	mapper.Register(ErrInvalidAssignment, http.StatusUnprocessableEntity, CodeInvalidAssignment)
}

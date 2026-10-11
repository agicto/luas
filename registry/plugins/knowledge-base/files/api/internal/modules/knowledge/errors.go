package knowledge

import (
	"errors"
	"net/http"

	"github.com/fastagi/fastbuild/api/pkg/response"
)

// Stable error codes owned by this starter.
const (
	CodeArticleNotFound      = "KNOWLEDGE.ARTICLE_NOT_FOUND"
	CodeInvalidArticle       = "KNOWLEDGE.INVALID_ARTICLE"
	CodeVersionConflict      = "KNOWLEDGE.VERSION_CONFLICT"
	CodePreconditionRequired = "KNOWLEDGE.PRECONDITION_REQUIRED"
)

var (
	// ErrArticleNotFound also covers articles of other organizations, so IDs disclose nothing.
	ErrArticleNotFound      = errors.New("knowledge article not found")
	ErrInvalidArticle       = errors.New("knowledge article is invalid")
	ErrVersionConflict      = errors.New("knowledge article changed since it was read")
	ErrPreconditionRequired = errors.New("knowledge article version precondition is required")
)

// RegisterErrorMappings maps this starter's errors to public status and error codes.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	mapper.Register(ErrArticleNotFound, http.StatusNotFound, CodeArticleNotFound)
	mapper.Register(ErrInvalidArticle, http.StatusUnprocessableEntity, CodeInvalidArticle)
	mapper.Register(ErrVersionConflict, http.StatusPreconditionFailed, CodeVersionConflict)
	mapper.Register(ErrPreconditionRequired, http.StatusPreconditionRequired, CodePreconditionRequired)
}

package permission

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

// Guard adapts the domain authorizer to route middleware without global state.
type Guard struct {
	authorizer domain.PermissionAuthorizer
}

// NewGuard creates a permission route guard factory.
func NewGuard(authorizer domain.PermissionAuthorizer) *Guard {
	return &Guard{authorizer: authorizer}
}

// Require returns middleware for one exact registered permission.
func (g *Guard) Require(permission domain.PermissionKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		organization, ok := domain.OrganizationContextFromContext(c.Request.Context())
		if !ok {
			abortPermission(c, domain.ErrOrganizationContextRequired)
			return
		}
		if g == nil || g.authorizer == nil {
			abortPermission(c, domain.ErrServiceUnavailable)
			return
		}
		if err := g.authorizer.Authorize(c.Request.Context(), organization, permission); err != nil {
			abortPermission(c, err)
			return
		}
		c.Next()
	}
}

func abortPermission(c *gin.Context, err error) {
	descriptor := response.DefaultErrorMapper.Resolve(err)
	message := "Permission check failed"
	if errors.Is(err, domain.ErrPermissionDenied) {
		message = "Permission denied"
	}
	response.AbortWithCode(c, descriptor.StatusCode, descriptor.ErrorCode, message)
}

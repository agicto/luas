package permission

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

type guardAuthorizerStub struct {
	err error
}

func (s guardAuthorizerStub) Effective(context.Context, domain.OrganizationContext) (*domain.PermissionContext, error) {
	return nil, s.err
}

func (s guardAuthorizerStub) Authorize(context.Context, domain.OrganizationContext, domain.PermissionKey) error {
	return s.err
}

func TestGuardRequiresTypedOrganizationContextAndAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response.DefaultErrorMapper.Register(domain.ErrOrganizationContextRequired, http.StatusBadRequest, domain.CodeOrganizationContextRequired)
	response.DefaultErrorMapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)

	organization := domain.OrganizationContext{
		OrganizationID:   1,
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
		MembershipID:     2,
		UserID:           3,
		Role:             domain.OrganizationRoleMember,
	}
	tests := []struct {
		name       string
		authorizer domain.PermissionAuthorizer
		context    *domain.OrganizationContext
		wantStatus int
		wantCode   string
		wantCalled bool
	}{
		{
			name:       "allowed",
			authorizer: guardAuthorizerStub{},
			context:    &organization,
			wantStatus: http.StatusNoContent,
			wantCalled: true,
		},
		{
			name:       "denied",
			authorizer: guardAuthorizerStub{err: domain.ErrPermissionDenied},
			context:    &organization,
			wantStatus: http.StatusForbidden,
			wantCode:   domain.CodePermissionDenied,
		},
		{
			name:       "missing organization context",
			authorizer: guardAuthorizerStub{},
			wantStatus: http.StatusBadRequest,
			wantCode:   domain.CodeOrganizationContextRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			engine := gin.New()
			engine.GET("/guarded", func(c *gin.Context) {
				if tt.context != nil {
					c.Request = c.Request.WithContext(domain.WithOrganizationContext(c.Request.Context(), *tt.context))
				}
			}, NewGuard(tt.authorizer).Require("projects.read"), func(c *gin.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/guarded", nil)
			engine.ServeHTTP(recorder, request)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantCalled, called)
			if tt.wantCode != "" {
				var body struct {
					ErrorCode string `json:"error_code"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				assert.Equal(t, tt.wantCode, body.ErrorCode)
			}
		})
	}
}

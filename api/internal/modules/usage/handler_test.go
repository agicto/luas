package usage

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

func TestUsageUserListReturnsFinitePrivateSummaryOnly(t *testing.T) {
	fixture := newUsageTestFixture(t)
	handler := &Handler{service: fixture.service}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/usage/user", func(c *gin.Context) {
		c.Set("userID", fixture.user.ID)
		handler.UserList(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/usage/user", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", recorder.Header().Get("Pragma"))
	assert.Equal(t, "Authorization", recorder.Header().Get("Vary"))

	var body struct {
		Data []UsageSummaryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data, 5)
	assert.Equal(t, domain.UsageScopeUser, body.Data[0].Scope)
	assert.NotContains(t, recorder.Body.String(), "event_id")
	assert.NotContains(t, recorder.Body.String(), "fingerprint")
	assert.NotContains(t, recorder.Body.String(), "dimensions")
}

func TestUsageOrganizationListRequiresManagerRoleAndPrivateContext(t *testing.T) {
	fixture := newUsageTestFixture(t)
	handler := &Handler{service: fixture.service}
	registerUsageTestErrors()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/organization-usage/:role", func(c *gin.Context) {
		organizationContext := domain.OrganizationContext{
			OrganizationID:   fixture.organization.ID,
			OrganizationName: fixture.organization.Name,
			OrganizationSlug: fixture.organization.Slug,
			MembershipID:     1,
			UserID:           fixture.user.ID,
			Role:             domain.OrganizationRole(c.Param("role")),
		}
		c.Request = c.Request.WithContext(domain.WithOrganizationContext(context.Background(), organizationContext))
		handler.OrganizationList(c)
	})

	member := httptest.NewRecorder()
	router.ServeHTTP(member, httptest.NewRequest(http.MethodGet, "/organization-usage/member", nil))
	assert.Equal(t, http.StatusForbidden, member.Code)
	assert.Equal(t, domain.CodePermissionDenied, usageResponseErrorCode(t, member))
	assert.Equal(t, "private, no-store", member.Header().Get("Cache-Control"))
	assert.Equal(t, "Authorization, Organization-Id", member.Header().Get("Vary"))

	admin := httptest.NewRecorder()
	router.ServeHTTP(admin, httptest.NewRequest(http.MethodGet, "/organization-usage/admin", nil))
	require.Equal(t, http.StatusOK, admin.Code)
	var body struct {
		Data []UsageSummaryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(admin.Body.Bytes(), &body))
	require.Len(t, body.Data, 5)
	assert.Equal(t, domain.UsageScopeOrganization, body.Data[0].Scope)
}

func registerUsageTestErrors() {
	response.DefaultErrorMapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
	response.DefaultErrorMapper.Register(domain.ErrOrganizationContextRequired, http.StatusBadRequest, domain.CodeOrganizationContextRequired)
}

func usageResponseErrorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.ErrorCode
}

package organization

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type contextServiceStub struct {
	Service
	resolveFn func(context.Context, uint, uint) (*domain.OrganizationContext, error)
}

func (s *contextServiceStub) ResolveContext(
	ctx context.Context,
	userID, organizationID uint,
) (*domain.OrganizationContext, error) {
	return s.resolveFn(ctx, userID, organizationID)
}

func TestParseOrganizationIDHeaderRejectsMissingAmbiguousAndNonCanonicalValues(t *testing.T) {
	overflow := "18446744073709551616"
	if strconv.IntSize == 32 {
		overflow = strconv.FormatUint(uint64(math.MaxUint32)+1, 10)
	}
	tests := []struct {
		name   string
		values []string
		err    error
	}{
		{name: "missing", err: domain.ErrOrganizationContextRequired},
		{name: "empty", values: []string{""}, err: domain.ErrOrganizationContextInvalid},
		{name: "whitespace", values: []string{"   "}, err: domain.ErrOrganizationContextInvalid},
		{name: "zero", values: []string{"0"}, err: domain.ErrOrganizationContextInvalid},
		{name: "leading zero", values: []string{"01"}, err: domain.ErrOrganizationContextInvalid},
		{name: "positive sign", values: []string{"+1"}, err: domain.ErrOrganizationContextInvalid},
		{name: "negative", values: []string{"-1"}, err: domain.ErrOrganizationContextInvalid},
		{name: "combined duplicate", values: []string{"1, 2"}, err: domain.ErrOrganizationContextInvalid},
		{name: "multiple fields", values: []string{"1", "2"}, err: domain.ErrOrganizationContextInvalid},
		{name: "overflow", values: []string{overflow}, err: domain.ErrOrganizationContextInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseOrganizationIDHeader(tt.values)
			require.ErrorIs(t, err, tt.err)
		})
	}

	organizationID, err := parseOrganizationIDHeader([]string{" 42 "})
	require.NoError(t, err)
	assert.Equal(t, uint(42), organizationID)
}

func TestContextResolverBindsVerifiedContextAndPreservesVary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &contextServiceStub{
		resolveFn: func(_ context.Context, userID, organizationID uint) (*domain.OrganizationContext, error) {
			assert.Equal(t, uint(17), userID)
			assert.Equal(t, uint(42), organizationID)
			return &domain.OrganizationContext{
				OrganizationID:   42,
				OrganizationName: "Context Org",
				OrganizationSlug: "context-org",
				MembershipID:     91,
				UserID:           17,
				Role:             domain.OrganizationRoleAdmin,
			}, nil
		},
	}
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userID", uint(17))
		c.Header("Vary", "Accept-Encoding")
		c.Next()
	})
	engine.Use(NewContextResolver(service).Middleware())
	engine.GET("/", func(c *gin.Context) {
		resolved, ok := domain.OrganizationContextFromContext(c.Request.Context())
		require.True(t, ok)
		assert.Equal(t, uint(91), resolved.MembershipID)
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(OrganizationIDHeader, "42")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.ElementsMatch(t, []string{"Accept-Encoding", OrganizationIDHeader}, response.Header().Values("Vary"))
}

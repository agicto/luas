package organization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestOperatorDirectoryListsEveryOrganizationWithoutMembership(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	ownerID := createOrganizationTestUser(t, db, "owner")
	memberID := createOrganizationTestUser(t, db, "member")
	goneID := createOrganizationTestUser(t, db, "gone")

	acme := &domain.Organization{Name: "Acme 100%", Slug: "acme", CreatedBy: ownerID}
	require.NoError(t, repo.CreateWithOwner(ctx, acme, &domain.OrganizationMembership{UserID: ownerID, Role: domain.OrganizationRoleOwner}))
	globex := &domain.Organization{Name: "Globex_Co", Slug: "globex", CreatedBy: ownerID}
	require.NoError(t, repo.CreateWithOwner(ctx, globex, &domain.OrganizationMembership{UserID: ownerID, Role: domain.OrganizationRoleOwner}))
	for _, userID := range []uint{memberID, goneID} {
		require.NoError(t, db.Create(&OrganizationMembershipPO{
			OrganizationID: acme.ID, UserID: userID, Role: string(domain.OrganizationRoleMember),
		}).Error)
	}
	require.NoError(t, db.Model(&user.UserPO{}).Where("id = ?", goneID).Update("deleted_at", time.Now()).Error)

	t.Run("newest first with member counts that skip deleted accounts", func(t *testing.T) {
		items, total, err := repo.operatorList(ctx, "", 1, 10)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Len(t, items, 2)
		assert.Equal(t, globex.ID, items[0].ID)
		assert.EqualValues(t, 1, items[0].MemberCount)
		assert.Equal(t, acme.ID, items[1].ID)
		assert.EqualValues(t, 2, items[1].MemberCount)
		assert.Equal(t, ownerID, items[1].CreatedBy)
	})

	t.Run("search matches name or slug and treats wildcards literally", func(t *testing.T) {
		byName, total, err := repo.operatorList(ctx, "ACME", 1, 10)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		assert.Equal(t, acme.ID, byName[0].ID)

		bySlug, _, err := repo.operatorList(ctx, "globex", 1, 10)
		require.NoError(t, err)
		require.Len(t, bySlug, 1)
		assert.Equal(t, globex.ID, bySlug[0].ID)

		for literal, wanted := range map[string]int{"%": 1, "0%": 1, "globex_": 1, "acme_": 0, "a_me": 0} {
			items, _, listErr := repo.operatorList(ctx, literal, 1, 10)
			require.NoError(t, listErr)
			assert.Len(t, items, wanted, literal)
		}
	})

	t.Run("pages and rejects out-of-range paging", func(t *testing.T) {
		items, total, err := repo.operatorList(ctx, "", 2, 1)
		require.NoError(t, err)
		assert.EqualValues(t, 2, total)
		require.Len(t, items, 1)
		assert.Equal(t, acme.ID, items[0].ID)

		_, _, err = repo.operatorList(ctx, "", 1, 101)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("finds one organization or reports not found", func(t *testing.T) {
		found, err := repo.operatorFind(ctx, acme.ID)
		require.NoError(t, err)
		assert.Equal(t, "acme", found.Slug)
		assert.EqualValues(t, 2, found.MemberCount)

		_, err = repo.operatorFind(ctx, acme.ID+1000)
		require.ErrorIs(t, err, domain.ErrOrganizationNotFound)
	})

	t.Run("lists members with email and skips deleted accounts", func(t *testing.T) {
		members, total, err := repo.operatorMembers(ctx, acme.ID, 1, 10)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Len(t, members, 2)
		assert.Equal(t, ownerID, members[0].UserID)
		assert.Equal(t, "owner@example.com", members[0].Email)
		assert.Equal(t, domain.OrganizationRoleOwner, members[0].Role)
		assert.Equal(t, memberID, members[1].UserID)
		assert.Equal(t, domain.OrganizationRoleMember, members[1].Role)
	})
}

func TestOperatorHandlerRequireStopsUnknownOrganizations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newOrganizationRepositoryTestDB(t)
	repo := NewRepository(db)
	ownerID := createOrganizationTestUser(t, db, "owner")
	acme := &domain.Organization{Name: "Acme", Slug: "acme", CreatedBy: ownerID}
	require.NoError(t, repo.CreateWithOwner(context.Background(), acme, &domain.OrganizationMembership{UserID: ownerID, Role: domain.OrganizationRoleOwner}))
	response.DefaultErrorMapper.Register(domain.ErrOrganizationNotFound, http.StatusNotFound, domain.CodeOrganizationNotFound)

	handler := NewOperatorHandler(repo)
	engine := gin.New()
	engine.GET("/organizations", handler.List)
	engine.GET("/organizations/:id", handler.Get)
	engine.GET("/organizations/:id/members", handler.ListMembers)
	reached := 0
	engine.GET("/organizations/:id/nested", handler.Require, func(c *gin.Context) {
		reached++
		c.Status(http.StatusNoContent)
	})

	get := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}

	missing := get("/organizations/999999/nested")
	assert.Equal(t, http.StatusNotFound, missing.Code)
	assert.Contains(t, missing.Body.String(), domain.CodeOrganizationNotFound)
	assert.Zero(t, reached)

	assert.Equal(t, http.StatusNoContent, get("/organizations/1/nested").Code)
	assert.Equal(t, 1, reached)
	assert.Equal(t, http.StatusNotFound, get("/organizations/999999").Code)
	assert.Equal(t, http.StatusBadRequest, get("/organizations?q="+strings.Repeat("a", 101)).Code)

	list := get("/organizations?q=acme")
	require.Equal(t, http.StatusOK, list.Code)
	var page struct {
		Data []OperatorOrganizationResponse `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &page))
	require.Len(t, page.Data, 1)
	assert.EqualValues(t, 1, page.Meta.Total)
	assert.EqualValues(t, 1, page.Data[0].MemberCount)

	unknownMembers := get("/organizations/999999/members")
	assert.Equal(t, http.StatusNotFound, unknownMembers.Code)
	assert.Contains(t, unknownMembers.Body.String(), domain.CodeOrganizationNotFound)
	assert.Equal(t, http.StatusBadRequest, get("/organizations/0").Code)

	members := get("/organizations/1/members")
	require.Equal(t, http.StatusOK, members.Code)
	assert.Contains(t, members.Body.String(), `"email":"owner@example.com"`)
}

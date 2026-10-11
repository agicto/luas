package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/pkg/response"
)

// memoryStore is an in-memory store that enforces the same organization scope and versions.
type memoryStore struct {
	articles map[uint]*Article
	nextID   uint
}

func newMemoryStore() *memoryStore { return &memoryStore{articles: map[uint]*Article{}} }

func (s *memoryStore) List(_ context.Context, organizationID uint, after *ArticleCursor, limit int) ([]*Article, *ArticleCursor, error) {
	var items []*Article
	for id := s.nextID; id > 0; id-- {
		article, ok := s.articles[id]
		if !ok || article.OrganizationID != organizationID || (after != nil && id >= after.ID) {
			continue
		}
		if len(items) == limit {
			return items, &ArticleCursor{ID: items[len(items)-1].ID}, nil
		}
		items = append(items, article)
	}
	return items, nil, nil
}

func (s *memoryStore) Get(_ context.Context, organizationID, id uint) (*Article, error) {
	article, ok := s.articles[id]
	if !ok || article.OrganizationID != organizationID {
		return nil, ErrArticleNotFound
	}
	copied := *article
	return &copied, nil
}

func (s *memoryStore) Create(_ context.Context, article *Article) error {
	s.nextID++
	article.ID, article.Version, article.CreatedAt, article.UpdatedAt = s.nextID, 1, time.Now(), time.Now()
	stored := *article
	s.articles[article.ID] = &stored
	return nil
}

func (s *memoryStore) Update(ctx context.Context, organizationID, id uint, expected uint64, changes ArticleChanges, actorID uint) (*Article, error) {
	article, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if article.Version != expected {
		return nil, ErrVersionConflict
	}
	if changes.Title != nil {
		article.Title = *changes.Title
	}
	if changes.Body != nil {
		article.Body = *changes.Body
	}
	article.Version++
	article.UpdatedBy = &actorID
	s.articles[id] = article
	return s.Get(ctx, organizationID, id)
}

func (s *memoryStore) Delete(ctx context.Context, organizationID, id uint, expected uint64) error {
	article, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if article.Version != expected {
		return ErrVersionConflict
	}
	delete(s.articles, id)
	return nil
}

func (s *memoryStore) Search(_ context.Context, organizationID uint, query string, limit int) ([]SearchHit, error) {
	var hits []SearchHit
	for _, article := range s.articles {
		if article.OrganizationID == organizationID && len(hits) < limit && bytes.Contains([]byte(article.Title+article.Body), []byte(query)) {
			hits = append(hits, SearchHit{ID: article.ID, Title: article.Title})
		}
	}
	return hits, nil
}

type harness struct {
	handler *Handler
	store   *memoryStore
}

func newHarness() *harness {
	gin.SetMode(gin.TestMode)
	store := newMemoryStore()
	handler := NewHandler(NewService(store))
	handler.RegisterErrorMappings(response.DefaultErrorMapper)
	// The application registers these core mappings at startup.
	response.DefaultErrorMapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
	response.DefaultErrorMapper.Register(domain.ErrInvalidInput, http.StatusBadRequest, domain.CodeInvalidInput)
	return &harness{handler: handler, store: store}
}

type request struct {
	method, path, body, ifMatch string
	organization                uint
	role                        domain.OrganizationRole
	params                      gin.Params
}

func (h *harness) do(t *testing.T, call request, run func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	httpRequest := httptest.NewRequest(call.method, call.path, bytes.NewBufferString(call.body))
	if call.body != "" {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if call.ifMatch != "" {
		httpRequest.Header.Set("If-Match", call.ifMatch)
	}
	organization := call.organization
	if organization == 0 {
		organization = 1
	}
	role := call.role
	if role == "" {
		role = domain.OrganizationRoleOwner
	}
	ctx := domain.WithOrganizationContext(httpRequest.Context(), domain.OrganizationContext{
		OrganizationID: organization, OrganizationName: "Acme", OrganizationSlug: "acme",
		MembershipID: 1, UserID: 7, Role: role,
	})
	c.Request = httpRequest.WithContext(ctx)
	c.Params = call.params
	c.Set("userID", uint(7))
	run(c)
	c.Writer.WriteHeaderNow()
	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.ErrorCode
}

var firstArticle = gin.Params{{Key: "id", Value: "1"}}

func TestAdminWritesAndEveryMemberReads(t *testing.T) {
	h := newHarness()
	created := h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"  Onboarding  ","body":"# Welcome"}`,
		role: domain.OrganizationRoleAdmin}, h.handler.Create)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	assert.Equal(t, `"knowledge-article-v1"`, created.Header().Get("ETag"))
	assert.Equal(t, "private, no-store", created.Header().Get("Cache-Control"))
	assert.Equal(t, "Onboarding", h.store.articles[1].Title, "titles are trimmed")

	read := h.do(t, request{method: http.MethodGet, path: "/v1/knowledge-articles/1", params: firstArticle, role: domain.OrganizationRoleMember}, h.handler.Get)
	require.Equal(t, http.StatusOK, read.Code)

	denied := h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"Mine"}`,
		role: domain.OrganizationRoleMember}, h.handler.Create)
	assert.Equal(t, http.StatusForbidden, denied.Code)
	assert.Equal(t, "PERMISSION.DENIED", errorCode(t, denied))
}

func TestOtherOrganizationsArticlesAreNotFound(t *testing.T) {
	h := newHarness()
	h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"Secret"}`}, h.handler.Create)
	other := h.do(t, request{method: http.MethodGet, path: "/v1/knowledge-articles/1", params: firstArticle, organization: 2}, h.handler.Get)
	assert.Equal(t, http.StatusNotFound, other.Code)
	assert.Equal(t, CodeArticleNotFound, errorCode(t, other))
}

func TestUpdatesRequireTheVersionThatWasRead(t *testing.T) {
	h := newHarness()
	h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"Draft"}`}, h.handler.Create)

	missing := h.do(t, request{method: http.MethodPatch, path: "/v1/knowledge-articles/1", body: `{"title":"Final"}`, params: firstArticle}, h.handler.Update)
	assert.Equal(t, http.StatusPreconditionRequired, missing.Code)
	assert.Equal(t, CodePreconditionRequired, errorCode(t, missing))

	malformed := h.do(t, request{method: http.MethodPatch, path: "/v1/knowledge-articles/1", body: `{"title":"Final"}`, params: firstArticle, ifMatch: "v1"}, h.handler.Update)
	assert.Equal(t, http.StatusBadRequest, malformed.Code)

	updated := h.do(t, request{method: http.MethodPatch, path: "/v1/knowledge-articles/1", body: `{"title":"Final"}`, params: firstArticle,
		ifMatch: `"knowledge-article-v1"`}, h.handler.Update)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	assert.Equal(t, `"knowledge-article-v2"`, updated.Header().Get("ETag"))

	stale := h.do(t, request{method: http.MethodDelete, path: "/v1/knowledge-articles/1", params: firstArticle, ifMatch: `"knowledge-article-v1"`}, h.handler.Delete)
	assert.Equal(t, http.StatusPreconditionFailed, stale.Code)
	assert.Equal(t, CodeVersionConflict, errorCode(t, stale))

	deleted := h.do(t, request{method: http.MethodDelete, path: "/v1/knowledge-articles/1", params: firstArticle, ifMatch: `"knowledge-article-v2"`}, h.handler.Delete)
	assert.Equal(t, http.StatusNoContent, deleted.Code)
}

func TestInvalidArticlesAndQueriesAreRejected(t *testing.T) {
	h := newHarness()
	blank := h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"   "}`}, h.handler.Create)
	assert.Equal(t, http.StatusUnprocessableEntity, blank.Code)
	assert.Equal(t, CodeInvalidArticle, errorCode(t, blank))

	h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"Draft"}`}, h.handler.Create)
	empty := h.do(t, request{method: http.MethodPatch, path: "/v1/knowledge-articles/1", body: `{}`, params: firstArticle, ifMatch: `"knowledge-article-v1"`}, h.handler.Update)
	assert.Equal(t, CodeInvalidArticle, errorCode(t, empty))

	for _, path := range []string{"/v1/knowledge-articles?cursor=***", "/v1/knowledge-articles?page=2"} {
		recorder := h.do(t, request{method: http.MethodGet, path: path}, h.handler.List)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, path)
	}
	blankQuery := h.do(t, request{method: http.MethodGet, path: "/v1/knowledge-search?q=%20%20"}, h.handler.Search)
	assert.Equal(t, http.StatusBadRequest, blankQuery.Code)
}

func TestListPagesFollowTheCursor(t *testing.T) {
	h := newHarness()
	for _, title := range []string{"one", "two", "three"} {
		h.do(t, request{method: http.MethodPost, path: "/v1/knowledge-articles", body: `{"title":"` + title + `"}`}, h.handler.Create)
	}
	var page struct {
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
		Meta struct {
			NextCursor *string `json:"next_cursor"`
		} `json:"meta"`
	}
	first := h.do(t, request{method: http.MethodGet, path: "/v1/knowledge-articles?per_page=2"}, h.handler.List)
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &page))
	require.Len(t, page.Data, 2)
	assert.Equal(t, "three", page.Data[0].Title)
	require.NotNil(t, page.Meta.NextCursor)

	second := h.do(t, request{method: http.MethodGet, path: "/v1/knowledge-articles?per_page=2&cursor=" + *page.Meta.NextCursor}, h.handler.List)
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &page))
	require.Len(t, page.Data, 1)
	assert.Equal(t, "one", page.Data[0].Title)
	assert.Nil(t, page.Meta.NextCursor)
}

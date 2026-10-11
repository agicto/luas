package comments

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/pkg/response"
)

type harness struct {
	handler   *Handler
	store     *memoryStore
	publisher *fakePublisher
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := newMemoryStore()
	catalog, err := NewResourceCatalog(testResourceType())
	require.NoError(t, err)
	publisher := &fakePublisher{}
	handler := NewHandler(NewService(store, catalog, memberSet{1: {7, 8, 9}, 2: {7}}, publisher))
	handler.RegisterErrorMappings(response.DefaultErrorMapper)
	// The application registers these core mappings at startup.
	response.DefaultErrorMapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
	response.DefaultErrorMapper.Register(domain.ErrInvalidInput, http.StatusBadRequest, domain.CodeInvalidInput)
	return &harness{handler: handler, store: store, publisher: publisher}
}

type request struct {
	method, path, body, ifMatch string
	organization, user          uint
	role                        domain.OrganizationRole
	id                          uint
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
	organization, user, role := call.organization, call.user, call.role
	if organization == 0 {
		organization = 1
	}
	if user == 0 {
		user = 7
	}
	if role == "" {
		role = domain.OrganizationRoleMember
	}
	ctx := domain.WithOrganizationContext(httpRequest.Context(), domain.OrganizationContext{
		OrganizationID: organization, OrganizationName: "Acme", OrganizationSlug: "acme",
		MembershipID: 1, UserID: user, Role: role,
	})
	c.Request = httpRequest.WithContext(ctx)
	if call.id != 0 {
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(call.id), 10)}}
	}
	c.Set("userID", user)
	run(c)
	c.Writer.WriteHeaderNow()
	return recorder
}

func (h *harness) create(t *testing.T, body string, user uint) uint {
	t.Helper()
	recorder := h.do(t, request{method: http.MethodPost, path: "/v1/comments", body: body, user: user}, h.handler.Create)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	var created struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &created))
	return created.Data.ID
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.ErrorCode
}

type commentJSON struct {
	ID        uint    `json:"id"`
	ParentID  *uint   `json:"parent_id"`
	AuthorID  *uint   `json:"author_id"`
	Body      *string `json:"body"`
	Mentions  []uint  `json:"mentions"`
	DeletedAt *string `json:"deleted_at"`
	EditedAt  *string `json:"edited_at"`
}

type threadJSON struct {
	commentJSON    `json:"comment"`
	Replies        []commentJSON `json:"replies"`
	HasMoreReplies bool          `json:"has_more_replies"`
}

type threadPageJSON struct {
	Data []threadJSON `json:"data"`
	Meta struct {
		NextCursor *string `json:"next_cursor"`
	} `json:"meta"`
}

func decodeThreads(t *testing.T, recorder *httptest.ResponseRecorder) threadPageJSON {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var page threadPageJSON
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &page))
	return page
}

type pageJSON struct {
	Data []commentJSON `json:"data"`
	Meta struct {
		NextCursor *string `json:"next_cursor"`
	} `json:"meta"`
}

func decodePage(t *testing.T, recorder *httptest.ResponseRecorder) pageJSON {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var page pageJSON
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &page))
	return page
}

const listPath = "/v1/comments?resource_type=test.doc&resource_id=doc-1"

func TestMembersCommentAndReplyInOneLevelThreads(t *testing.T) {
	h := newHarness(t)
	root := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"  Looks good  "}`, 7)
	reply := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","parent_id":`+strconv.Itoa(int(root))+`,"body":"Agreed"}`, 8)
	// Replying to a reply attaches to the thread root.
	nested := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","parent_id":`+strconv.Itoa(int(reply))+`,"body":"Me too"}`, 9)
	require.NotNil(t, h.store.comments[nested].ParentID)
	assert.Equal(t, root, *h.store.comments[nested].ParentID)
	assert.Equal(t, "Looks good", h.store.comments[root].Body, "bodies are trimmed")

	list := h.do(t, request{method: http.MethodGet, path: listPath}, h.handler.List)
	assert.Equal(t, "private, no-store", list.Header().Get("Cache-Control"))
	page := decodeThreads(t, list)
	require.Len(t, page.Data, 1, "only roots are listed at the top level")
	require.Len(t, page.Data[0].Replies, 2)
	assert.Equal(t, reply, page.Data[0].Replies[0].ID)
	assert.False(t, page.Data[0].HasMoreReplies)
}

func TestThreadsBoundRepliesAndPageThem(t *testing.T) {
	h := newHarness(t)
	root := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"root"}`, 7)
	for index := 0; index < replyPreviewLimit+2; index++ {
		h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","parent_id":`+strconv.Itoa(int(root))+`,"body":"r`+strconv.Itoa(index)+`"}`, 8)
	}
	page := decodeThreads(t, h.do(t, request{method: http.MethodGet, path: listPath}, h.handler.List))
	require.Len(t, page.Data, 1)
	assert.Len(t, page.Data[0].Replies, replyPreviewLimit)
	assert.True(t, page.Data[0].HasMoreReplies)

	var bodies []string
	path := "/v1/comments/1/replies?per_page=2"
	for pages := 0; pages < 5; pages++ {
		replies := decodePage(t, h.do(t, request{method: http.MethodGet, path: path, id: root}, h.handler.ListReplies))
		for _, item := range replies.Data {
			bodies = append(bodies, *item.Body)
		}
		if replies.Meta.NextCursor == nil {
			break
		}
		path = "/v1/comments/1/replies?per_page=2&cursor=" + *replies.Meta.NextCursor
	}
	assert.Equal(t, []string{"r0", "r1", "r2", "r3", "r4"}, bodies)
}

func TestRootsPageFollowsTheCursor(t *testing.T) {
	h := newHarness(t)
	for _, body := range []string{"one", "two", "three"} {
		h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"`+body+`"}`, 7)
	}
	h.create(t, `{"resource_type":"test.doc","resource_id":"doc-2","body":"elsewhere"}`, 7)
	first := decodeThreads(t, h.do(t, request{method: http.MethodGet, path: listPath + "&per_page=2"}, h.handler.List))
	require.Len(t, first.Data, 2)
	assert.Equal(t, "one", *first.Data[0].Body, "threads read oldest first")
	require.NotNil(t, first.Meta.NextCursor)
	second := decodeThreads(t, h.do(t, request{method: http.MethodGet, path: listPath + "&per_page=2&cursor=" + *first.Meta.NextCursor}, h.handler.List))
	require.Len(t, second.Data, 1)
	assert.Equal(t, "three", *second.Data[0].Body)
	assert.Nil(t, second.Meta.NextCursor)

	for _, path := range []string{listPath + "&cursor=***", listPath + "&page=2", "/v1/comments?resource_type=test.doc"} {
		assert.Equal(t, http.StatusBadRequest, h.do(t, request{method: http.MethodGet, path: path}, h.handler.List).Code, path)
	}
}

func TestOtherOrganizationsAndHiddenResourcesAreNotFound(t *testing.T) {
	h := newHarness(t)
	id := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"secret"}`, 7)

	other := h.do(t, request{method: http.MethodGet, path: listPath, organization: 2}, h.handler.List)
	assert.Equal(t, http.StatusNotFound, other.Code)
	assert.Equal(t, CodeResourceNotFound, errorCode(t, other))

	for _, call := range []struct {
		run    func(*gin.Context)
		method string
	}{{h.handler.Delete, http.MethodDelete}, {h.handler.ListReplies, http.MethodGet}} {
		recorder := h.do(t, request{method: call.method, path: "/v1/comments/1", id: id, organization: 2}, call.run)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Equal(t, CodeCommentNotFound, errorCode(t, recorder))
	}
	edit := h.do(t, request{method: http.MethodPatch, path: "/v1/comments/1", id: id, organization: 2, body: `{"body":"x"}`, ifMatch: `"comment-v1"`}, h.handler.Update)
	assert.Equal(t, CodeCommentNotFound, errorCode(t, edit))

	hidden := h.do(t, request{method: http.MethodGet, path: "/v1/comments?resource_type=test.doc&resource_id=hidden-1"}, h.handler.List)
	assert.Equal(t, CodeResourceNotFound, errorCode(t, hidden), "a resource the member may not read is not found")
	visible := h.do(t, request{method: http.MethodGet, path: "/v1/comments?resource_type=test.doc&resource_id=hidden-1", role: domain.OrganizationRoleAdmin}, h.handler.List)
	assert.Equal(t, http.StatusOK, visible.Code)

	unknown := h.do(t, request{method: http.MethodGet, path: "/v1/comments?resource_type=task&resource_id=1"}, h.handler.List)
	assert.Equal(t, http.StatusUnprocessableEntity, unknown.Code)
	assert.Equal(t, CodeUnsupportedResourceType, errorCode(t, unknown))
}

func TestCreateValidatesBodyParentAndMentions(t *testing.T) {
	h := newHarness(t)
	root := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"root"}`, 7)
	cases := map[string]struct {
		body   string
		status int
		code   string
	}{
		"blank body":           {`{"resource_type":"test.doc","resource_id":"doc-1","body":"   "}`, http.StatusUnprocessableEntity, CodeInvalidComment},
		"too long":             {`{"resource_type":"test.doc","resource_id":"doc-1","body":"` + strings.Repeat("a", maxBodyLength+1) + `"}`, http.StatusUnprocessableEntity, "COMMON.VALIDATION_FAILED"},
		"parent elsewhere":     {`{"resource_type":"test.doc","resource_id":"doc-2","parent_id":` + strconv.Itoa(int(root)) + `,"body":"x"}`, http.StatusUnprocessableEntity, CodeInvalidParent},
		"missing parent":       {`{"resource_type":"test.doc","resource_id":"doc-1","parent_id":999,"body":"x"}`, http.StatusUnprocessableEntity, CodeInvalidParent},
		"non-member mention":   {`{"resource_type":"test.doc","resource_id":"doc-1","body":"x","mentions":[42]}`, http.StatusUnprocessableEntity, CodeInvalidMentions},
		"duplicate mention":    {`{"resource_type":"test.doc","resource_id":"doc-1","body":"x","mentions":[8,8]}`, http.StatusUnprocessableEntity, CodeInvalidMentions},
		"closed resource":      {`{"resource_type":"test.doc","resource_id":"locked-1","body":"x"}`, http.StatusForbidden, domain.CodePermissionDenied},
		"malformed resource":   {`{"resource_type":"test.doc","resource_id":"../etc","body":"x"}`, http.StatusBadRequest, domain.CodeInvalidInput},
		"unsupported resource": {`{"resource_type":"invoice","resource_id":"1","body":"x"}`, http.StatusUnprocessableEntity, CodeUnsupportedResourceType},
	}
	for name, tc := range cases {
		recorder := h.do(t, request{method: http.MethodPost, path: "/v1/comments", body: tc.body}, h.handler.Create)
		assert.Equal(t, tc.status, recorder.Code, name)
		assert.Equal(t, tc.code, errorCode(t, recorder), name)
	}
	tooMany := make([]string, maxMentions+1)
	for index := range tooMany {
		tooMany[index] = strconv.Itoa(index + 1)
	}
	recorder := h.do(t, request{method: http.MethodPost, path: "/v1/comments",
		body: `{"resource_type":"test.doc","resource_id":"doc-1","body":"x","mentions":[` + strings.Join(tooMany, ",") + `]}`}, h.handler.Create)
	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
}

func TestOnlyTheAuthorEditsWithTheVersionThatWasRead(t *testing.T) {
	h := newHarness(t)
	id := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"draft"}`, 8)
	call := func(user uint, body, ifMatch string) *httptest.ResponseRecorder {
		return h.do(t, request{method: http.MethodPatch, path: "/v1/comments/1", id: id, user: user, body: body, ifMatch: ifMatch,
			role: domain.OrganizationRoleOwner}, h.handler.Update)
	}

	missing := call(8, `{"body":"final"}`, "")
	assert.Equal(t, http.StatusPreconditionRequired, missing.Code)
	assert.Equal(t, CodePreconditionRequired, errorCode(t, missing))
	assert.Equal(t, http.StatusBadRequest, call(8, `{"body":"final"}`, "v1").Code)

	owner := call(7, `{"body":"final"}`, `"comment-v1"`)
	assert.Equal(t, http.StatusForbidden, owner.Code, "owners moderate by deleting, never by rewriting")

	empty := call(8, `{}`, `"comment-v1"`)
	assert.Equal(t, CodeInvalidComment, errorCode(t, empty))

	updated := call(8, `{"body":"final"}`, `"comment-v1"`)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	assert.Equal(t, `"comment-v2"`, updated.Header().Get("ETag"))
	var edited struct {
		Data commentJSON `json:"data"`
	}
	require.NoError(t, json.Unmarshal(updated.Body.Bytes(), &edited))
	assert.NotNil(t, edited.Data.EditedAt)

	stale := call(8, `{"body":"again"}`, `"comment-v1"`)
	assert.Equal(t, http.StatusPreconditionFailed, stale.Code)
	assert.Equal(t, CodeVersionConflict, errorCode(t, stale))
}

func TestAuthorsAndModeratorsSoftDelete(t *testing.T) {
	h := newHarness(t)
	root := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"root","mentions":[9]}`, 8)
	h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","parent_id":`+strconv.Itoa(int(root))+`,"body":"reply"}`, 9)
	other := h.create(t, `{"resource_type":"test.doc","resource_id":"doc-1","body":"other"}`, 8)

	denied := h.do(t, request{method: http.MethodDelete, path: "/v1/comments/1", id: root, user: 9}, h.handler.Delete)
	assert.Equal(t, http.StatusForbidden, denied.Code, "members delete only their own comments")

	assert.Equal(t, http.StatusNoContent, h.do(t, request{method: http.MethodDelete, path: "/v1/comments/1", id: root, user: 7,
		role: domain.OrganizationRoleAdmin}, h.handler.Delete).Code)
	assert.Equal(t, http.StatusNoContent, h.do(t, request{method: http.MethodDelete, path: "/v1/comments/3", id: other, user: 8}, h.handler.Delete).Code)
	again := h.do(t, request{method: http.MethodDelete, path: "/v1/comments/3", id: other, user: 8}, h.handler.Delete)
	assert.Equal(t, CodeCommentNotFound, errorCode(t, again))

	page := decodeThreads(t, h.do(t, request{method: http.MethodGet, path: listPath}, h.handler.List))
	require.Len(t, page.Data, 2, "deleted comments keep their place")
	assert.Nil(t, page.Data[0].Body)
	assert.Empty(t, page.Data[0].Mentions)
	assert.NotNil(t, page.Data[0].DeletedAt)
	require.Len(t, page.Data[0].Replies, 1, "replies of a deleted root survive")
	assert.Equal(t, "reply", *page.Data[0].Replies[0].Body)

	reply := h.do(t, request{method: http.MethodPost, path: "/v1/comments",
		body: `{"resource_type":"test.doc","resource_id":"doc-1","parent_id":` + strconv.Itoa(int(root)) + `,"body":"late"}`}, h.handler.Create)
	assert.Equal(t, CodeInvalidParent, errorCode(t, reply), "a deleted thread takes no new replies")
	edit := h.do(t, request{method: http.MethodPatch, path: "/v1/comments/1", id: root, user: 8, body: `{"body":"x"}`, ifMatch: `"comment-v2"`}, h.handler.Update)
	assert.Equal(t, CodeCommentNotFound, errorCode(t, edit))
}

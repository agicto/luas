package tag

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
	handler *Handler
	store   *memoryStore
}

func newHarness() *harness {
	gin.SetMode(gin.TestMode)
	store := newMemoryStore()
	handler := NewHandler(NewService(store, newTestResourceTypes()))
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

func (h *harness) create(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	return h.do(t, request{method: http.MethodPost, path: "/v1/tags", body: body}, h.handler.Create)
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.ErrorCode
}

func tagParams(id uint) gin.Params {
	return gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
}

func resourceParams(resourceType, resourceID string) gin.Params {
	return gin.Params{{Key: "resource_type", Value: resourceType}, {Key: "resource_id", Value: resourceID}}
}

type tagList struct {
	Data []struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"data"`
	Meta struct {
		NextCursor *string `json:"next_cursor"`
	} `json:"meta"`
}

func decodeTags(t *testing.T, recorder *httptest.ResponseRecorder) tagList {
	t.Helper()
	var list tagList
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &list), recorder.Body.String())
	return list
}

func TestAdminsWriteTagsAndEveryMemberReads(t *testing.T) {
	h := newHarness()
	created := h.do(t, request{method: http.MethodPost, path: "/v1/tags", body: `{"name":"  Urgent  ","color":"red"}`,
		role: domain.OrganizationRoleAdmin}, h.handler.Create)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	assert.Equal(t, `"tag-v1"`, created.Header().Get("ETag"))
	assert.Equal(t, "private, no-store", created.Header().Get("Cache-Control"))
	assert.Equal(t, "Urgent", h.store.tags[1].Name, "names are trimmed")

	defaulted := h.create(t, `{"name":"Later"}`)
	require.Equal(t, http.StatusCreated, defaulted.Code)
	assert.Equal(t, "gray", h.store.tags[2].Color, "color defaults to gray")

	read := h.do(t, request{method: http.MethodGet, path: "/v1/tags/1", params: tagParams(1), role: domain.OrganizationRoleMember}, h.handler.Get)
	require.Equal(t, http.StatusOK, read.Code)
	assert.Equal(t, `"tag-v1"`, read.Header().Get("ETag"))

	denied := h.do(t, request{method: http.MethodPost, path: "/v1/tags", body: `{"name":"Mine"}`, role: domain.OrganizationRoleMember}, h.handler.Create)
	assert.Equal(t, http.StatusForbidden, denied.Code)
	assert.Equal(t, "PERMISSION.DENIED", errorCode(t, denied))

	assign := h.do(t, request{method: http.MethodPut, path: "/v1/resource-tags/test_resource/1", body: `{"tag_ids":[1]}`,
		params: resourceParams(testResourceType, "1"), role: domain.OrganizationRoleMember}, h.handler.ReplaceResourceTags)
	assert.Equal(t, http.StatusForbidden, assign.Code, "members do not change assignments")
}

func TestTagNamesAreUniquePerOrganizationIgnoringCase(t *testing.T) {
	h := newHarness()
	require.Equal(t, http.StatusCreated, h.create(t, `{"name":"Bug"}`).Code)
	taken := h.create(t, `{"name":"bUG"}`)
	assert.Equal(t, http.StatusConflict, taken.Code)
	assert.Equal(t, CodeNameTaken, errorCode(t, taken))

	other := h.do(t, request{method: http.MethodPost, path: "/v1/tags", body: `{"name":"bug"}`, organization: 2}, h.handler.Create)
	assert.Equal(t, http.StatusCreated, other.Code, "another organization may use the same name")

	require.Equal(t, http.StatusCreated, h.create(t, `{"name":"Feature"}`).Code)
	renamed := h.do(t, request{method: http.MethodPatch, path: "/v1/tags/3", body: `{"name":"BUG"}`, params: tagParams(3), ifMatch: `"tag-v1"`}, h.handler.Update)
	assert.Equal(t, http.StatusConflict, renamed.Code)
}

func TestOtherOrganizationsTagsAreNotFound(t *testing.T) {
	h := newHarness()
	h.create(t, `{"name":"Secret"}`)
	for _, call := range []struct {
		method string
		run    func(*gin.Context)
		body   string
	}{
		{http.MethodGet, h.handler.Get, ""},
		{http.MethodGet, h.handler.Resources, ""},
		{http.MethodPatch, h.handler.Update, `{"name":"Mine"}`},
		{http.MethodDelete, h.handler.Delete, ""},
	} {
		recorder := h.do(t, request{method: call.method, path: "/v1/tags/1", body: call.body, params: tagParams(1),
			organization: 2, ifMatch: `"tag-v1"`}, call.run)
		assert.Equal(t, http.StatusNotFound, recorder.Code, call.method)
		assert.Equal(t, CodeNotFound, errorCode(t, recorder), call.method)
	}

	// Organization 2 cannot see organization 1's resources or attach organization 1's tags.
	foreign := h.do(t, request{method: http.MethodPut, path: "/v1/resource-tags/test_resource/1", body: `{"tag_ids":[]}`,
		params: resourceParams(testResourceType, "1"), organization: 2}, h.handler.ReplaceResourceTags)
	assert.Equal(t, http.StatusNotFound, foreign.Code)
	assert.Equal(t, CodeResourceNotFound, errorCode(t, foreign))

	h.do(t, request{method: http.MethodPost, path: "/v1/tags", body: `{"name":"Theirs"}`, organization: 2}, h.handler.Create)
	crossTag := h.do(t, request{method: http.MethodPut, path: "/v1/resource-tags/test_resource/1", body: `{"tag_ids":[2]}`,
		params: resourceParams(testResourceType, "1")}, h.handler.ReplaceResourceTags)
	assert.Equal(t, http.StatusUnprocessableEntity, crossTag.Code)
	assert.Equal(t, CodeInvalidAssignment, errorCode(t, crossTag))
}

func TestTagUpdatesRequireTheVersionThatWasRead(t *testing.T) {
	h := newHarness()
	h.create(t, `{"name":"Draft"}`)

	missing := h.do(t, request{method: http.MethodPatch, path: "/v1/tags/1", body: `{"color":"blue"}`, params: tagParams(1)}, h.handler.Update)
	assert.Equal(t, http.StatusPreconditionRequired, missing.Code)
	assert.Equal(t, CodePreconditionRequired, errorCode(t, missing))

	malformed := h.do(t, request{method: http.MethodPatch, path: "/v1/tags/1", body: `{"color":"blue"}`, params: tagParams(1), ifMatch: "v1"}, h.handler.Update)
	assert.Equal(t, http.StatusBadRequest, malformed.Code)

	updated := h.do(t, request{method: http.MethodPatch, path: "/v1/tags/1", body: `{"color":"blue","description":"Work in progress"}`,
		params: tagParams(1), ifMatch: `"tag-v1"`}, h.handler.Update)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	assert.Equal(t, `"tag-v2"`, updated.Header().Get("ETag"))
	assert.Equal(t, "blue", h.store.tags[1].Color)

	stale := h.do(t, request{method: http.MethodDelete, path: "/v1/tags/1", params: tagParams(1), ifMatch: `"tag-v1"`}, h.handler.Delete)
	assert.Equal(t, http.StatusPreconditionFailed, stale.Code)
	assert.Equal(t, CodeVersionConflict, errorCode(t, stale))

	deleted := h.do(t, request{method: http.MethodDelete, path: "/v1/tags/1", params: tagParams(1), ifMatch: `"tag-v2"`}, h.handler.Delete)
	assert.Equal(t, http.StatusNoContent, deleted.Code)
}

func TestInvalidTagsAndQueriesAreRejected(t *testing.T) {
	h := newHarness()
	for _, body := range []string{`{"name":"   "}`, `{"name":"Ok","color":"teal"}`} {
		recorder := h.create(t, body)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code, body)
		assert.Equal(t, CodeInvalid, errorCode(t, recorder), body)
	}
	tooLong := h.create(t, `{"name":"`+string(bytes.Repeat([]byte("x"), 51))+`"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, tooLong.Code)
	assert.Equal(t, "COMMON.VALIDATION_FAILED", errorCode(t, tooLong))

	h.create(t, `{"name":"Draft"}`)
	empty := h.do(t, request{method: http.MethodPatch, path: "/v1/tags/1", body: `{}`, params: tagParams(1), ifMatch: `"tag-v1"`}, h.handler.Update)
	assert.Equal(t, CodeInvalid, errorCode(t, empty))

	for _, path := range []string{"/v1/tags?cursor=***", "/v1/tags?page=2", "/v1/tags?cursor=" + string(bytes.Repeat([]byte("A"), 300))} {
		recorder := h.do(t, request{method: http.MethodGet, path: path}, h.handler.List)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, path)
	}
	for _, path := range []string{"/v1/tags/1/resources?resource_type=Bad-Type", "/v1/tags/1/resources?cursor=MA"} {
		recorder := h.do(t, request{method: http.MethodGet, path: path, params: tagParams(1)}, h.handler.Resources)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, path)
	}
	malformedRef := h.do(t, request{method: http.MethodGet, path: "/v1/resource-tags/Test/1", params: resourceParams("Test", "1")}, h.handler.ResourceTags)
	assert.Equal(t, http.StatusBadRequest, malformedRef.Code)
}

func TestTagListPagesFollowTheCursorInNameOrder(t *testing.T) {
	h := newHarness()
	for _, name := range []string{"charlie", "Alpha", "bravo"} {
		h.create(t, `{"name":"`+name+`"}`)
	}
	first := decodeTags(t, h.do(t, request{method: http.MethodGet, path: "/v1/tags?per_page=2"}, h.handler.List))
	require.Len(t, first.Data, 2)
	assert.Equal(t, "Alpha", first.Data[0].Name)
	assert.Equal(t, "bravo", first.Data[1].Name)
	require.NotNil(t, first.Meta.NextCursor)

	second := decodeTags(t, h.do(t, request{method: http.MethodGet, path: "/v1/tags?per_page=2&cursor=" + *first.Meta.NextCursor}, h.handler.List))
	require.Len(t, second.Data, 1)
	assert.Equal(t, "charlie", second.Data[0].Name)
	assert.Nil(t, second.Meta.NextCursor)
}

func TestReplacingResourceTagsIsIdempotentAndBounded(t *testing.T) {
	h := newHarness()
	for _, name := range []string{"one", "two", "three"} {
		h.create(t, `{"name":"`+name+`"}`)
	}
	put := func(resourceID, body string) *httptest.ResponseRecorder {
		return h.do(t, request{method: http.MethodPut, path: "/v1/resource-tags/test_resource/" + resourceID, body: body,
			params: resourceParams(testResourceType, resourceID)}, h.handler.ReplaceResourceTags)
	}

	first := put("1", `{"tag_ids":[1,2]}`)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	again := put("1", `{"tag_ids":[1,2]}`)
	require.Equal(t, http.StatusOK, again.Code)
	assert.JSONEq(t, first.Body.String(), again.Body.String(), "repeating the request changes nothing")
	assert.Len(t, h.store.assignments, 2)

	replaced := decodeTags(t, put("1", `{"tag_ids":[3,2]}`))
	assert.Equal(t, []string{"three", "two"}, []string{replaced.Data[0].Name, replaced.Data[1].Name}, "tags are ordered by name")

	read := decodeTags(t, h.do(t, request{method: http.MethodGet, path: "/v1/resource-tags/test_resource/1",
		params: resourceParams(testResourceType, "1"), role: domain.OrganizationRoleMember}, h.handler.ResourceTags))
	assert.Len(t, read.Data, 2)

	cleared := decodeTags(t, put("1", `{"tag_ids":[]}`))
	assert.Empty(t, cleared.Data)

	missingField := put("1", `{}`)
	assert.Equal(t, "COMMON.VALIDATION_FAILED", errorCode(t, missingField), "an absent tag set is not an empty one")
	duplicate := put("1", `{"tag_ids":[1,1]}`)
	assert.Equal(t, http.StatusUnprocessableEntity, duplicate.Code)
	ids := make([]string, 21)
	for index := range ids {
		ids[index] = strconv.Itoa(index + 1)
	}
	tooMany := put("1", `{"tag_ids":[`+strings.Join(ids, ",")+`]}`)
	assert.Equal(t, http.StatusUnprocessableEntity, tooMany.Code)

	unknownType := h.do(t, request{method: http.MethodGet, path: "/v1/resource-tags/unknown_type/1",
		params: resourceParams("unknown_type", "1")}, h.handler.ResourceTags)
	assert.Equal(t, http.StatusNotFound, unknownType.Code)
	assert.Equal(t, CodeResourceNotFound, errorCode(t, unknownType))
	missingResource := put("0", `{"tag_ids":[1]}`)
	assert.Equal(t, CodeResourceNotFound, errorCode(t, missingResource))
}

func TestTaggedResourcesPageNewestFirstAndFilterByType(t *testing.T) {
	h := newHarness()
	h.create(t, `{"name":"shared"}`)
	for _, resourceID := range []string{"1", "2", "3"} {
		recorder := h.do(t, request{method: http.MethodPut, path: "/v1/resource-tags/test_resource/" + resourceID, body: `{"tag_ids":[1]}`,
			params: resourceParams(testResourceType, resourceID)}, h.handler.ReplaceResourceTags)
		require.Equal(t, http.StatusOK, recorder.Code)
	}
	var page struct {
		Data []struct {
			ResourceType string `json:"resource_type"`
			ResourceID   string `json:"resource_id"`
		} `json:"data"`
		Meta struct {
			NextCursor *string `json:"next_cursor"`
		} `json:"meta"`
	}
	first := h.do(t, request{method: http.MethodGet, path: "/v1/tags/1/resources?per_page=2&resource_type=test_resource",
		params: tagParams(1), role: domain.OrganizationRoleMember}, h.handler.Resources)
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &page))
	require.Len(t, page.Data, 2)
	assert.Equal(t, "3", page.Data[0].ResourceID)
	require.NotNil(t, page.Meta.NextCursor)

	second := h.do(t, request{method: http.MethodGet, path: "/v1/tags/1/resources?per_page=2&cursor=" + *page.Meta.NextCursor,
		params: tagParams(1)}, h.handler.Resources)
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &page))
	require.Len(t, page.Data, 1)
	assert.Equal(t, "1", page.Data[0].ResourceID)
	assert.Nil(t, page.Meta.NextCursor)

	other := h.do(t, request{method: http.MethodGet, path: "/v1/tags/1/resources?resource_type=other_type", params: tagParams(1)}, h.handler.Resources)
	require.NoError(t, json.Unmarshal(other.Body.Bytes(), &page))
	assert.Empty(t, page.Data)
}

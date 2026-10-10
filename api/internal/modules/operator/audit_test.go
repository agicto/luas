package operator

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeAuditQuery struct {
	filter domain.AuditLogFilter
	after  *domain.AuditLogCursor
	limit  int
	err    error
}

func (f *fakeAuditQuery) ListAuditLogsAfter(_ context.Context, filter domain.AuditLogFilter, after *domain.AuditLogCursor, limit int) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	f.filter, f.after, f.limit = filter, after, limit
	if f.err != nil {
		return nil, nil, f.err
	}
	items := []*domain.AuditLog{{ID: 9, ActorType: "user", Action: "disable", Resource: "users", Method: "POST", Path: "/v1/operator/users/:id/disable", StatusCode: 200}}
	return items, &domain.AuditLogCursor{CreatedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), ID: 9}, nil
}

func (f *fakeAuditQuery) ListAuditLogs(_ context.Context, filter domain.AuditLogFilter, _, _ int) ([]*domain.AuditLog, int64, error) {
	f.filter = filter
	if f.err != nil {
		return nil, 0, f.err
	}
	return []*domain.AuditLog{{ID: 9, ActorType: "user", Action: "disable", Resource: "users", Method: "POST", Path: "/v1/operator/users/:id/disable", StatusCode: 200}}, 1, nil
}

func TestListAuditLogsPassesFiltersAndReturnsAuditShape(t *testing.T) {
	f := newFixture(t)

	recorder := f.do(t, call{method: http.MethodGet, cookie: testCredential,
		path: "/v1/operator/audit-logs?user_id=2&action=disable&from=2026-09-01T00:00:00Z&to=2026-09-30T00:00:00%2B08:00"})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, f.audit.filter.UserID)
	assert.EqualValues(t, 2, *f.audit.filter.UserID)
	assert.Equal(t, "disable", f.audit.filter.Action)
	assert.True(t, f.audit.filter.From.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)))
	assert.True(t, f.audit.filter.To.Equal(time.Date(2026, 9, 29, 16, 0, 0, 0, time.UTC)))
	assert.Contains(t, recorder.Body.String(), `"path":"/v1/operator/users/:id/disable"`)
	assert.Contains(t, recorder.Body.String(), `"meta"`)
}

func TestListAuditLogsRejectsInvalidRanges(t *testing.T) {
	f := newFixture(t)
	recorder := f.do(t, call{method: http.MethodGet, cookie: testCredential, path: "/v1/operator/audit-logs?from=yesterday"})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, domain.CodeInvalidInput, errorCode(t, recorder))

	f.audit.err = domain.ErrInvalidInput
	recorder = f.do(t, call{method: http.MethodGet, cookie: testCredential, path: "/v1/operator/audit-logs"})
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "service range violations are input errors")
}

func TestListAuditLogsRequiresOperator(t *testing.T) {
	f := newFixture(t)
	delete(f.grants.operators, 1)
	recorder := f.do(t, call{method: http.MethodGet, cookie: testCredential, path: "/v1/operator/audit-logs"})
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, domain.CodeOperatorForbidden, errorCode(t, recorder))
}

func TestListAuditLogsKeysetModeSkipsTheCountAndReturnsACursor(t *testing.T) {
	f := newFixture(t)

	first := f.do(t, call{method: http.MethodGet, cookie: testCredential, path: "/v1/operator/audit-logs?cursor=&per_page=25"})
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	var body struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			PerPage    int     `json:"per_page"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
			Total      *int    `json:"total"`
		} `json:"meta"`
		Links any `json:"links"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &body))
	assert.Nil(t, f.audit.after, "an empty cursor starts at the newest record")
	assert.Equal(t, 25, f.audit.limit)
	assert.Len(t, body.Data, 1)
	assert.Equal(t, 25, body.Meta.PerPage)
	assert.True(t, body.Meta.HasMore)
	require.NotNil(t, body.Meta.NextCursor)
	assert.Nil(t, body.Meta.Total, "keyset pages carry no total")
	assert.Nil(t, body.Links)

	next := f.do(t, call{method: http.MethodGet, cookie: testCredential, path: "/v1/operator/audit-logs?cursor=" + *body.Meta.NextCursor})
	require.Equal(t, http.StatusOK, next.Code, next.Body.String())
	require.NotNil(t, f.audit.after)
	assert.Equal(t, uint(9), f.audit.after.ID)
	assert.True(t, f.audit.after.CreatedAt.Equal(time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)))
}

func TestListAuditLogsRejectsMalformedOrMixedCursors(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{
		"/v1/operator/audit-logs?cursor=not*base64",
		"/v1/operator/audit-logs?cursor=&page=2",
		"/v1/operator/audit-logs?cursor=a&cursor=b",
	} {
		recorder := f.do(t, call{method: http.MethodGet, cookie: testCredential, path: path})
		assert.Equal(t, http.StatusBadRequest, recorder.Code, path)
		assert.Equal(t, "COMMON.INVALID_INPUT", errorCode(t, recorder), path)
	}
}

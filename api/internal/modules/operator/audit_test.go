package operator

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeAuditQuery struct {
	filter domain.AuditLogFilter
	err    error
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

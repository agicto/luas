package operator

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeAdmin struct {
	users        map[uint]*domain.User
	lastFilter   domain.UserListFilter
	revokedFor   []uint
	statusWrites int
}

func newFakeAdmin(users ...*domain.User) *fakeAdmin {
	admin := &fakeAdmin{users: map[uint]*domain.User{}}
	for _, user := range users {
		copied := *user
		admin.users[user.ID] = &copied
	}
	return admin
}

func (f *fakeAdmin) ListUsers(_ context.Context, filter domain.UserListFilter, _, _ int) ([]*domain.User, int64, error) {
	f.lastFilter = filter
	result := []*domain.User{}
	for _, id := range []uint{2, 1} {
		if user, ok := f.users[id]; ok {
			result = append(result, user)
		}
	}
	return result, int64(len(result)), nil
}

func (f *fakeAdmin) GetUser(_ context.Context, id uint) (*domain.User, error) {
	if user, ok := f.users[id]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeAdmin) SetUserActive(_ context.Context, id uint, active bool) (*domain.User, error) {
	f.statusWrites++
	user := f.users[id]
	user.Status = 0
	if active {
		user.Status = 1
	}
	return user, nil
}

func (f *fakeAdmin) RevokeUserSessions(_ context.Context, id uint) error {
	f.revokedFor = append(f.revokedFor, id)
	return nil
}

func postCall(path string) call {
	return call{method: http.MethodPost, path: path, cookie: testCredential, origin: testOrigin, csrf: csrfToken(testCredential)}
}

func TestListUsersMarksOperatorsAndPassesFilters(t *testing.T) {
	f := newFixture(t)

	recorder := f.do(t, call{method: http.MethodGet, path: "/v1/operator/users?q=mem&status=disabled", cookie: testCredential})

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var body struct {
		Data []managedUserResponse `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	assert.Equal(t, "member", body.Data[0].Username)
	assert.False(t, body.Data[0].IsOperator)
	assert.True(t, body.Data[1].IsOperator)
	assert.Equal(t, 2, body.Meta.Total)
	assert.Equal(t, "mem", f.admin.lastFilter.Query)
	require.NotNil(t, f.admin.lastFilter.Active)
	assert.False(t, *f.admin.lastFilter.Active)
	assert.NotContains(t, recorder.Body.String(), "password")
}

func TestListUsersRejectsInvalidQueries(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{
		"/v1/operator/users?status=deleted",
		"/v1/operator/users?q=" + strings.Repeat("a", 101),
	} {
		recorder := f.do(t, call{method: http.MethodGet, path: path, cookie: testCredential})
		assert.Equal(t, http.StatusBadRequest, recorder.Code, path)
		assert.Equal(t, domain.CodeInvalidInput, errorCode(t, recorder))
	}
}

func TestDisableAndEnableUser(t *testing.T) {
	f := newFixture(t)

	recorder := f.do(t, postCall("/v1/operator/users/2/disable"))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Contains(t, recorder.Body.String(), `"status":"disabled"`)

	recorder = f.do(t, postCall("/v1/operator/users/2/enable"))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"status":"active"`)
	assert.Equal(t, 2, f.admin.statusWrites)
}

func TestOperatorAccountsAreProtectedTargets(t *testing.T) {
	f := newFixture(t)
	f.admin.users[3] = &domain.User{ID: 3, Username: "other-ops", Status: 1}
	f.grants.operators[3] = true

	for _, path := range []string{
		"/v1/operator/users/1/disable",         // the caller's own account
		"/v1/operator/users/3/disable",         // another operator
		"/v1/operator/users/3/enable",          // enabling is also a CLI decision
		"/v1/operator/users/3/sessions/revoke", // ending an operator's sessions
	} {
		recorder := f.do(t, postCall(path))
		assert.Equal(t, http.StatusConflict, recorder.Code, path)
		assert.Equal(t, domain.CodeOperatorTargetProtected, errorCode(t, recorder), path)
	}
	assert.Zero(t, f.admin.statusWrites)
	assert.Empty(t, f.admin.revokedFor)
}

func TestUserMutationsRequireCSRFAndExistingTarget(t *testing.T) {
	f := newFixture(t)

	recorder := f.do(t, call{method: http.MethodPost, path: "/v1/operator/users/2/disable", cookie: testCredential, origin: testOrigin})
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, domain.CodeOperatorCSRFRejected, errorCode(t, recorder))

	recorder = f.do(t, postCall("/v1/operator/users/404/disable"))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, domain.CodeUserNotFound, errorCode(t, recorder))

	recorder = f.do(t, postCall("/v1/operator/users/abc/disable"))
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	recorder = f.do(t, postCall("/v1/operator/users/2/sessions/revoke"))
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, []uint{2}, f.admin.revokedFor)
	assert.Zero(t, f.admin.statusWrites)
}

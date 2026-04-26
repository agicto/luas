package feature

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuditLogsCaptureProfileUpdates(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	email := fmt.Sprintf("audit_%d@example.com", rand.Intn(100000))
	password := "password123"

	tc := NewTestCase(t)

	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "audituser",
			"email":    email,
			"password": password,
		}).
		Call().
		AssertCreated()

	loginJSON := tc.Post("/v1/login").
		WithJSON(map[string]any{
			"username": email,
			"password": password,
		}).
		Call().
		AssertOk().
		JSON()

	loginData := loginJSON["data"].(map[string]interface{})
	token := loginData["access_token"].(string)

	tc.Put("/v1/users/profile").
		WithToken(token).
		WithJSON(map[string]any{
			"nickname": "audited-profile",
		}).
		Call().
		AssertOk()

	auditJSON := tc.Get("/v1/audit-logs").
		WithToken(token).
		Call().
		AssertOk().
		JSON()

	data, ok := auditJSON["data"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, data)

	entry, ok := data[0].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "users.profile", entry["resource"])
	require.Equal(t, "update", entry["action"])
	require.Equal(t, "PUT", entry["method"])
	require.Equal(t, "users.profile.update", entry["route_name"])
}

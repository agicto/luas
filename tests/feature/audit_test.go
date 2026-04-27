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

	registerJSON := tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "audituser",
			"email":    email,
			"password": password,
		}).
		Call().
		AssertCreated().
		JSON()

	registerData := registerJSON["data"].(map[string]interface{})
	userID := registerData["id"].(float64)

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
	require.Equal(t, "user", entry["target_type"])
	require.Equal(t, fmt.Sprintf("%.0f", userID), entry["target_id"])
	require.Equal(t, "success", entry["result"])

	changes, ok := entry["changes"].(map[string]interface{})
	require.True(t, ok)
	nickname, ok := changes["nickname"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "", nickname["before"])
	require.Equal(t, "audited-profile", nickname["after"])
}

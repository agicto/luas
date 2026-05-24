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

func TestAuditLogsCaptureAPIKeyLifecycle(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	email := fmt.Sprintf("audit_api_key_%d@example.com", rand.Intn(100000))
	password := "password123"

	tc := NewTestCase(t)

	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "auditapikeyuser",
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

	createJSON := tc.Post("/v1/api-keys").
		WithToken(token).
		WithJSON(map[string]any{
			"name":   "Automation Key",
			"scopes": []string{"models:invoke", "models:read"},
		}).
		Call().
		AssertCreated().
		JSON()

	createData := createJSON["data"].(map[string]interface{})
	createdKey := createData["api_key"].(map[string]interface{})
	apiKeyID := fmt.Sprintf("%.0f", createdKey["id"].(float64))

	tc.Delete("/v1/api-keys/" + apiKeyID).
		WithToken(token).
		Call().
		AssertNoContent()

	auditJSON := tc.Get("/v1/audit-logs").
		WithToken(token).
		Call().
		AssertOk().
		JSON()

	data, ok := auditJSON["data"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(data), 2)

	revokeEntry, ok := data[0].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "api_keys", revokeEntry["resource"])
	require.Equal(t, "revoke", revokeEntry["action"])
	require.Equal(t, "api_key", revokeEntry["target_type"])
	require.Equal(t, apiKeyID, revokeEntry["target_id"])
	require.Equal(t, "success", revokeEntry["result"])

	revokeChanges, ok := revokeEntry["changes"].(map[string]interface{})
	require.True(t, ok)
	revokedAt, ok := revokeChanges["revoked_at"].(map[string]interface{})
	require.True(t, ok)
	require.NotEmpty(t, revokedAt["after"])

	createEntry, ok := data[1].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "api_keys", createEntry["resource"])
	require.Equal(t, "create", createEntry["action"])
	require.Equal(t, "api_key", createEntry["target_type"])
	require.Equal(t, apiKeyID, createEntry["target_id"])
	require.Equal(t, "success", createEntry["result"])

	createChanges, ok := createEntry["changes"].(map[string]interface{})
	require.True(t, ok)

	nameChange, ok := createChanges["name"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "Automation Key", nameChange["after"])

	scopeChange, ok := createChanges["scopes"].(map[string]interface{})
	require.True(t, ok)
	scopeValues, ok := scopeChange["after"].([]interface{})
	require.True(t, ok)
	require.ElementsMatch(t, []interface{}{"models:invoke", "models:read"}, scopeValues)
}

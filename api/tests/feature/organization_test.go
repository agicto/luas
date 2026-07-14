package feature

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
)

func TestOrganizationRoutesAreAbsentByDefault(t *testing.T) {
	NewTestCase(t).
		Get("/v1/organizations").
		Call().
		AssertNotFound()
}

func TestOrganizationOwnershipKernel(t *testing.T) {
	tc := NewTestCaseWithOptionalStarters(t, "organization")
	ownerToken := registerAndLoginOrganizationUser(t, tc, "orgowner", "owner@example.com")
	otherToken := registerAndLoginOrganizationUser(t, tc, "orgother", "other@example.com")

	createJSON := tc.Post("/v1/organizations").
		WithToken(ownerToken).
		WithJSON(map[string]any{
			"name": "Acme Europe",
			"slug": "acme-europe",
		}).
		Call().
		AssertCreated().
		AssertJSONPath("data.role", "owner").
		AssertJSONPath("data.slug", "acme-europe").
		JSON()

	created := createJSON["data"].(map[string]interface{})
	organizationID := fmt.Sprintf("%.0f", created["id"].(float64))

	tc.Post("/v1/organizations").
		WithToken(ownerToken).
		WithJSON(map[string]any{
			"name": "Duplicate Slug",
			"slug": "acme-europe",
		}).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.SLUG_ALREADY_EXISTS")

	tc.Get("/v1/organizations").
		WithToken(ownerToken).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(1))

	tc.Get("/v1/organizations/"+organizationID).
		WithToken(ownerToken).
		Call().
		AssertOk().
		AssertJSONPath("data.name", "Acme Europe")

	tc.Patch("/v1/organizations/"+organizationID).
		WithToken(ownerToken).
		WithJSON(map[string]any{"name": "Acme Ireland"}).
		Call().
		AssertOk().
		AssertJSONPath("data.name", "Acme Ireland")

	tc.Get("/v1/organizations/"+organizationID).
		WithToken(otherToken).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.NOT_FOUND")

	tc.Post("/v1/organizations").
		WithToken(ownerToken).
		WithJSON(map[string]any{
			"name": "Invalid Slug",
			"slug": "Not Valid",
		}).
		Call().
		AssertUnprocessable().
		AssertJSONPath("error_code", "COMMON.VALIDATION_FAILED")

	tc.Delete("/v1/users/account").
		WithToken(ownerToken).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED")
}

func registerAndLoginOrganizationUser(t *testing.T, tc *testplatform.TestCase, username, email string) string {
	t.Helper()
	password := "password123"
	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": username,
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
	data, ok := loginJSON["data"].(map[string]interface{})
	require.True(t, ok)
	token, ok := data["access_token"].(string)
	require.True(t, ok)
	return token
}

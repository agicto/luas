package feature

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/config"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
)

var organizationInvitationTokenPattern = regexp.MustCompile(`oinv_[a-z0-9_-]+\.[a-f0-9]{48}`)

type organizationEmailTransport struct {
	invitationTokens chan<- string
}

func (transport *organizationEmailTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	defer request.Body.Close()
	var payload struct {
		Subject string `json:"subject"`
		HTML    string `json:"html"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode feature email request: %w", err)
	}
	if payload.Subject == "Organization invitation" {
		token := organizationInvitationTokenPattern.FindString(payload.HTML)
		if token == "" {
			return nil, fmt.Errorf("organization invitation email has no token")
		}
		transport.invitationTokens <- token
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"email_feature_test"}`)),
		Request:    request,
	}, nil
}

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

func TestOrganizationInvitationHTTPContract(t *testing.T) {
	invitationTokens := make(chan string, 4)
	previousTransport := http.DefaultTransport
	http.DefaultTransport = &organizationEmailTransport{invitationTokens: invitationTokens}
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	engine := setupApp(func(cfg *config.Config) {
		cfg.Email.From = "Luas <noreply@example.com>"
		cfg.Email.ResendAPIKey = "feature-test-key"
		cfg.Email.RequestTimeout = time.Second
	}, "organization")
	tc := testplatform.NewTestCase(t, engine)
	ownerToken := registerAndLoginOrganizationUser(t, tc, "inviteowner", "invite-owner@example.com")
	inviteeToken := registerAndLoginOrganizationUser(t, tc, "invitee", "Member@Example.com")

	createJSON := tc.Post("/v1/organizations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"name": "Invitation Org", "slug": "invitation-org"}).
		Call().
		AssertCreated().
		JSON()
	organization := createJSON["data"].(map[string]interface{})
	organizationID := fmt.Sprintf("%.0f", organization["id"].(float64))

	inviteJSON := tc.Post("/v1/organizations/"+organizationID+"/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{
			"email": "  MEMBER@example.com ",
			"role":  "member",
		}).
		Call().
		AssertCreated().
		AssertJSONPath("data.email_send_status", "accepted_by_provider").
		AssertJSONPath("data.invitation.email", "member@example.com").
		AssertJSONPath("data.invitation.status", "pending").
		JSON()
	inviteData := inviteJSON["data"].(map[string]interface{})
	invitation := inviteData["invitation"].(map[string]interface{})
	invitationID := fmt.Sprintf("%.0f", invitation["id"].(float64))
	require.NotContains(t, invitation, "token")
	require.NotContains(t, invitation, "token_hash")
	invitationToken := <-invitationTokens

	tc.Get("/v1/organizations/"+organizationID+"/invitations").
		WithToken(ownerToken).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(1))

	tc.Post("/v1/organizations/"+organizationID+"/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"email": "member@example.com", "role": "member"}).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.INVITATION.ALREADY_PENDING")

	tc.Post("/v1/organizations/"+organizationID+"/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"email": "another@example.com", "role": "owner"}).
		Call().
		AssertUnprocessable().
		AssertJSONPath("error_code", "COMMON.VALIDATION_FAILED")

	tc.Post("/v1/organization-invitations/accept").
		WithToken(inviteeToken).
		WithJSON(map[string]any{"token": "oinv_unknown.secret"}).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.INVITATION.INVALID")

	tc.Post("/v1/organization-invitations/accept").
		WithToken(ownerToken).
		WithJSON(map[string]any{"token": invitationToken}).
		Call().
		AssertStatus(403).
		AssertJSONPath("error_code", "ORGANIZATION.INVITATION.EMAIL_MISMATCH")

	tc.Post("/v1/organization-invitations/accept").
		WithToken(inviteeToken).
		WithJSON(map[string]any{"token": invitationToken}).
		Call().
		AssertOk().
		AssertJSONPath("data.id", organization["id"]).
		AssertJSONPath("data.role", "member")

	tc.Post("/v1/organization-invitations/accept").
		WithToken(inviteeToken).
		WithJSON(map[string]any{"token": invitationToken}).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.INVITATION.INVALID")

	listJSON := tc.Get("/v1/organizations/"+organizationID+"/invitations").
		WithToken(ownerToken).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(1)).
		JSON()
	listedInvitations, ok := listJSON["data"].([]interface{})
	require.True(t, ok)
	require.Len(t, listedInvitations, 1)
	require.Equal(t, "accepted", listedInvitations[0].(map[string]interface{})["status"])

	tc.Post("/v1/organizations/"+organizationID+"/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"email": "member@example.com", "role": "member"}).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.MEMBER_ALREADY_EXISTS")

	tc.Delete("/v1/organizations/"+organizationID+"/invitations/"+invitationID).
		WithToken(ownerToken).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.INVITATION.NOT_FOUND")

	revocableJSON := tc.Post("/v1/organizations/" + organizationID + "/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"email": "another@example.com", "role": "admin"}).
		Call().
		AssertCreated().
		JSON()
	revocableData := revocableJSON["data"].(map[string]interface{})
	revocable := revocableData["invitation"].(map[string]interface{})
	revocableID := fmt.Sprintf("%.0f", revocable["id"].(float64))

	tc.Delete("/v1/organizations/" + organizationID + "/invitations/" + revocableID).
		WithToken(ownerToken).
		Call().
		AssertNoContent()
	tc.Delete("/v1/organizations/"+organizationID+"/invitations/"+revocableID).
		WithToken(ownerToken).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.INVITATION.NOT_FOUND")

	tc.Post("/v1/organizations/" + organizationID + "/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"email": "another@example.com", "role": "member"}).
		Call().
		AssertCreated()
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

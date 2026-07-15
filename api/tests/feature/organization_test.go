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

func TestOrganizationMemberLifecycleHTTPContract(t *testing.T) {
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
	owner := registerOrganizationTestUser(t, tc, "lifecycle-owner", "lifecycle-owner@example.com")
	admin := registerOrganizationTestUser(t, tc, "lifecycle-admin", "lifecycle-admin@example.com")
	member := registerOrganizationTestUser(t, tc, "lifecycle-member", "lifecycle-member@example.com")
	outsider := registerOrganizationTestUser(t, tc, "lifecycle-outsider", "lifecycle-outsider@example.com")

	created := tc.Post("/v1/organizations").
		WithToken(owner.Token).
		WithJSON(map[string]any{"name": "Lifecycle Org", "slug": "lifecycle-org"}).
		Call().
		AssertCreated().
		JSON()["data"].(map[string]interface{})
	organizationID := fmt.Sprintf("%.0f", created["id"].(float64))

	inviteAndAcceptOrganizationMember(t, tc, owner.Token, admin, organizationID, "admin", invitationTokens)
	inviteAndAcceptOrganizationMember(t, tc, owner.Token, member, organizationID, "member", invitationTokens)

	membersJSON := tc.Get("/v1/organizations/"+organizationID+"/members").
		WithToken(member.Token).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(3)).
		JSON()
	members, ok := membersJSON["data"].([]interface{})
	require.True(t, ok)
	require.Len(t, members, 3)
	membershipIDs := make(map[uint]uint, len(members))
	for _, item := range members {
		entry := item.(map[string]interface{})
		require.NotContains(t, entry, "email")
		require.NotContains(t, entry, "phone")
		require.NotContains(t, entry, "status")
		userID := uint(entry["user_id"].(float64))
		membershipIDs[userID] = uint(entry["id"].(float64))
	}
	require.NotEmpty(t, membershipIDs[owner.ID])
	require.NotEmpty(t, membershipIDs[admin.ID])
	require.NotEmpty(t, membershipIDs[member.ID])

	tc.Get("/v1/organizations/"+organizationID+"/members").
		WithToken(outsider.Token).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.NOT_FOUND")

	tc.Patch(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[member.ID])).
		WithToken(admin.Token).
		WithJSON(map[string]any{"role": "admin"}).
		Call().
		AssertStatus(403).
		AssertJSONPath("error_code", "PERMISSION.DENIED")

	tc.Patch(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[member.ID])).
		WithToken(owner.Token).
		WithJSON(map[string]any{"role": "admin"}).
		Call().
		AssertOk().
		AssertJSONPath("data.role", "admin")
	tc.Patch(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[member.ID])).
		WithToken(owner.Token).
		WithJSON(map[string]any{"role": "member"}).
		Call().
		AssertOk().
		AssertJSONPath("data.role", "member")
	roleAudit := tc.Get("/v1/audit-logs?action=change_role&resource=organization_members").
		WithToken(owner.Token).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(2)).
		JSON()
	assertOrganizationAuditContainsNoProfileFields(t, roleAudit)

	tc.Patch("/v1/organizations/"+organizationID+"/members/999999").
		WithToken(owner.Token).
		WithJSON(map[string]any{"role": "member"}).
		Call().
		AssertNotFound().
		AssertJSONPath("error_code", "ORGANIZATION.MEMBER_NOT_FOUND")

	tc.Delete("/v1/users/account").
		WithToken(member.Token).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED")

	tc.Delete(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[owner.ID])).
		WithToken(admin.Token).
		Call().
		AssertStatus(403).
		AssertJSONPath("error_code", "PERMISSION.DENIED")
	tc.Delete(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[member.ID])).
		WithToken(admin.Token).
		Call().
		AssertNoContent()
	removeAudit := tc.Get("/v1/audit-logs?action=remove&resource=organization_members").
		WithToken(admin.Token).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(1)).
		JSON()
	assertOrganizationAuditContainsNoProfileFields(t, removeAudit)
	tc.Delete("/v1/users/account").
		WithToken(member.Token).
		Call().
		AssertNoContent()

	transferJSON := tc.Post("/v1/organizations/"+organizationID+"/ownership-transfer").
		WithToken(owner.Token).
		WithJSON(map[string]any{"new_owner_member_id": membershipIDs[admin.ID]}).
		Call().
		AssertOk().
		AssertJSONPath("data.previous_owner.role", "admin").
		AssertJSONPath("data.new_owner.role", "owner").
		JSON()
	transferData := transferJSON["data"].(map[string]interface{})
	require.NotContains(t, transferData["previous_owner"].(map[string]interface{}), "email")
	require.NotContains(t, transferData["new_owner"].(map[string]interface{}), "email")
	transferAudit := tc.Get("/v1/audit-logs?action=transfer_ownership&resource=organizations").
		WithToken(owner.Token).
		Call().
		AssertOk().
		AssertJSONPath("meta.total", float64(1)).
		JSON()
	assertOrganizationAuditContainsNoProfileFields(t, transferAudit)

	tc.Post("/v1/organizations/"+organizationID+"/ownership-transfer").
		WithToken(admin.Token).
		WithJSON(map[string]any{"new_owner_member_id": membershipIDs[admin.ID]}).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.OWNERSHIP_TRANSFER_TARGET_INVALID")

	tc.Delete("/v1/users/account").
		WithToken(owner.Token).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED")
	tc.Delete(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[owner.ID])).
		WithToken(owner.Token).
		Call().
		AssertNoContent()
	tc.Delete("/v1/users/account").
		WithToken(owner.Token).
		Call().
		AssertNoContent()

	tc.Delete(fmt.Sprintf("/v1/organizations/%s/members/%d", organizationID, membershipIDs[admin.ID])).
		WithToken(admin.Token).
		Call().
		AssertStatus(409).
		AssertJSONPath("error_code", "ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED")
}

type organizationTestUser struct {
	ID    uint
	Token string
	Email string
}

func registerOrganizationTestUser(
	t *testing.T,
	tc *testplatform.TestCase,
	username string,
	email string,
) organizationTestUser {
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
	userData, ok := data["user"].(map[string]interface{})
	require.True(t, ok)
	return organizationTestUser{
		ID:    uint(userData["id"].(float64)),
		Token: token,
		Email: email,
	}
}

func inviteAndAcceptOrganizationMember(
	t *testing.T,
	tc *testplatform.TestCase,
	ownerToken string,
	invitee organizationTestUser,
	organizationID string,
	role string,
	invitationTokens <-chan string,
) {
	t.Helper()
	tc.Post("/v1/organizations/" + organizationID + "/invitations").
		WithToken(ownerToken).
		WithJSON(map[string]any{"email": invitee.Email, "role": role}).
		Call().
		AssertCreated()
	token := <-invitationTokens
	tc.Post("/v1/organization-invitations/accept").
		WithToken(invitee.Token).
		WithJSON(map[string]any{"token": token}).
		Call().
		AssertOk()
}

func assertOrganizationAuditContainsNoProfileFields(t *testing.T, payload map[string]interface{}) {
	t.Helper()
	items, ok := payload["data"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)
	for _, item := range items {
		entry := item.(map[string]interface{})
		metadata, ok := entry["metadata"].(map[string]interface{})
		require.True(t, ok)
		require.NotContains(t, metadata, "email")
		require.NotContains(t, metadata, "username")
		require.NotContains(t, metadata, "nickname")
		require.NotContains(t, metadata, "avatar")
	}
}

func registerAndLoginOrganizationUser(t *testing.T, tc *testplatform.TestCase, username, email string) string {
	return registerOrganizationTestUser(t, tc, username, email).Token
}

package redact

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSensitiveKey(t *testing.T) {
	tests := map[string]bool{
		"Authorization":         true,
		"X-API-Key":             true,
		"current_password":      true,
		"password_confirmation": true,
		"client_secret":         true,
		"invitation_token":      true,
		"webhook_signature":     true,
		"session_id":            true,
		"mfa_code":              true,
		"password_hash":         true,
		"api_key_plaintext":     true,
		"R2_SECRET_ACCESS_KEY":  true,
		"request_id":            false,
		"error_code":            false,
		"token_budget":          false,
		"status":                false,
	}

	for key, want := range tests {
		t.Run(key, func(t *testing.T) {
			assert.Equal(t, want, IsSensitiveKey(key))
		})
	}
}

func TestMapRedactsRecursivelyWithoutMutatingInput(t *testing.T) {
	type namedContext map[string]any
	input := map[string]any{
		"password": "plain-password",
		"nested": map[string]any{
			"access_token": "plain-token",
			"outcome":      "success",
		},
		"named": namedContext{"refresh_token": "named-token", "count": 2},
	}

	result := Map(input)

	assert.Equal(t, Placeholder, result["password"])
	nested, ok := result["nested"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, Placeholder, nested["access_token"])
	assert.Equal(t, "success", nested["outcome"])
	named, ok := result["named"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, Placeholder, named["refresh_token"])
	assert.Equal(t, 2, named["count"])
	assert.Equal(t, "plain-password", input["password"])
}

func TestRequestDataRedaction(t *testing.T) {
	headers := http.Header{
		"Authorization": []string{"Bearer header-secret"},
		"Cookie":        []string{"session=cookie-secret"},
		"X-Request-ID":  []string{"req-1"},
	}
	query := url.Values{
		"access_token": []string{"query-secret"},
		"code":         []string{"oauth-code"},
		"page":         []string{"2"},
	}

	assert.Equal(t, Placeholder, Headers(headers)["Authorization"])
	assert.Equal(t, Placeholder, Headers(headers)["Cookie"])
	assert.Equal(t, "req-1", Headers(headers)["X-Request-ID"])
	assert.Equal(t, Placeholder, Query(query)["access_token"])
	assert.Equal(t, Placeholder, Query(query)["code"])
	assert.Equal(t, "2", Query(query)["page"])

	parsed, err := url.Parse("/callback?access_token=query-secret&page=2")
	require.NoError(t, err)
	redactedURL := URL(parsed)
	assert.NotContains(t, redactedURL, "query-secret")
	assert.Contains(t, redactedURL, "page=2")
	assert.Contains(t, redactedURL, "%5BREDACTED%5D")
	assert.Contains(t, URLWithPath(parsed, "/callback/:provider"), "/callback/:provider")
}

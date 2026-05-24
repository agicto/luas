package feature

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestUserRegistration(t *testing.T) {
	// Generate unique email to avoid constraint errors if DB persists
	rand.Seed(time.Now().UnixNano())
	email := fmt.Sprintf("test_%d@example.com", rand.Intn(100000))

	tc := NewTestCase(t)

	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "testuser",
			"email":    email,
			"password": "password123",
		}).
		Call().
		AssertCreated().
		AssertJSONPath("data.username", "testuser").
		AssertJSONPath("data.email", email)
}

func TestUserLogin(t *testing.T) {
	// 1. Register
	rand.Seed(time.Now().UnixNano())
	email := fmt.Sprintf("login_%d@example.com", rand.Intn(100000))
	password := "password123"

	tc := NewTestCase(t)
	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "loginuser",
			"email":    email,
			"password": password,
		}).
		Call().
		AssertCreated()

	// 2. Login
	tc.Post("/v1/login").
		WithJSON(map[string]any{
			"username": email,
			"password": password,
		}).
		Call().
		AssertOk().
		AssertJSONStructure([]string{"data.access_token", "data.user"})
}

func TestPasswordResetRequestReturnsGenericSuccessMessage(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	email := fmt.Sprintf("reset_%d@example.com", rand.Intn(100000))

	tc := NewTestCase(t)
	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "resetuser",
			"email":    email,
			"password": "password123",
		}).
		Call().
		AssertCreated()

	tc.Post("/v1/password/reset").
		WithJSON(map[string]any{
			"email": email,
		}).
		Call().
		AssertOk().
		AssertJSONPath("data.message", "If the account exists, password reset instructions have been sent")
}

func TestUserDirectoryRoutesAreNotExposedByDefaultStarter(t *testing.T) {
	NewTestCase(t).
		Get("/v1/users").
		Call().
		AssertNotFound()
}

func TestRegistrationConflictReturnsStableErrorCodeAndRequestID(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	email := fmt.Sprintf("dup_%d@example.com", rand.Intn(100000))

	tc := NewTestCase(t)
	tc.Post("/v1/register").
		WithJSON(map[string]any{
			"username": "duplicate-user",
			"email":    email,
			"password": "password123",
		}).
		Call().
		AssertCreated()

	tc.Post("/v1/register").
		WithHeader("X-Request-ID", "req-feature-1").
		WithJSON(map[string]any{
			"username": "duplicate-user",
			"email":    fmt.Sprintf("other_%d@example.com", rand.Intn(100000)),
			"password": "password123",
		}).
		Call().
		AssertStatus(409).
		AssertHeader("X-Request-ID", "req-feature-1").
		AssertJSONPath("error_code", "USER.USERNAME_ALREADY_EXISTS").
		AssertJSONPath("request_id", "req-feature-1")
}

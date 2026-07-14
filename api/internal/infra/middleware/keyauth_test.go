package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/pkg/response"
)

func TestKeyAuthWithConfig_UsesValidationErrorHandler(t *testing.T) {
	validationErr := errors.New("dependency unavailable")
	var (
		handlerCalled bool
		routeCalled   bool
	)

	router := gin.New()
	router.Use(KeyAuthWithConfig(KeyAuthConfig{
		ValidatorWithContext: func(*gin.Context, string) (*KeyAuthResult, error) {
			return nil, validationErr
		},
		ValidationErrorHandler: func(c *gin.Context, err error) {
			handlerCalled = true
			if !errors.Is(err, validationErr) {
				t.Fatalf("validation error = %v, want %v", err, validationErr)
			}
			response.Abort(c, http.StatusServiceUnavailable, "API key validation unavailable")
		},
	}))
	router.GET("/protected", func(c *gin.Context) {
		routeCalled = true
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("X-API-Key", "luas_test.secret")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !handlerCalled {
		t.Fatal("validation error handler was not called")
	}
	if routeCalled {
		t.Fatal("protected route ran after validation failed")
	}
}

func TestKeyAuthWithConfig_DefaultsValidatorErrorsToUnauthorized(t *testing.T) {
	router := gin.New()
	router.Use(KeyAuthWithConfig(KeyAuthConfig{
		ValidatorWithContext: func(*gin.Context, string) (*KeyAuthResult, error) {
			return nil, errors.New("invalid key")
		},
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("X-API-Key", "invalid")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

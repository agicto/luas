package bootstrap

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestRegisterDomainErrorMappings(t *testing.T) {
	mapper := &response.ErrorMapper{}
	registerDomainErrorMappings(mapper)

	tests := []struct {
		name       string
		err        error
		statusCode int
		errorCode  string
	}{
		{
			name:       "wrapped user not found",
			err:        fmt.Errorf("load profile: %w", domain.ErrUserNotFound),
			statusCode: http.StatusNotFound,
			errorCode:  domain.CodeUserNotFound,
		},
		{
			name:       "invalid credentials",
			err:        domain.ErrInvalidCredentials,
			statusCode: http.StatusUnauthorized,
			errorCode:  domain.CodeInvalidCredentials,
		},
		{
			name:       "permission denied",
			err:        domain.ErrPermissionDenied,
			statusCode: http.StatusForbidden,
			errorCode:  domain.CodePermissionDenied,
		},
		{
			name:       "username already exists",
			err:        domain.ErrUsernameAlreadyExists,
			statusCode: http.StatusConflict,
			errorCode:  domain.CodeUsernameAlreadyExists,
		},
		{
			name:       "invalid input",
			err:        domain.ErrInvalidInput,
			statusCode: http.StatusUnprocessableEntity,
			errorCode:  domain.CodeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			descriptor := mapper.Resolve(tt.err)
			if descriptor.StatusCode != tt.statusCode {
				t.Fatalf("status = %d, want %d", descriptor.StatusCode, tt.statusCode)
			}
			if descriptor.ErrorCode != tt.errorCode {
				t.Fatalf("error_code = %q, want %q", descriptor.ErrorCode, tt.errorCode)
			}
		})
	}
}

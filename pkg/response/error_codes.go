package response

import (
	"errors"
	"net/http"

	"github.com/zgiai/zgo/internal/domain"
	"gorm.io/gorm"
)

const (
	ErrorCodeInternal         = "COMMON.INTERNAL"
	ErrorCodeValidationFailed = "COMMON.VALIDATION_FAILED"
	ErrorCodeRateLimited      = "COMMON.RATE_LIMITED"
	ErrorCodeServiceDown      = "COMMON.SERVICE_UNAVAILABLE"
	ErrorCodeUnauthorized     = "AUTH.UNAUTHORIZED"
	ErrorCodeForbidden        = "AUTH.FORBIDDEN"
	ErrorCodeNotFound         = domain.CodeNotFound
	ErrorCodeConflict         = domain.CodeConflict
	ErrorCodeInvalidInput     = domain.CodeInvalidInput
)

// ErrorDescriptor combines transport status with a stable machine-readable code.
type ErrorDescriptor struct {
	StatusCode int
	ErrorCode  string
}

// ErrorMapper maps errors to response descriptors.
type ErrorMapper struct {
	mappings map[error]ErrorDescriptor
}

// Register adds a custom error mapping.
func (m *ErrorMapper) Register(err error, statusCode int, errorCode string) {
	if m == nil {
		return
	}
	if m.mappings == nil {
		m.mappings = make(map[error]ErrorDescriptor)
	}
	m.mappings[err] = ErrorDescriptor{
		StatusCode: statusCode,
		ErrorCode:  errorCode,
	}
}

// Resolve returns the response descriptor for an error.
func (m *ErrorMapper) Resolve(err error) ErrorDescriptor {
	if m == nil {
		return ErrorDescriptor{
			StatusCode: http.StatusInternalServerError,
			ErrorCode:  ErrorCodeInternal,
		}
	}

	if err != nil {
		for mappedErr, descriptor := range m.mappings {
			if errors.Is(err, mappedErr) {
				return descriptor
			}
		}
	}

	return ErrorDescriptor{
		StatusCode: http.StatusInternalServerError,
		ErrorCode:  ErrorCodeInternal,
	}
}

// GetStatusCode returns the HTTP status code for an error.
func (m *ErrorMapper) GetStatusCode(err error) int {
	return m.Resolve(err).StatusCode
}

// GetErrorCode returns the stable machine-readable error code for an error.
func (m *ErrorMapper) GetErrorCode(err error) string {
	return m.Resolve(err).ErrorCode
}

// DefaultErrorMapper provides default error mappings for framework and domain errors.
var DefaultErrorMapper = &ErrorMapper{
	mappings: map[error]ErrorDescriptor{
		ErrNotFound:                  {StatusCode: http.StatusNotFound, ErrorCode: ErrorCodeNotFound},
		ErrUnauthorized:              {StatusCode: http.StatusUnauthorized, ErrorCode: ErrorCodeUnauthorized},
		ErrForbidden:                 {StatusCode: http.StatusForbidden, ErrorCode: ErrorCodeForbidden},
		ErrConflict:                  {StatusCode: http.StatusConflict, ErrorCode: ErrorCodeConflict},
		ErrValidation:                {StatusCode: http.StatusUnprocessableEntity, ErrorCode: ErrorCodeValidationFailed},
		gorm.ErrRecordNotFound:       {StatusCode: http.StatusNotFound, ErrorCode: ErrorCodeNotFound},
		domain.ErrNotFound:           {StatusCode: http.StatusNotFound, ErrorCode: domain.CodeNotFound},
		domain.ErrUserNotFound:       {StatusCode: http.StatusNotFound, ErrorCode: domain.CodeUserNotFound},
		domain.ErrRoleNotFound:       {StatusCode: http.StatusNotFound, ErrorCode: domain.CodeRoleNotFound},
		domain.ErrAPIKeyNotFound:     {StatusCode: http.StatusNotFound, ErrorCode: domain.CodeAPIKeyNotFound},
		domain.ErrInvalidCredentials: {StatusCode: http.StatusUnauthorized, ErrorCode: domain.CodeInvalidCredentials},
		domain.ErrAPIKeyInvalid:      {StatusCode: http.StatusUnauthorized, ErrorCode: domain.CodeAPIKeyInvalid},
		domain.ErrAPIKeyExpired:      {StatusCode: http.StatusUnauthorized, ErrorCode: domain.CodeAPIKeyExpired},
		domain.ErrAPIKeyRevoked:      {StatusCode: http.StatusUnauthorized, ErrorCode: domain.CodeAPIKeyRevoked},
		domain.ErrAccountDisabled:    {StatusCode: http.StatusForbidden, ErrorCode: domain.CodeAccountDisabled},
		domain.ErrPermissionDenied:   {StatusCode: http.StatusForbidden, ErrorCode: domain.CodePermissionDenied},
		domain.ErrEmailAlreadyExists: {StatusCode: http.StatusConflict, ErrorCode: domain.CodeEmailAlreadyExists},
		domain.ErrUsernameAlreadyExists: {
			StatusCode: http.StatusConflict,
			ErrorCode:  domain.CodeUsernameAlreadyExists,
		},
		domain.ErrPasswordResetTokenInvalid: {
			StatusCode: http.StatusUnauthorized,
			ErrorCode:  domain.CodePasswordResetTokenInvalid,
		},
		domain.ErrPasswordResetTokenExpired: {
			StatusCode: http.StatusUnauthorized,
			ErrorCode:  domain.CodePasswordResetTokenExpired,
		},
		domain.ErrConflict:     {StatusCode: http.StatusConflict, ErrorCode: domain.CodeConflict},
		domain.ErrInvalidInput: {StatusCode: http.StatusUnprocessableEntity, ErrorCode: domain.CodeInvalidInput},
	},
}

func defaultErrorCodeForStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrorCodeInvalidInput
	case http.StatusUnauthorized:
		return ErrorCodeUnauthorized
	case http.StatusForbidden:
		return ErrorCodeForbidden
	case http.StatusNotFound:
		return ErrorCodeNotFound
	case http.StatusConflict:
		return ErrorCodeConflict
	case http.StatusUnprocessableEntity:
		return ErrorCodeValidationFailed
	case http.StatusTooManyRequests:
		return ErrorCodeRateLimited
	case http.StatusServiceUnavailable:
		return ErrorCodeServiceDown
	default:
		return ErrorCodeInternal
	}
}

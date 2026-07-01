package response

import (
	"errors"
	"net/http"

	"gorm.io/gorm"
)

const (
	ErrorCodeInternal           = "COMMON.INTERNAL"
	ErrorCodeValidationFailed   = "COMMON.VALIDATION_FAILED"
	ErrorCodeRateLimited        = "COMMON.RATE_LIMITED"
	ErrorCodeRequestTooLarge    = "COMMON.REQUEST_TOO_LARGE"
	ErrorCodeTimeout            = "COMMON.TIMEOUT"
	ErrorCodeServiceUnavailable = "COMMON.SERVICE_UNAVAILABLE"
	ErrorCodeUnauthorized       = "AUTH.UNAUTHORIZED"
	ErrorCodeForbidden          = "AUTH.FORBIDDEN"
	ErrorCodeNotFound           = "COMMON.NOT_FOUND"
	ErrorCodeConflict           = "COMMON.CONFLICT"
	ErrorCodeInvalidInput       = "COMMON.INVALID_INPUT"
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

// DefaultErrorMapper provides default mappings for transport-level response errors.
var DefaultErrorMapper = &ErrorMapper{
	mappings: map[error]ErrorDescriptor{
		ErrNotFound:            {StatusCode: http.StatusNotFound, ErrorCode: ErrorCodeNotFound},
		ErrUnauthorized:        {StatusCode: http.StatusUnauthorized, ErrorCode: ErrorCodeUnauthorized},
		ErrForbidden:           {StatusCode: http.StatusForbidden, ErrorCode: ErrorCodeForbidden},
		ErrConflict:            {StatusCode: http.StatusConflict, ErrorCode: ErrorCodeConflict},
		ErrValidation:          {StatusCode: http.StatusUnprocessableEntity, ErrorCode: ErrorCodeValidationFailed},
		gorm.ErrRecordNotFound: {StatusCode: http.StatusNotFound, ErrorCode: ErrorCodeNotFound},
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
	case http.StatusRequestEntityTooLarge:
		return ErrorCodeRequestTooLarge
	case http.StatusServiceUnavailable:
		return ErrorCodeServiceUnavailable
	default:
		return ErrorCodeInternal
	}
}

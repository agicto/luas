package handler

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/zgiai/luas/api/pkg/response"
)

var registerFieldNames sync.Once

// useWireFieldNames makes validation errors name fields as clients send them (the json, then form,
// tag) instead of Go struct field names.
func useWireFieldNames() {
	registerFieldNames.Do(func() {
		engine, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			return
		}
		engine.RegisterTagNameFunc(func(field reflect.StructField) string {
			for _, tag := range []string{"json", "form", "uri"} {
				name := strings.Split(field.Tag.Get(tag), ",")[0]
				if name == "-" {
					return ""
				}
				if name != "" {
					return name
				}
			}
			return field.Name
		})
	})
}

// WriteBodyError answers a request body that could not be bound, following the global contract:
// field validation failures are 422 COMMON.VALIDATION_FAILED with per-field messages, an oversized
// body is 413 COMMON.REQUEST_TOO_LARGE, and malformed JSON is 400 COMMON.INVALID_INPUT. Handlers
// that decode bodies themselves use it for decoder errors.
func WriteBodyError(c *gin.Context, err error) {
	var fields validator.ValidationErrors
	if errors.As(err, &fields) {
		response.ValidationFailed(c, validationMessages(fields))
		return
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		response.ErrorWithCode(c, http.StatusRequestEntityTooLarge, response.ErrorCodeRequestTooLarge, "Request body too large")
		return
	}
	response.BadRequest(c, "Invalid request body", err)
}

func validationMessages(fields validator.ValidationErrors) map[string][]string {
	messages := make(map[string][]string, len(fields))
	for _, field := range fields {
		name := fieldPath(field)
		messages[name] = append(messages[name], validationMessage(field))
	}
	return messages
}

// fieldPath drops the root struct name: "UserRegisterRequest.username" becomes "username".
func fieldPath(field validator.FieldError) string {
	namespace := field.Namespace()
	if index := strings.IndexByte(namespace, '.'); index >= 0 {
		return namespace[index+1:]
	}
	return field.Field()
}

func validationMessage(field validator.FieldError) string {
	param := field.Param()
	isText := field.Kind() == reflect.String
	isList := field.Kind() == reflect.Slice || field.Kind() == reflect.Array || field.Kind() == reflect.Map
	switch field.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "url", "http_url":
		return "must be a valid URL"
	case "oneof":
		return "must be one of: " + strings.Join(strings.Fields(param), ", ")
	case "excludesrune", "excludes":
		return fmt.Sprintf("must not contain %q", param)
	case "min", "gte":
		switch {
		case isText:
			return "must be at least " + param + " characters"
		case isList:
			return "must contain at least " + param + " items"
		default:
			return "must be at least " + param
		}
	case "max", "lte":
		switch {
		case isText:
			return "must be at most " + param + " characters"
		case isList:
			return "must contain at most " + param + " items"
		default:
			return "must be at most " + param
		}
	case "len":
		return "must have length " + param
	default:
		return "is invalid"
	}
}

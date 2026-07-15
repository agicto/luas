// Package redact removes credential-shaped values before they reach telemetry
// or developer diagnostics.
package redact

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"unicode"
)

const (
	// Placeholder replaces a value whose key identifies credential material.
	Placeholder = "[REDACTED]"
	maxDepth    = 8
)

var sensitiveExact = map[string]struct{}{
	"passwordconfirmation": {},
	"mfacode":              {},
	"pin":                  {},
	"recoverycode":         {},
	"verificationcode":     {},
}

var sensitiveQueryExact = map[string]struct{}{
	"code": {},
	"key":  {},
	"sig":  {},
}

// IsSensitiveKey reports whether a field name conventionally carries a secret.
func IsSensitiveKey(key string) bool {
	key = canonicalKey(key)
	if key == "" {
		return false
	}
	if _, ok := sensitiveExact[key]; ok {
		return true
	}
	if hasSensitiveSuffix(key) {
		return true
	}
	if base := trimSensitiveQualifier(key); base != "" {
		return hasSensitiveSuffix(base)
	}
	return false
}

func hasSensitiveSuffix(key string) bool {
	if key == "" {
		return false
	}
	switch key[len(key)-1] {
	case 'd':
		return strings.HasSuffix(key, "password") ||
			strings.HasSuffix(key, "passwd") ||
			strings.HasSuffix(key, "sessionid") ||
			strings.HasSuffix(key, "accesskeyid")
	case 'e':
		return strings.HasSuffix(key, "cookie") || strings.HasSuffix(key, "signature")
	case 'f':
		return strings.HasSuffix(key, "csrf")
	case 'h':
		return strings.HasSuffix(key, "auth")
	case 'l':
		return strings.HasSuffix(key, "credential")
	case 'n':
		return strings.HasSuffix(key, "token") || strings.HasSuffix(key, "authorization")
	case 'p':
		return strings.HasSuffix(key, "otp")
	case 's':
		return strings.HasSuffix(key, "credentials")
	case 't':
		return strings.HasSuffix(key, "secret")
	case 'y':
		return strings.HasSuffix(key, "apikey") ||
			strings.HasSuffix(key, "privatekey") ||
			strings.HasSuffix(key, "signingkey") ||
			strings.HasSuffix(key, "secretaccesskey") ||
			strings.HasSuffix(key, "encryptionkey")
	default:
		return false
	}
}

func trimSensitiveQualifier(key string) string {
	if key == "" {
		return ""
	}
	switch key[len(key)-1] {
	case 'e':
		return strings.TrimSuffix(key, "value")
	case 'h':
		return strings.TrimSuffix(key, "hash")
	case 'r':
		return strings.TrimSuffix(key, "header")
	case 't':
		return strings.TrimSuffix(key, "plaintext")
	default:
		return ""
	}
}

// Map returns a recursively copied map with sensitive fields replaced.
func Map(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	return mapValue(values, 0)
}

// Value recursively copies supported maps and slices while redacting nested fields.
func Value(value any) any {
	return redactValue(value, 0)
}

// Headers returns the first value for each header after credential headers are redacted.
func Headers(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if IsSensitiveKey(key) {
			result[key] = Placeholder
			continue
		}
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// Query returns the first query value per key after credential parameters are redacted.
func Query(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, entries := range values {
		if isSensitiveQueryKey(key) {
			result[key] = Placeholder
			continue
		}
		if len(entries) > 0 {
			result[key] = entries[0]
		}
	}
	return result
}

// URL returns a copy of a URL string with credentials and sensitive query values redacted.
func URL(value *url.URL) string {
	if value == nil {
		return ""
	}
	cloned := *value
	query := cloned.Query()
	for key, values := range query {
		if !isSensitiveQueryKey(key) {
			continue
		}
		for index := range values {
			values[index] = Placeholder
		}
		query[key] = values
	}
	cloned.RawQuery = query.Encode()
	if cloned.User != nil {
		cloned.User = url.User(Placeholder)
	}
	return cloned.String()
}

// URLWithPath redacts a URL after replacing its concrete path with a route template.
func URLWithPath(value *url.URL, path string) string {
	if value == nil {
		return ""
	}
	cloned := *value
	if path = strings.TrimSpace(path); path != "" {
		cloned.Path = path
		cloned.RawPath = ""
	}
	return URL(&cloned)
}

func mapValue(values map[string]any, depth int) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		if IsSensitiveKey(key) {
			result[key] = Placeholder
			continue
		}
		result[key] = redactValue(value, depth+1)
	}
	return result
}

func redactValue(value any, depth int) any {
	if depth >= maxDepth {
		return Placeholder
	}

	switch typed := value.(type) {
	case http.Header:
		return Headers(typed)
	case url.Values:
		return Query(typed)
	case map[string]any:
		return mapValue(typed, depth)
	case map[string]string:
		result := make(map[string]string, len(typed))
		for key, item := range typed {
			if IsSensitiveKey(key) {
				result[key] = Placeholder
			} else {
				result[key] = item
			}
		}
		return result
	case map[string][]string:
		result := make(map[string][]string, len(typed))
		for key, items := range typed {
			if IsSensitiveKey(key) {
				result[key] = []string{Placeholder}
			} else {
				result[key] = append([]string(nil), items...)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = redactValue(item, depth+1)
		}
		return result
	case []string:
		return append([]string(nil), typed...)
	default:
		return redactReflected(value, depth)
	}
}

func redactReflected(value any, depth int) any {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return nil
	}

	switch reflected.Kind() {
	case reflect.Interface, reflect.Pointer:
		if reflected.IsNil() {
			return value
		}
		return redactValue(reflected.Elem().Interface(), depth+1)
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return value
		}
		result := make(map[string]any, reflected.Len())
		iterator := reflected.MapRange()
		for iterator.Next() {
			key := iterator.Key().String()
			if IsSensitiveKey(key) {
				result[key] = Placeholder
				continue
			}
			result[key] = redactValue(iterator.Value().Interface(), depth+1)
		}
		return result
	case reflect.Slice, reflect.Array:
		result := make([]any, reflected.Len())
		for index := 0; index < reflected.Len(); index++ {
			result[index] = redactValue(reflected.Index(index).Interface(), depth+1)
		}
		return result
	default:
		return value
	}
}

func isSensitiveQueryKey(key string) bool {
	if IsSensitiveKey(key) {
		return true
	}
	_, ok := sensitiveQueryExact[canonicalKey(key)]
	return ok
}

func canonicalKey(key string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(strings.TrimSpace(key)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

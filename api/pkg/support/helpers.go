package support

import (
	"reflect"
	"strings"
)

// Blank determines if the given value is "blank"
// Returns true for: nil, empty string, empty slice/map, zero values
func Blank(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Bool:
		return false // booleans are never blank
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return true
		}
		return Blank(v.Elem().Interface())
	default:
		return v.IsZero()
	}
}

// Filled determines if a value is "filled" (not blank)
func Filled(value any) bool {
	return !Blank(value)
}

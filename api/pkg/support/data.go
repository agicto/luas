package support

import (
	"reflect"
	"strconv"
	"strings"
)

// DataGet retrieves a value from a nested data structure using "dot" notation
// Example: DataGet(data, "user.profile.name")
func DataGet(target any, key string, defaultVal ...any) any {
	if target == nil {
		return getDefault(defaultVal)
	}

	if key == "" {
		return target
	}

	keys := strings.Split(key, ".")
	current := target

	for _, k := range keys {
		current = getSegment(current, k)
		if current == nil {
			return getDefault(defaultVal)
		}
	}

	return current
}

// DataHas checks if a key exists in a nested data structure using "dot" notation
func DataHas(target any, key string) bool {
	if target == nil || key == "" {
		return false
	}

	keys := strings.Split(key, ".")
	current := target

	for _, k := range keys {
		current = getSegment(current, k)
		if current == nil {
			return false
		}
	}

	return true
}

// getSegment retrieves a single segment from a value
func getSegment(target any, key string) any {
	if target == nil {
		return nil
	}

	v := reflect.ValueOf(target)

	// Handle pointers
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Map:
		mapVal := v.MapIndex(reflect.ValueOf(key))
		if !mapVal.IsValid() {
			return nil
		}
		return mapVal.Interface()

	case reflect.Struct:
		field := v.FieldByName(key)
		if !field.IsValid() {
			// Try case-insensitive match
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				if strings.EqualFold(t.Field(i).Name, key) {
					field = v.Field(i)
					break
				}
			}
		}
		if field.IsValid() && field.CanInterface() {
			return field.Interface()
		}
		return nil

	case reflect.Slice, reflect.Array:
		index, err := strconv.Atoi(key)
		if err != nil {
			return nil
		}
		if index < 0 || index >= v.Len() {
			return nil
		}
		return v.Index(index).Interface()

	default:
		return nil
	}
}

// getDefault returns the default value from a variadic slice
func getDefault(defaultVal []any) any {
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return nil
}

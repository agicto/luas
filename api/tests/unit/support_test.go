package unit

import (
	"testing"

	"github.com/zgiai/luas/api/pkg/support"
)

func TestBlank(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"whitespace string", "   ", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"positive int", 42, false},
		{"false bool", false, false},
		{"true bool", true, false},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1, 2, 3}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := support.Blank(tt.value)
			if result != tt.expected {
				t.Errorf("Blank(%v) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestFilled(t *testing.T) {
	if support.Filled(nil) {
		t.Error("Filled(nil) should be false")
	}
	if !support.Filled("hello") {
		t.Error("Filled(\"hello\") should be true")
	}
	if support.Filled("") {
		t.Error("Filled(\"\") should be false")
	}
}

func TestDataGet(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "John",
			"age":  30,
		},
		"items": []any{"a", "b", "c"},
	}

	// Test nested map access
	name := support.DataGet(data, "user.name")
	if name != "John" {
		t.Errorf("Expected 'John', got '%v'", name)
	}

	// Test with default
	missing := support.DataGet(data, "user.email", "default@example.com")
	if missing != "default@example.com" {
		t.Errorf("Expected default value, got '%v'", missing)
	}

	// Test array access
	item := support.DataGet(data, "items.1")
	if item != "b" {
		t.Errorf("Expected 'b', got '%v'", item)
	}
}

func TestDataHas(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "John",
		},
	}

	if !support.DataHas(data, "user.name") {
		t.Error("DataHas should return true for existing key")
	}

	if support.DataHas(data, "user.email") {
		t.Error("DataHas should return false for non-existing key")
	}
}

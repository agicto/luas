package support

import "testing"

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
		{"non-zero int", 1, false},
		{"zero float", 0.0, true},
		{"non-zero float", 1.5, false},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1, 2}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
		{"false bool", false, false}, // booleans are never blank
		{"true bool", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Blank(tt.value); got != tt.expected {
				t.Errorf("Blank(%v) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestFilled(t *testing.T) {
	if !Filled("hello") {
		t.Error("Filled('hello') should be true")
	}
	if Filled("") {
		t.Error("Filled('') should be false")
	}
}

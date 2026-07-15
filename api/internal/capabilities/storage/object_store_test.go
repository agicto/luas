package storage

import "testing"

func TestValidateObjectKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "nested generated key", key: "asset-uploads/019bf6d8-17c5-7a98-a084-6d45793f5f0c/object", valid: true},
		{name: "final key", key: "assets/019bf6d8-17c5-7a98-a084-6d45793f5f0c/object.pdf", valid: true},
		{name: "empty", key: "", valid: false},
		{name: "absolute", key: "/etc/passwd", valid: false},
		{name: "parent traversal", key: "../escape", valid: false},
		{name: "nested parent traversal", key: "safe/../../escape", valid: false},
		{name: "current segment", key: "safe/./object", valid: false},
		{name: "backslash", key: `safe\object`, valid: false},
		{name: "empty segment", key: "safe//object", valid: false},
		{name: "trailing slash", key: "safe/", valid: false},
		{name: "uppercase", key: "Assets/object", valid: false},
		{name: "query-like", key: "assets/object?version=1", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateObjectKey(test.key)
			if test.valid && err != nil {
				t.Fatalf("ValidateObjectKey() error = %v, want nil", err)
			}
			if !test.valid && err == nil {
				t.Fatal("ValidateObjectKey() error = nil, want rejection")
			}
		})
	}
}

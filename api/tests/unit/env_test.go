package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zgiai/luas/api/pkg/env"
)

func TestEnv_Get(t *testing.T) {
	// Set a test variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	value := env.Get("TEST_VAR")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}
}

func TestEnv_GetWithDefault(t *testing.T) {
	os.Unsetenv("NON_EXISTENT_VAR")

	value := env.Get("NON_EXISTENT_VAR", "default_value")
	if value != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", value)
	}
}

func TestEnv_GetBool(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
	}

	for _, tt := range tests {
		os.Setenv("TEST_BOOL", tt.value)
		result := env.GetBool("TEST_BOOL")
		if result != tt.expected {
			t.Errorf("GetBool(%q) = %v, expected %v", tt.value, result, tt.expected)
		}
	}
	os.Unsetenv("TEST_BOOL")
}

func TestEnv_GetBoolDefault(t *testing.T) {
	os.Unsetenv("NON_EXISTENT_BOOL")

	if env.GetBool("NON_EXISTENT_BOOL") != false {
		t.Error("Expected false for non-existent bool without default")
	}

	if env.GetBool("NON_EXISTENT_BOOL", true) != true {
		t.Error("Expected true for non-existent bool with default true")
	}
}

func TestEnv_GetInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	value := env.GetInt("TEST_INT")
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
}

func TestEnv_GetIntDefault(t *testing.T) {
	os.Unsetenv("NON_EXISTENT_INT")

	value := env.GetInt("NON_EXISTENT_INT", 100)
	if value != 100 {
		t.Errorf("Expected 100, got %d", value)
	}
}

func TestEnv_GetIntInvalid(t *testing.T) {
	os.Setenv("INVALID_INT", "not_a_number")
	defer os.Unsetenv("INVALID_INT")

	value := env.GetInt("INVALID_INT", 50)
	if value != 50 {
		t.Errorf("Expected default 50 for invalid int, got %d", value)
	}
}

func TestEnv_GetFloat(t *testing.T) {
	os.Setenv("TEST_FLOAT", "3.14")
	defer os.Unsetenv("TEST_FLOAT")

	value := env.GetFloat("TEST_FLOAT")
	if value != 3.14 {
		t.Errorf("Expected 3.14, got %f", value)
	}
}

func TestEnv_GetSlice(t *testing.T) {
	os.Setenv("TEST_SLICE", "a,b,c,d")
	defer os.Unsetenv("TEST_SLICE")

	value := env.GetSlice("TEST_SLICE")
	expected := []string{"a", "b", "c", "d"}

	if len(value) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(value))
	}

	for i, v := range expected {
		if value[i] != v {
			t.Errorf("Expected value[%d] = %s, got %s", i, v, value[i])
		}
	}
}

func TestEnv_GetSliceWithSpaces(t *testing.T) {
	os.Setenv("TEST_SLICE_SPACES", " a , b , c ")
	defer os.Unsetenv("TEST_SLICE_SPACES")

	value := env.GetSlice("TEST_SLICE_SPACES")
	expected := []string{"a", "b", "c"}

	if len(value) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(value))
	}

	for i, v := range expected {
		if value[i] != v {
			t.Errorf("Expected value[%d] = %s, got %s", i, v, value[i])
		}
	}
}

func TestEnv_GetSliceDefault(t *testing.T) {
	os.Unsetenv("NON_EXISTENT_SLICE")

	defaultSlice := []string{"x", "y", "z"}
	value := env.GetSlice("NON_EXISTENT_SLICE", defaultSlice)

	if len(value) != len(defaultSlice) {
		t.Errorf("Expected %d items, got %d", len(defaultSlice), len(value))
	}
}

func TestEnv_Set(t *testing.T) {
	env.Set("NEW_VAR", "new_value")
	defer os.Unsetenv("NEW_VAR")

	if os.Getenv("NEW_VAR") != "new_value" {
		t.Error("Set did not work")
	}
}

func TestEnv_Unset(t *testing.T) {
	os.Setenv("TO_UNSET", "value")
	env.Unset("TO_UNSET")

	if _, exists := os.LookupEnv("TO_UNSET"); exists {
		t.Error("Unset did not work")
	}
}

func TestEnv_AppEnv(t *testing.T) {
	os.Setenv("APP_ENV", "testing")
	defer os.Unsetenv("APP_ENV")

	mustLoadFresh(t)
	result := env.AppEnv()
	if result != "testing" {
		t.Errorf("Expected 'testing', got '%s'", result)
	}
}

func TestEnv_IsProduction(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"production", true},
		{"prod", true},
		{"release", true},
		{"development", false},
		{"dev", false},
		{"testing", false},
	}

	for _, tt := range tests {
		os.Setenv("APP_ENV", tt.env)
		mustLoadFresh(t)
		result := env.IsProduction()
		if result != tt.expected {
			t.Errorf("IsProduction() with APP_ENV=%s: expected %v, got %v", tt.env, tt.expected, result)
		}
	}
	os.Unsetenv("APP_ENV")
}

func TestEnv_IsDevelopment(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"development", true},
		{"dev", true},
		{"local", true},
		{"debug", true},
		{"production", false},
		{"testing", false},
	}

	for _, tt := range tests {
		os.Setenv("APP_ENV", tt.env)
		mustLoadFresh(t)
		result := env.IsDevelopment()
		if result != tt.expected {
			t.Errorf("IsDevelopment() with APP_ENV=%s: expected %v, got %v", tt.env, tt.expected, result)
		}
	}
	os.Unsetenv("APP_ENV")
}

func TestEnv_IsTesting(t *testing.T) {
	os.Setenv("APP_ENV", "testing")
	defer os.Unsetenv("APP_ENV")

	mustLoadFresh(t)
	if !env.IsTesting() {
		t.Error("Expected IsTesting() to return true")
	}
}

func TestEnv_SystemEnvPriority(t *testing.T) {
	// System env should have highest priority
	os.Setenv("PRIORITY_TEST", "from_system")
	defer os.Unsetenv("PRIORITY_TEST")

	mustLoadFresh(t)

	value := env.Get("PRIORITY_TEST")
	if value != "from_system" {
		t.Errorf("Expected 'from_system', got '%s'", value)
	}
}

func TestEnv_FilePrecedence(t *testing.T) {
	const key = "LUAS_TEST_FILE_PRECEDENCE"
	useIsolatedEnvDir(t, key)

	writeEnvFile(t, ".env", key+"=base\n")
	writeEnvFile(t, ".env.production", key+"=environment\n")
	writeEnvFile(t, ".env.local", key+"=local\n")
	writeEnvFile(t, ".env.production.local", key+"=environment_local\n")
	if err := os.Setenv("APP_ENV", "production"); err != nil {
		t.Fatalf("set APP_ENV: %v", err)
	}

	mustLoadFresh(t)

	if got := env.Get(key); got != "environment_local" {
		t.Fatalf("Get(%q) = %q, want %q", key, got, "environment_local")
	}
}

func TestEnv_EnvironmentAliasesSelectCanonicalEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		value    string
	}{
		{name: "GO_ENV", selector: "GO_ENV", value: "production"},
		{name: "GIN_MODE", selector: "GIN_MODE", value: "release"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const key = "LUAS_TEST_ENV_ALIAS_FILE"
			useIsolatedEnvDir(t, key)
			writeEnvFile(t, ".env."+test.value, key+"=selected\n")
			if err := os.Setenv(test.selector, " "+test.value+" "); err != nil {
				t.Fatalf("set %s: %v", test.selector, err)
			}

			mustLoadFresh(t)

			if got := env.AppEnv(); got != test.value {
				t.Fatalf("AppEnv() = %q, want %q", got, test.value)
			}
			if got := env.Get(key); got != "selected" {
				t.Fatalf("Get(%q) = %q, want selected", key, got)
			}
		})
	}
}

func TestEnv_LocalOverlayCannotChangeSelectedEnvironment(t *testing.T) {
	useIsolatedEnvDir(t)
	writeEnvFile(t, ".env", "APP_ENV=production\n")
	writeEnvFile(t, ".env.local", "APP_ENV=staging\n")

	mustLoadFresh(t)

	if got := env.AppEnv(); got != "production" {
		t.Fatalf("AppEnv() = %q, want production", got)
	}
	if got := os.Getenv("APP_ENV"); got != "production" {
		t.Fatalf("canonical APP_ENV = %q, want production", got)
	}
}

func TestEnv_SystemValueOverridesFiles(t *testing.T) {
	const key = "LUAS_TEST_SYSTEM_PRECEDENCE"
	useIsolatedEnvDir(t, key)

	writeEnvFile(t, ".env", key+"=base\n")
	writeEnvFile(t, ".env.production.local", key+"=environment_local\n")
	if err := os.Setenv("APP_ENV", "production"); err != nil {
		t.Fatalf("set APP_ENV: %v", err)
	}
	if err := os.Setenv(key, "system"); err != nil {
		t.Fatalf("set system override: %v", err)
	}

	mustLoadFresh(t)

	if got := env.Get(key); got != "system" {
		t.Fatalf("Get(%q) = %q, want %q", key, got, "system")
	}
}

func TestEnv_ExplicitFileOverridesStandardFiles(t *testing.T) {
	const key = "LUAS_TEST_EXPLICIT_FILE_PRECEDENCE"
	dir := useIsolatedEnvDir(t, key)

	writeEnvFile(t, ".env", key+"=base\n")
	writeEnvFile(t, ".env.production.local", key+"=environment_local\n")
	explicitPath := filepath.Join(dir, "runtime.env")
	writeEnvFile(t, explicitPath, key+"=explicit\n")
	if err := os.Setenv("APP_ENV", "production"); err != nil {
		t.Fatalf("set APP_ENV: %v", err)
	}
	if err := os.Setenv("LUAS_ENV_FILE", explicitPath); err != nil {
		t.Fatalf("set LUAS_ENV_FILE: %v", err)
	}

	mustLoadFresh(t)

	if got := env.Get(key); got != "explicit" {
		t.Fatalf("Get(%q) = %q, want %q", key, got, "explicit")
	}
}

func TestEnv_LoadFreshReplacesAndRemovesFileValues(t *testing.T) {
	const (
		changedKey = "LUAS_TEST_RELOADED_VALUE"
		removedKey = "LUAS_TEST_REMOVED_VALUE"
	)
	useIsolatedEnvDir(t, changedKey, removedKey)

	writeEnvFile(t, ".env", changedKey+"=first\n"+removedKey+"=present\n")
	mustLoadFresh(t)
	if got := env.Get(changedKey); got != "first" {
		t.Fatalf("initial Get(%q) = %q, want %q", changedKey, got, "first")
	}
	if got := env.Get(removedKey); got != "present" {
		t.Fatalf("initial Get(%q) = %q, want %q", removedKey, got, "present")
	}

	writeEnvFile(t, ".env", changedKey+"=second\n")
	mustLoadFresh(t)

	if got := env.Get(changedKey); got != "second" {
		t.Errorf("reloaded Get(%q) = %q, want %q", changedKey, got, "second")
	}
	if _, ok := os.LookupEnv(removedKey); ok {
		t.Errorf("removed file key %q is still present", removedKey)
	}
}

func TestEnv_LoadFreshReportsMalformedFile(t *testing.T) {
	useIsolatedEnvDir(t)
	writeEnvFile(t, ".env", "APP_NAME=Luas\nTHIS IS NOT AN ENV ENTRY\n")

	err := env.LoadFresh()

	if err == nil || !strings.Contains(err.Error(), ".env") {
		t.Fatalf("LoadFresh() error = %v, want malformed .env error", err)
	}
}

func TestEnv_LoadFreshRequiresExplicitFile(t *testing.T) {
	dir := useIsolatedEnvDir(t)
	missingPath := filepath.Join(dir, "missing.env")
	if err := os.Setenv("LUAS_ENV_FILE", missingPath); err != nil {
		t.Fatalf("set LUAS_ENV_FILE: %v", err)
	}

	err := env.LoadFresh()

	if err == nil || !strings.Contains(err.Error(), "LUAS_ENV_FILE") {
		t.Fatalf("LoadFresh() error = %v, want missing explicit file error", err)
	}
}

func TestEnv_GetOrFail(t *testing.T) {
	os.Setenv("REQUIRED_VAR", "required_value")
	defer os.Unsetenv("REQUIRED_VAR")

	value := env.GetOrFail("REQUIRED_VAR")
	if value != "required_value" {
		t.Errorf("Expected 'required_value', got '%s'", value)
	}
}

func TestEnv_GetOrFailPanic(t *testing.T) {
	os.Unsetenv("NON_EXISTENT_REQUIRED")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for missing required env var")
		}
	}()

	env.GetOrFail("NON_EXISTENT_REQUIRED")
}

func useIsolatedEnvDir(t *testing.T, keys ...string) string {
	t.Helper()

	names := append([]string{"APP_ENV", "GO_ENV", "GIN_MODE", "LUAS_ENV_FILE"}, keys...)
	type originalValue struct {
		value string
		set   bool
	}
	original := make(map[string]originalValue, len(names))
	for _, name := range names {
		value, set := os.LookupEnv(name)
		original[name] = originalValue{value: value, set: set}
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}

	// Register before t.Chdir so the working directory is restored before
	// resetting the process-wide loader to the repository environment.
	t.Cleanup(func() {
		for _, name := range names {
			previous := original[name]
			if previous.set {
				_ = os.Setenv(name, previous.value)
			} else {
				_ = os.Unsetenv(name)
			}
		}
		mustLoadFresh(t)
	})

	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}

func writeEnvFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustLoadFresh(t *testing.T) {
	t.Helper()
	if err := env.LoadFresh(); err != nil {
		t.Fatalf("env.LoadFresh() error = %v", err)
	}
}

func TestEnv_GetInt64(t *testing.T) {
	os.Setenv("TEST_INT64", "9223372036854775807")
	defer os.Unsetenv("TEST_INT64")

	value := env.GetInt64("TEST_INT64")
	if value != 9223372036854775807 {
		t.Errorf("Expected max int64, got %d", value)
	}
}

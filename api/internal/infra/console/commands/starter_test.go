package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zgiai/luas/api/internal/starter"
)

func TestParseStarterCommandOptions(t *testing.T) {
	options, err := parseStarterCommandOptions(
		[]string{"permission", "--env-file", "local.env", "--dry-run"},
		false,
		true,
	)
	if err != nil {
		t.Fatalf("parseStarterCommandOptions() error = %v", err)
	}
	if options.envFile != "local.env" || !options.dryRun || len(options.names) != 1 {
		t.Fatalf("options = %+v", options)
	}

	if _, err := parseStarterCommandOptions([]string{"--format=yaml"}, true, false); err == nil {
		t.Fatal("invalid list format was accepted")
	}
	if _, err := parseStarterCommandOptions(nil, false, true); err == nil {
		t.Fatal("missing starter names were accepted")
	}
}

func TestLoadStarterCatalogDocumentValidatesConfiguredDependencies(t *testing.T) {
	path := writeStarterEnvironment(t, "OPTIONAL_STARTERS=permission\n")
	if _, err := loadStarterCatalogDocument(path); err == nil || !strings.Contains(err.Error(), "requires \"organization\"") {
		t.Fatalf("loadStarterCatalogDocument() error = %v, want dependency error", err)
	}
}

func TestStarterCatalogDocumentHasStableJSONIdentity(t *testing.T) {
	path := writeStarterEnvironment(t, "OPTIONAL_STARTERS=organization,permission\n")
	document, err := loadStarterCatalogDocument(path)
	if err != nil {
		t.Fatalf("loadStarterCatalogDocument() error = %v", err)
	}
	if document.Kind != starterCatalogKind || document.SchemaVersion != starterCatalogSchemaVersion {
		t.Fatalf("document identity = %q/%d", document.Kind, document.SchemaVersion)
	}

	var output bytes.Buffer
	if err := writeStarterCatalogDocument(&output, document); err != nil {
		t.Fatalf("writeStarterCatalogDocument() error = %v", err)
	}
	var decoded starterCatalogDocument
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("catalog JSON is invalid: %v", err)
	}
	if len(decoded.Starters) < 3 || strings.Join(decoded.OptionalSelection, ",") != "organization,permission" {
		t.Fatalf("decoded document = %+v", decoded)
	}
	if decoded.Starters[0].Dependencies == nil || decoded.Starters[0].SeederNames == nil {
		t.Fatalf("catalog collections must encode as arrays: %+v", decoded.Starters[0])
	}
}

func TestEnableStartersAddsDependenciesAndWritesAtomically(t *testing.T) {
	path := writeStarterEnvironment(t, "APP_NAME=Luas\nOPTIONAL_STARTERS=\n")
	options := starterCommandOptions{envFile: path, names: []string{"permission"}}
	if err := changeStarterSelection(NewStarterEnableCommand().output, options, true); err != nil {
		t.Fatalf("changeStarterSelection() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated environment: %v", err)
	}
	if !strings.Contains(string(content), "OPTIONAL_STARTERS=organization,permission") {
		t.Fatalf("updated environment = %q", content)
	}
}

func TestEnableStartersRepairsMissingDependencies(t *testing.T) {
	path := writeStarterEnvironment(t, "export OPTIONAL_STARTERS=permission\n")
	options := starterCommandOptions{envFile: path, names: []string{"permission"}}
	if err := changeStarterSelection(NewStarterEnableCommand().output, options, true); err != nil {
		t.Fatalf("changeStarterSelection() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated environment: %v", err)
	}
	if string(content) != "OPTIONAL_STARTERS=organization,permission\n" {
		t.Fatalf("updated environment = %q", content)
	}
}

func TestDisableStartersProtectsDependenciesUnlessCascadeIsExplicit(t *testing.T) {
	catalog, err := starter.AvailableCatalog()
	if err != nil {
		t.Fatalf("starter.AvailableCatalog() error = %v", err)
	}
	current := []string{"organization", "permission", "setting"}

	if _, disableErr := disableStarters(catalog, current, []string{"organization"}, false); disableErr == nil {
		t.Fatal("disableStarters() accepted a required dependency without cascade")
	}
	remaining, err := disableStarters(catalog, current, []string{"organization"}, true)
	if err != nil {
		t.Fatalf("disableStarters() cascade error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %v, want no starters", remaining)
	}
}

func TestStarterChangeDryRunDoesNotWrite(t *testing.T) {
	path := writeStarterEnvironment(t, "OPTIONAL_STARTERS=\n")
	before, _ := os.ReadFile(path)
	options := starterCommandOptions{envFile: path, names: []string{"asset"}, dryRun: true}
	if err := changeStarterSelection(NewStarterEnableCommand().output, options, true); err != nil {
		t.Fatalf("changeStarterSelection() error = %v", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatalf("dry run changed the environment file: before=%q after=%q", before, after)
	}
}

func TestLoadStarterEnvironmentRefusesExampleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.example")
	if err := os.WriteFile(path, []byte("OPTIONAL_STARTERS=\n"), 0o600); err != nil {
		t.Fatalf("write example environment: %v", err)
	}
	if _, err := loadStarterEnvironment(path); err == nil {
		t.Fatal("loadStarterEnvironment() accepted .env.example")
	}
}

func writeStarterEnvironment(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write starter environment: %v", err)
	}
	return path
}

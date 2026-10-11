package pluginregistry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	startersFile = "api/internal/infra/config/starters.go"
	defaultsFile = "api/internal/starter/defaults.go"
	featuresFile = "web/src/config/optional-features.ts"
	testModule   = "github.com/fastagi/fastbuild/api"
)

var slotFixtures = map[string]string{
	startersFile: `package config

const (
	StarterUser = "user"
	// fastbuild:slot api.starter.constants
)

func StarterNames() []string {
	return []string{
		StarterUser,
		// fastbuild:slot api.starter.names
	}
}
`,
	defaultsFile: `package starter

import (
	"github.com/fastagi/fastbuild/api/internal/modules/user"
	// fastbuild:slot api.starter.imports
)

var ProviderSet = []any{
	user.ProviderSet,
	// fastbuild:slot api.starter.providers
}

type Handlers struct {
	User *user.Handler
	// fastbuild:slot api.starter.handlers
}

func OptionalManifests(h *Handlers) []any {
	return []any{
		// fastbuild:slot api.starter.manifests
	}
}
`,
	featuresFile: `export const OPTIONAL_WEB_FEATURES = [
  'organization',
  // fastbuild:slot web.features
] as const;
`,
	ContractFile: "openapi: 3.1.0\npaths:\n  /v1/existing:\n    get: {}\n",
}

// projectWithSlots returns a project root holding the slot files the fixtures wire.
func projectWithSlots(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for file, content := range slotFixtures {
		writeFile(t, root, file, content)
	}
	return root
}

func writeFile(t *testing.T, root, file, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(file))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func readFile(t *testing.T, root, file string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	require.NoError(t, err)
	return string(content)
}

// fakeContract merges by deleting the fragment, like the real tooling, and records removals.
type fakeContract struct {
	root     string
	mergeErr error
	merged   []string
	removed  []ContractAdditions
}

func (f *fakeContract) Merge(fragment string) (*ContractAdditions, error) {
	if f.mergeErr != nil {
		return nil, f.mergeErr
	}
	f.merged = append(f.merged, fragment)
	if err := os.Remove(filepath.Join(f.root, filepath.FromSlash(fragment))); err != nil {
		return nil, err
	}
	contract := filepath.Join(f.root, ContractFile)
	current, err := os.ReadFile(contract)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(contract, append(current, "  /v1/demo-notes:\n    get: {}\n"...), 0o644); err != nil {
		return nil, err
	}
	return &ContractAdditions{Paths: []string{"/v1/demo-notes"}, Schemas: []string{"DemoNote"}}, nil
}

func (f *fakeContract) Remove(additions ContractAdditions) error {
	f.removed = append(f.removed, additions)
	contract := filepath.Join(f.root, ContractFile)
	current, err := os.ReadFile(contract)
	if err != nil {
		return err
	}
	return os.WriteFile(contract, []byte(strings.Replace(string(current), "  /v1/demo-notes:\n    get: {}\n", "", 1)), 0o644)
}

func newInstaller(t *testing.T, root string) *Installer {
	t.Helper()
	return &Installer{
		Root:           root,
		Source:         writeFixtureIndex(t),
		ScaffoldModule: testModule,
		KnownStarters:  []string{"user", "audit", "organization"},
	}
}

func TestInstallWritesFilesWiresTheStarterThroughSlotsAndLocks(t *testing.T) {
	root := projectWithSlots(t)
	installer := newInstaller(t, root)

	result, err := installer.Install("demo-notes")
	require.NoError(t, err)
	assert.Len(t, result.Written, 2)
	assert.Equal(t, []string{startersFile, defaultsFile}, result.Edited())
	assert.FileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))

	starters := readFile(t, root, startersFile)
	assert.Contains(t, starters, "StarterDemoNotes = \"demonotes\"\n\t// fastbuild:slot api.starter.constants")
	assert.Contains(t, starters, "\t\tStarterDemoNotes,\n\t\t// fastbuild:slot api.starter.names")
	defaults := readFile(t, root, defaultsFile)
	assert.Contains(t, defaults, "\t\"github.com/fastagi/fastbuild/api/internal/modules/demonotes\"\n")
	assert.Contains(t, defaults, "\tdemonotes.ProviderSet,\n")
	assert.Contains(t, defaults, "\tDemoNotes *demonotes.Handler\n")
	assert.Contains(t, defaults, "\t\tdemonotes.NewStarterManifest(h.DemoNotes),\n")

	lock, err := ReadLock(root)
	require.NoError(t, err)
	locked := lock.Plugins["demo-notes"]
	assert.Equal(t, "0.1.0", locked.Version)
	assert.Equal(t, "demonotes", locked.Starter)
	assert.Len(t, locked.Files, 2)
	assert.Len(t, locked.Wiring, 6)

	_, err = installer.Install("demo-notes")
	assert.ErrorContains(t, err, "already installed")
}

func TestInstallRefusesToOverwriteAndChangesNothing(t *testing.T) {
	root := projectWithSlots(t)
	writeFile(t, root, "contracts/fragments/demo-notes.yaml", "mine\n")

	_, err := newInstaller(t, root).Install("demo-notes")
	require.ErrorContains(t, err, "would overwrite existing files")
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.NoFileExists(t, filepath.Join(root, LockFile))
	assert.Equal(t, "mine\n", readFile(t, root, "contracts/fragments/demo-notes.yaml"))
	assert.Equal(t, slotFixtures[startersFile], readFile(t, root, startersFile))
}

func TestInstallRollsBackEverythingWhenTheContractMergeFails(t *testing.T) {
	root := projectWithSlots(t)
	installer := newInstaller(t, root)
	installer.Contract = &fakeContract{root: root, mergeErr: errors.New("merge exploded")}

	_, err := installer.Install("demo-notes")
	require.ErrorContains(t, err, "merge exploded")
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.NoDirExists(t, filepath.Join(root, "api/internal/modules/demonotes"), "created directories are removed")
	for file, content := range slotFixtures {
		assert.Equal(t, content, readFile(t, root, file), file)
	}
	assert.NoFileExists(t, filepath.Join(root, LockFile))
}

func TestInstallFailsWithoutChangesWhenASlotIsMissing(t *testing.T) {
	root := projectWithSlots(t)
	writeFile(t, root, defaultsFile, "package starter\n")

	_, err := newInstaller(t, root).Install("demo-notes")
	require.ErrorContains(t, err, "slot marker not found: api.starter.imports")
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.Equal(t, slotFixtures[startersFile], readFile(t, root, startersFile))
}

func TestInstallChecksRequirementsBeforeWriting(t *testing.T) {
	root := projectWithSlots(t)
	installer := newInstaller(t, root)
	installer.KnownStarters = []string{"user", "audit"}
	installer.ScaffoldVersion = "v0.23.0"

	_, err := installer.Install("demo-notes")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `needs the "organization" starter`)
	assert.Contains(t, err.Error(), "needs FastBuild 0.24.0 or newer")
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))

	installer.KnownStarters = []string{"organization", "demonotes"}
	installer.ScaffoldVersion = ""
	_, err = installer.Install("demo-notes")
	assert.ErrorContains(t, err, `already has a "demonotes" starter`)
}

func TestMergedFragmentRecordsItsAdditionsAndIsNotTracked(t *testing.T) {
	root := projectWithSlots(t)
	installer := newInstaller(t, root)
	contract := &fakeContract{root: root}
	installer.Contract = contract

	result, err := installer.Install("demo-notes")
	require.NoError(t, err)
	assert.True(t, result.FragmentMerged)
	assert.Contains(t, result.Edited(), ContractFile)
	lock, _ := ReadLock(root)
	locked := lock.Plugins["demo-notes"]
	assert.Equal(t, "contracts/fragments/demo-notes.yaml", locked.Fragment)
	assert.NotEmpty(t, locked.FragmentChecksum)
	assert.Equal(t, []string{"/v1/demo-notes"}, locked.Contract.Paths)
	assert.NotContains(t, locked.Files, "contracts/fragments/demo-notes.yaml")
	for _, status := range Status(root, locked) {
		assert.Equal(t, "unchanged", status.State, status.Path)
	}
}

func TestStatusReportsModifiedAndMissingFiles(t *testing.T) {
	root := projectWithSlots(t)
	_, err := newInstaller(t, root).Install("demo-notes")
	require.NoError(t, err)
	writeFile(t, root, "api/internal/modules/demonotes/doc.go", "package demonotes\n")
	require.NoError(t, os.Remove(filepath.Join(root, "contracts/fragments/demo-notes.yaml")))

	lock, _ := ReadLock(root)
	states := map[string]string{}
	for _, status := range Status(root, lock.Plugins["demo-notes"]) {
		states[status.Path] = status.State
	}
	assert.Equal(t, "modified", states["api/internal/modules/demonotes/doc.go"])
	assert.Equal(t, "missing", states["contracts/fragments/demo-notes.yaml"])
}

func TestDryRunReportsFilesAndWiringWithoutWriting(t *testing.T) {
	root := projectWithSlots(t)
	installer := newInstaller(t, root)
	installer.DryRun = true
	result, err := installer.Install("demo-notes")
	require.NoError(t, err)
	assert.Len(t, result.Written, 2)
	assert.Len(t, result.Wiring, 6)
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.NoFileExists(t, filepath.Join(root, LockFile))
	assert.Equal(t, slotFixtures[defaultsFile], readFile(t, root, defaultsFile))
}

func TestTransformAdaptsFilesAndStarterImports(t *testing.T) {
	root := projectWithSlots(t)
	installer := newInstaller(t, root)
	installer.Transform = func(_ string, content []byte) []byte {
		return []byte(strings.ReplaceAll(string(content), testModule, "github.com/acme/platform/api"))
	}
	_, err := installer.Install("demo-notes")
	require.NoError(t, err)
	written := readFile(t, root, "api/internal/modules/demonotes/doc.go")
	assert.Contains(t, written, "github.com/acme/platform/api/internal/domain")
	assert.Contains(t, readFile(t, root, defaultsFile), "\"github.com/acme/platform/api/internal/modules/demonotes\"")
	lock, _ := ReadLock(root)
	assert.Equal(t, Checksum([]byte(written)), lock.Plugins["demo-notes"].Files["api/internal/modules/demonotes/doc.go"])
}

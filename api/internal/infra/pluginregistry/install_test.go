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

const wiringFile = "api/internal/starter/defaults.go"

func projectWithWiring(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "api/internal/starter"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, wiringFile), []byte("package starter\n"), 0o644))
	return root
}

func newInstaller(t *testing.T, root string) *Installer {
	t.Helper()
	return &Installer{
		Root:          root,
		Source:        writeFixtureIndex(t),
		KnownStarters: []string{"user", "audit", "organization"},
		EditedFiles:   []string{wiringFile},
		WireStarter: func(spec StarterSpec) ([]string, error) {
			path := filepath.Join(root, wiringFile)
			current, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			return []string{wiringFile}, os.WriteFile(path, append(current, []byte("// "+spec.Name+"\n")...), 0o644)
		},
	}
}

func TestInstallWritesFilesWiresTheStarterAndLocks(t *testing.T) {
	root := projectWithWiring(t)
	installer := newInstaller(t, root)

	result, err := installer.Install("demo-notes")
	require.NoError(t, err)
	assert.Len(t, result.Written, 2)
	assert.Equal(t, []string{wiringFile}, result.Edited)
	assert.FileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	wiring, _ := os.ReadFile(filepath.Join(root, wiringFile))
	assert.Contains(t, string(wiring), "// demonotes")

	lock, err := ReadLock(root)
	require.NoError(t, err)
	locked := lock.Plugins["demo-notes"]
	assert.Equal(t, "0.1.0", locked.Version)
	assert.Equal(t, "demonotes", locked.Starter)
	assert.Len(t, locked.Files, 2)

	_, err = installer.Install("demo-notes")
	assert.ErrorContains(t, err, "already installed")
}

func TestInstallRefusesToOverwriteAndChangesNothing(t *testing.T) {
	root := projectWithWiring(t)
	conflict := filepath.Join(root, "contracts/fragments/demo-notes.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflict), 0o755))
	require.NoError(t, os.WriteFile(conflict, []byte("mine\n"), 0o644))

	_, err := newInstaller(t, root).Install("demo-notes")
	require.ErrorContains(t, err, "would overwrite existing files")
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.NoFileExists(t, filepath.Join(root, LockFile))
	mine, _ := os.ReadFile(conflict)
	assert.Equal(t, "mine\n", string(mine))
}

func TestInstallRollsBackWhenAHookFails(t *testing.T) {
	root := projectWithWiring(t)
	installer := newInstaller(t, root)
	installer.MergeFragment = func(string) error {
		// The starter hook already edited the wiring file; the merge fails afterwards.
		return errors.New("merge exploded")
	}

	_, err := installer.Install("demo-notes")
	require.ErrorContains(t, err, "merge exploded")
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	wiring, _ := os.ReadFile(filepath.Join(root, wiringFile))
	assert.Equal(t, "package starter\n", string(wiring), "edited files are restored")
	assert.NoFileExists(t, filepath.Join(root, LockFile))
}

func TestInstallChecksRequirementsBeforeWriting(t *testing.T) {
	root := projectWithWiring(t)
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

func TestMergedFragmentIsRecordedButNotTracked(t *testing.T) {
	root := projectWithWiring(t)
	installer := newInstaller(t, root)
	installer.MergeFragment = func(fragment string) error {
		return os.Remove(filepath.Join(root, filepath.FromSlash(fragment)))
	}
	result, err := installer.Install("demo-notes")
	require.NoError(t, err)
	assert.True(t, result.FragmentMerged)
	lock, _ := ReadLock(root)
	locked := lock.Plugins["demo-notes"]
	assert.Equal(t, "contracts/fragments/demo-notes.yaml", locked.Fragment)
	assert.NotContains(t, locked.Files, "contracts/fragments/demo-notes.yaml")
	for _, status := range Status(root, locked) {
		assert.Equal(t, "unchanged", status.State, status.Path)
	}
}

func TestStatusReportsModifiedAndMissingFiles(t *testing.T) {
	root := projectWithWiring(t)
	_, err := newInstaller(t, root).Install("demo-notes")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "api/internal/modules/demonotes/doc.go"), []byte("package demonotes\n"), 0o644))
	require.NoError(t, os.Remove(filepath.Join(root, "contracts/fragments/demo-notes.yaml")))

	lock, _ := ReadLock(root)
	states := map[string]string{}
	for _, status := range Status(root, lock.Plugins["demo-notes"]) {
		states[status.Path] = status.State
	}
	assert.Equal(t, "modified", states["api/internal/modules/demonotes/doc.go"])
	assert.Equal(t, "missing", states["contracts/fragments/demo-notes.yaml"])
}

func TestDryRunReportsWithoutWriting(t *testing.T) {
	root := projectWithWiring(t)
	installer := newInstaller(t, root)
	installer.DryRun = true
	result, err := installer.Install("demo-notes")
	require.NoError(t, err)
	assert.Len(t, result.Written, 2)
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.NoFileExists(t, filepath.Join(root, LockFile))
}

func TestTransformAdaptsFilesAndTheLockRecordsTheResult(t *testing.T) {
	root := projectWithWiring(t)
	installer := newInstaller(t, root)
	installer.Transform = func(_ string, content []byte) []byte {
		return append([]byte("// adapted\n"), content...)
	}
	_, err := installer.Install("demo-notes")
	require.NoError(t, err)
	written, _ := os.ReadFile(filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	assert.True(t, strings.HasPrefix(string(written), "// adapted\n"))
	lock, _ := ReadLock(root)
	assert.Equal(t, Checksum(written), lock.Plugins["demo-notes"].Files["api/internal/modules/demonotes/doc.go"])
}

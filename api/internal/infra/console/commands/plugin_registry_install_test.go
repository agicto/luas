package commands

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
)

// TestOfficialPluginsInstallBuildAndTest installs every registry plugin into a scratch copy of the
// API wiring, compiles the API with the installed files overlaid, and runs the plugin's own tests,
// so a core change that breaks an official plugin fails here instead of in a downstream project.
// The plugin's PostgreSQL tests run when FASTBUILD_TEST_POSTGRES_DSN is set, as in CI.
func TestOfficialPluginsInstallBuildAndTest(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the API once per plugin")
	}
	apiRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	require.NoError(t, err)
	repoRoot := filepath.Dir(apiRoot)
	source := pluginregistry.NewLocalSource(filepath.Join(repoRoot, registryDir), "local:registry")
	index, err := source.Index()
	require.NoError(t, err)

	for _, entry := range index.Plugins {
		t.Run(entry.Name, func(t *testing.T) {
			scratch := t.TempDir()
			for _, file := range pluginEditedFiles {
				raw, readErr := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(file)))
				require.NoError(t, readErr)
				target := filepath.Join(scratch, filepath.FromSlash(file))
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
				require.NoError(t, os.WriteFile(target, raw, 0o644))
			}
			require.NoError(t, os.WriteFile(filepath.Join(scratch, "api", "go.mod"), []byte("module "+scaffoldModulePath+"\n"), 0o644))

			installer, setupErr := newProjectInstaller(scratch, source)
			require.NoError(t, setupErr)
			installer.MergeFragment = nil // the contracts check merges and lints fragments
			result, installErr := installer.Install(entry.Name)
			require.NoError(t, installErr)

			replace := map[string]string{}
			for _, file := range append(append([]string{}, result.Written...), result.Edited...) {
				if strings.HasPrefix(file, "api/") {
					replace[filepath.Join(repoRoot, filepath.FromSlash(file))] = filepath.Join(scratch, filepath.FromSlash(file))
				}
			}
			overlay, marshalErr := json.Marshal(map[string]any{"Replace": replace})
			require.NoError(t, marshalErr)
			overlayPath := filepath.Join(t.TempDir(), "overlay.json")
			require.NoError(t, os.WriteFile(overlayPath, overlay, 0o644))

			packages := []string{"./internal/starter", "./internal/infra/config", "./database/migrations", "./internal/wiring"}
			var pluginPackage string
			if entry.Starter != nil {
				pluginPackage = "./internal/modules/" + entry.Starter.Directory
				packages = append(packages, pluginPackage)
			}
			run := func(args ...string) {
				command := exec.Command("go", args...)
				command.Dir = apiRoot
				output, cmdErr := command.CombinedOutput()
				require.NoError(t, cmdErr, "go %s\n%s", strings.Join(args, " "), output)
			}
			run(append([]string{"build", "-overlay=" + overlayPath}, packages...)...)
			if pluginPackage == "" {
				return
			}
			// go vet cannot enter a directory that exists only in the overlay, so compile the plugin's
			// tests with vet disabled and run the binary from the scratch copy.
			binary := filepath.Join(t.TempDir(), entry.Name+".test")
			run("test", "-c", "-vet=off", "-overlay="+overlayPath, "-o", binary, pluginPackage)
			tests := exec.Command(binary, "-test.count=1")
			tests.Dir = filepath.Join(scratch, "api", "internal", "modules", entry.Starter.Directory)
			output, testErr := tests.CombinedOutput()
			require.NoError(t, testErr, "%s tests\n%s", entry.Name, output)
		})
	}
}

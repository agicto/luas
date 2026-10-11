package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
)

// The committed registry/index.json must match the plugin manifests and files it describes.
func TestPluginRegistryIndexIsCurrent(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "..", "..", registryDir)
	index, err := pluginregistry.BuildIndex(dir)
	require.NoError(t, err)
	want, err := index.Marshal()
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(dir, pluginregistry.IndexFile))
	require.NoError(t, err)
	require.Equal(t, string(want), string(got), "registry/index.json is stale; run: cd api && go run ./cmd/fastbuild plugin:index")
}

// fixtureRegistry copies the pluginregistry test registry and writes its index.
func fixtureRegistry(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.CopyFS(dir, os.DirFS(filepath.Join("..", "..", "pluginregistry", "testdata", "registry"))))
	index, err := pluginregistry.BuildIndex(dir)
	require.NoError(t, err)
	raw, err := index.Marshal()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginregistry.IndexFile), raw, 0o644))
	return dir
}

func captureStdout(t *testing.T, run func() error) string {
	t.Helper()
	previous := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	runErr := run()
	require.NoError(t, writer.Close())
	os.Stdout = previous
	out, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, runErr)
	return string(out)
}

func TestPluginSearchReturnsRankedJSONForAgents(t *testing.T) {
	registry := fixtureRegistry(t)
	out := captureStdout(t, func() error {
		return NewPluginSearchCommand().Run([]string{"笔记", "--source=" + registry, "--format=json"})
	})
	var body struct {
		Source  string `json:"source"`
		Plugins []struct {
			Name     string   `json:"name"`
			UseWhen  []string `json:"use_when"`
			Requires []string `json:"requires_starters"`
		} `json:"plugins"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &body))
	require.Len(t, body.Plugins, 1)
	assert.Equal(t, "demo-notes", body.Plugins[0].Name)
	assert.Equal(t, []string{"organization"}, body.Plugins[0].Requires)
	assert.NotEmpty(t, body.Plugins[0].UseWhen)
}

func TestPluginInfoIncludesIntegrationSteps(t *testing.T) {
	registry := fixtureRegistry(t)
	out := captureStdout(t, func() error {
		return NewPluginInfoCommand().Run([]string{"demo-notes", "--source=" + registry, "--format=json"})
	})
	var body struct {
		Plugin      pluginregistry.IndexEntry `json:"plugin"`
		Integration string                    `json:"integration"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &body))
	assert.Equal(t, "demonotes", body.Plugin.Starter.Name)
	assert.Contains(t, body.Integration, "Enable the starter")
	assert.Error(t, NewPluginInfoCommand().Run([]string{"missing", "--source=" + registry}))
}

// downstreamProject builds a minimal checkout of a project renamed by project:init, with the real
// starter wiring files so plugin:add edits them exactly as it would in a project.
func downstreamProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(relative, content string) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	write("api/go.mod", "module github.com/example-demo/platform/api\n\ngo 1.26.9\n")
	write("contracts/openapi.yaml", "openapi: 3.1.0\npaths: {}\n")
	for _, file := range []string{"internal/infra/config/starters.go", "internal/starter/defaults.go"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", filepath.FromSlash(file)))
		require.NoError(t, err)
		write("api/"+file, string(raw))
	}
	return root
}

func TestPluginAddInstallsAdaptsWiresAndLocks(t *testing.T) {
	scaffoldAPIRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	require.NoError(t, err)
	registry := fixtureRegistry(t)
	root := downstreamProject(t)
	t.Chdir(filepath.Join(root, "api"))

	require.NoError(t, NewPluginAddCommand().Run([]string{"demo-notes", "--source=" + registry, "--dry-run"}))
	assert.NoFileExists(t, filepath.Join(root, "api/internal/modules/demonotes/doc.go"), "dry run writes nothing")

	captureStdout(t, func() error {
		return NewPluginAddCommand().Run([]string{"demo-notes", "--source=" + registry})
	})

	doc, err := os.ReadFile(filepath.Join(root, "api/internal/modules/demonotes/doc.go"))
	require.NoError(t, err)
	assert.Contains(t, string(doc), `"github.com/example-demo/platform/api/internal/domain"`, "imports follow the project's module")
	assert.NotContains(t, string(doc), "fastagi/fastbuild")

	starters, _ := os.ReadFile(filepath.Join(root, "api/internal/infra/config/starters.go"))
	assert.Contains(t, string(starters), `StarterDemoNotes`)
	defaults, _ := os.ReadFile(filepath.Join(root, "api/internal/starter/defaults.go"))
	assert.Contains(t, string(defaults), "demonotes.ProviderSet")
	assert.Contains(t, string(defaults), "demonotes.NewStarterManifest(h.DemoNotes)")

	out := captureStdout(t, func() error { return NewPluginStatusCommand().Run([]string{"--format=json"}) })
	var status struct {
		Plugins []struct {
			Name  string                      `json:"name"`
			Files []pluginregistry.FileStatus `json:"files"`
		} `json:"plugins"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &status))
	require.Len(t, status.Plugins, 1)
	for _, file := range status.Plugins[0].Files {
		assert.Equal(t, "unchanged", file.State, file.Path)
	}

	err = NewPluginAddCommand().Run([]string{"demo-notes", "--source=" + registry})
	assert.ErrorContains(t, err, "already installed")

	// Without contract tooling the fragment waits to be merged, and the doctor says so.
	doctorErr := NewPluginDoctorCommand().Run(nil)
	assert.ErrorContains(t, doctorErr, "1 plugin integration problems")

	captureStdout(t, func() error { return NewPluginRemoveCommand().Run([]string{"demo-notes"}) })
	for _, file := range []string{"internal/infra/config/starters.go", "internal/starter/defaults.go"} {
		original, readErr := os.ReadFile(filepath.Join(scaffoldAPIRoot, filepath.FromSlash(file)))
		require.NoError(t, readErr)
		current, _ := os.ReadFile(filepath.Join(root, "api", filepath.FromSlash(file)))
		assert.Equal(t, string(original), string(current), "plugin:remove restores %s", file)
	}
	assert.NoDirExists(t, filepath.Join(root, "api/internal/modules/demonotes"))
	assert.NoFileExists(t, filepath.Join(root, pluginregistry.LockFile))
}

// TestProjectSlotsAreMarkedExactlyOnce keeps the slot catalog and the markers in the source aligned:
// every cataloged slot is marked once in its file, and no file marks an uncataloged slot.
func TestProjectSlotsAreMarkedExactlyOnce(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	require.NoError(t, err)
	found, err := pluginregistry.ScanSlotMarkers(repoRoot, "api/internal", "web/src", "admin/src")
	require.NoError(t, err)
	for _, slot := range pluginregistry.ProjectSlots {
		assert.Equal(t, []string{slot.File}, found[slot.Name], "slot %s", slot.Name)
		delete(found, slot.Name)
	}
	assert.Empty(t, found, "markers without a catalog entry in pluginregistry.ProjectSlots")
}

func TestCommunitySourcesNeedExplicitTrust(t *testing.T) {
	community, err := pluginregistry.ParseRemoteSource("github:someone/plugins@v1.0.0")
	require.NoError(t, err)
	official, err := pluginregistry.ParseRemoteSource("github:" + scaffoldGitHubRepository + "@v0.25.0")
	require.NoError(t, err)
	assert.Equal(t, trustCommunity, sourceTrust(community))
	assert.Equal(t, trustOfficial, sourceTrust(official))
	assert.Equal(t, trustLocal, sourceTrust(pluginregistry.NewLocalSource(t.TempDir(), "path:./mine")))

	err = requireTrust(community, nil)
	require.ErrorContains(t, err, "https://github.com/someone/plugins/tree/v1.0.0/registry")
	assert.NoError(t, requireTrust(community, []string{"--trust"}))
	assert.NoError(t, requireTrust(official, nil))
}

// TestPluginSearchSelectsTheRightPlugin replays real requests, in English and Chinese, against the
// official registry: an agent that installs the top result must pick the expected plugin. Add a
// case whenever a plugin is added or a request was matched wrongly.
func TestPluginSearchSelectsTheRightPlugin(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	require.NoError(t, err)
	index, err := pluginregistry.NewLocalSource(filepath.Join(repoRoot, registryDir), "local:registry").Index()
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(repoRoot, registryDir, "evals", "search-cases.tsv"))
	require.NoError(t, err)

	cases, correct := 0, 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n")[1:] {
		query, expected, ok := strings.Cut(line, "\t")
		require.True(t, ok, "malformed case %q", line)
		_, known := index.Find(expected)
		require.True(t, known, "case expects unknown plugin %q", expected)
		cases++
		results := index.Search(query)
		if len(results) > 0 && results[0].Entry.Name == expected {
			correct++
			continue
		}
		top := "nothing"
		if len(results) > 0 {
			top = results[0].Entry.Name
		}
		t.Errorf("%q: want %s, top result %s", query, expected, top)
	}
	t.Logf("plugin selection: %d/%d correct", correct, cases)
}

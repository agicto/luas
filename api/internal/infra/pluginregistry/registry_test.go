package pluginregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureRegistry = "testdata/registry"

func writeFixtureIndex(t *testing.T) *LocalSource {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.CopyFS(dir, os.DirFS(fixtureRegistry)))
	index, err := BuildIndex(dir)
	require.NoError(t, err)
	raw, err := index.Marshal()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, IndexFile), raw, 0o644))
	return NewLocalSource(dir, "")
}

func TestBuildIndexChecksumsEveryShippedFile(t *testing.T) {
	index, err := BuildIndex(fixtureRegistry)
	require.NoError(t, err)
	require.Len(t, index.Plugins, 1)
	entry := index.Plugins[0]
	assert.Equal(t, "demo-notes", entry.Name)
	assert.Len(t, entry.Files, 2)
	content, err := os.ReadFile(PluginFilePath(fixtureRegistry, "demo-notes", "api/internal/modules/demonotes/doc.go"))
	require.NoError(t, err)
	assert.Equal(t, Checksum(content), entry.Files["api/internal/modules/demonotes/doc.go"])
}

func TestLocalSourceRejectsAFileThatDoesNotMatchTheIndex(t *testing.T) {
	source := writeFixtureIndex(t)
	index, err := source.Index()
	require.NoError(t, err)
	entry, ok := index.Find("demo-notes")
	require.True(t, ok)

	_, err = source.ReadFile(entry, "api/internal/modules/demonotes/doc.go")
	require.NoError(t, err)

	tampered := PluginFilePath(source.Dir, "demo-notes", "api/internal/modules/demonotes/doc.go")
	require.NoError(t, os.WriteFile(tampered, []byte("package demonotes\n// changed\n"), 0o644))
	_, err = source.ReadFile(entry, "api/internal/modules/demonotes/doc.go")
	require.ErrorContains(t, err, "does not match the registry checksum")

	_, err = source.ReadFile(entry, "api/not-listed.go")
	require.ErrorContains(t, err, "does not list")
}

func TestSearchRanksNameAndKeywordsAboveDescriptions(t *testing.T) {
	index := &Index{Version: 1, Plugins: []IndexEntry{
		{Manifest: Manifest{Name: "knowledge-base", Summary: "Searchable articles", Keywords: []string{"docs", "知识库"}}},
		{Manifest: Manifest{Name: "billing", Summary: "Plans", Description: "Links invoices to knowledge of usage"}},
	}}
	results := index.Search("knowledge")
	require.Len(t, results, 2)
	assert.Equal(t, "knowledge-base", results[0].Entry.Name)

	chinese := index.Search("知识库")
	require.Len(t, chinese, 1)
	assert.Equal(t, "knowledge-base", chinese[0].Entry.Name)

	assert.Len(t, index.Search(""), 2, "an empty query lists every plugin")
	assert.Empty(t, index.Search("payments"))
}

func TestManifestValidationReportsEveryProblem(t *testing.T) {
	err := Manifest{Name: "Bad Name", Version: "one", FastBuild: "", Fragment: "api/x.yaml",
		Starter: &StarterSpec{Name: "x", Const: "Knowledge", Directory: "x", Handler: "x"}}.Validate()
	require.Error(t, err)
	for _, want := range []string{"name must", "version must", "fastbuild must", "summary", "use_when",
		"starter.const", "starter.handler", "fragment must"} {
		assert.Contains(t, err.Error(), want)
	}
}

func TestValidFilePathRejectsEscapes(t *testing.T) {
	for _, bad := range []string{"", "/etc/passwd", "../x", "a/../../x", "a\\b", ".git/config", "./a", "a//b"} {
		assert.False(t, ValidFilePath(bad), bad)
	}
	assert.True(t, ValidFilePath("api/internal/modules/x/service.go"))
}

func TestParseVersionAcceptsGitDescribeOutput(t *testing.T) {
	version, err := ParseVersion("v0.24.0-3-g614077d")
	require.NoError(t, err)
	assert.Equal(t, "0.24.0", version.String())
	minimum, _ := ParseVersion("0.25.0")
	assert.False(t, version.AtLeast(minimum))
	assert.True(t, minimum.AtLeast(version))
	_, err = ParseVersion("main")
	assert.Error(t, err)
}

func TestReadDocumentStaysInsideThePluginDirectory(t *testing.T) {
	source := writeFixtureIndex(t)
	doc, err := source.ReadDocument("demo-notes", IntegrationFile)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(doc), "# Integration"))
	_, err = source.ReadDocument("demo-notes", "../index.json")
	assert.Error(t, err)
}

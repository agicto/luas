package pluginregistry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/infra/projectfiles"
)

type archiveEntry struct {
	name     string
	body     string
	typeflag byte
	linkname string
}

func tarGz(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.body)), Typeflag: typeflag, Linkname: entry.linkname}
		if typeflag != tar.TypeReg {
			header.Size = 0
		}
		require.NoError(t, writer.WriteHeader(header))
		if typeflag == tar.TypeReg {
			_, err := writer.Write([]byte(entry.body))
			require.NoError(t, err)
		}
	}
	require.NoError(t, writer.Close())
	require.NoError(t, gz.Close())
	return buffer.Bytes()
}

// fixtureArchive packs the fixture registry the way GitHub packs a release.
func fixtureArchive(t *testing.T) []byte {
	t.Helper()
	source := writeFixtureIndex(t)
	var entries []archiveEntry
	entries = append(entries, archiveEntry{name: "fastbuild-v0.25.0/README.md", body: "core"})
	require.NoError(t, filepath.WalkDir(source.Dir, func(current string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(source.Dir, current)
		content, readErr := os.ReadFile(current)
		require.NoError(t, readErr)
		entries = append(entries, archiveEntry{name: "fastbuild-v0.25.0/registry/" + filepath.ToSlash(relative), body: string(content)})
		return nil
	}))
	return tarGz(t, entries...)
}

func TestRemoteSourceDownloadsAndVerifiesTheReleaseRegistry(t *testing.T) {
	archive := fixtureArchive(t)
	var requested string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	previous := projectfiles.GitHubArchiveBaseURL
	projectfiles.GitHubArchiveBaseURL = server.URL
	t.Cleanup(func() { projectfiles.GitHubArchiveBaseURL = previous })

	source, err := ParseRemoteSource("github:fastagi/fastbuild@v0.25.0")
	require.NoError(t, err)
	defer source.Close()
	index, err := source.Index()
	require.NoError(t, err)
	assert.Equal(t, "/fastagi/fastbuild/tar.gz/v0.25.0", requested)
	entry, ok := index.Find("demo-notes")
	require.True(t, ok)
	content, err := source.ReadFile(entry, "contracts/fragments/demo-notes.yaml")
	require.NoError(t, err)
	assert.Equal(t, "paths: {}\n", string(content))
	assert.Equal(t, "github:fastagi/fastbuild@v0.25.0", source.Describe())

	dir := source.local.Dir
	require.NoError(t, source.Close())
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "Close removes the extracted archive")
}

func TestExtractRegistryRejectsUnsafeArchives(t *testing.T) {
	cases := map[string][]archiveEntry{
		"path traversal": {{name: "repo-v1/registry/../../escape.txt", body: "x"}},
		"symlink":        {{name: "repo-v1/registry/plugins/a/link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}},
		"no registry":    {{name: "repo-v1/README.md", body: "x"}},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			err := ExtractRegistry(bytes.NewReader(tarGz(t, entries...)), t.TempDir())
			assert.Error(t, err)
		})
	}
	assert.Error(t, ExtractRegistry(strings.NewReader("not gzip"), t.TempDir()))
}

func TestParseRemoteSourceRequiresAPinnedGitHubReference(t *testing.T) {
	for _, bad := range []string{"github:fastagi/fastbuild", "https://github.com/fastagi/fastbuild", "github:a/b@../x", "gitlab:a/b@v1"} {
		_, err := ParseRemoteSource(bad)
		assert.Error(t, err, bad)
	}
}

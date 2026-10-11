package projectfiles

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		target := filepath.Join(dir, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
		require.NoError(t, os.WriteFile(target, []byte(content), 0o644))
	}
	return dir
}

func read(t *testing.T, root, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	require.NoError(t, err)
	return string(content)
}

const (
	lines1 = "one\ntwo\nthree\nfour\nfive\n"
	lines2 = "one\ntwo\nthree\nfour\nFIVE\n"
)

func TestUpgradeKeepsProjectWorkAndTakesReleaseChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for three-way merges")
	}
	base := tree(t, map[string]string{
		"untouched.txt": "v1\n", "edited.txt": lines1, "clash.txt": "a\n", "gone.txt": "old\n",
		"deleted-here.txt": "x\n", "kept-edit.txt": "k\n", "same.txt": "same\n", "go.mod": "module scaffold/api\n",
		"registry/index.json": "{}\n",
	})
	target := tree(t, map[string]string{
		"untouched.txt": "v2\n", "edited.txt": lines2, "clash.txt": "b\n", "new.txt": "fresh\n",
		"deleted-here.txt": "y\n", "same.txt": "same\n", "go.mod": "module scaffold/api\n\ngo 1.27\n",
		"registry/index.json": "{\"v\":2}\n",
	})
	root := tree(t, map[string]string{
		"untouched.txt": "v1\n", "edited.txt": "ONE\ntwo\nthree\nfour\nfive\n", "clash.txt": "mine\n",
		"gone.txt": "old\n", "kept-edit.txt": "mine\n", "same.txt": "same\n", "go.mod": "module acme/api\n",
		"marker.json": "{\"version\":\"1\"}\n",
	})
	upgrade := &Upgrade{
		Root: root, Base: base, Target: target,
		Adapt: func(_ string, content []byte) []byte {
			return []byte(strings.ReplaceAll(string(content), "scaffold/api", "acme/api"))
		},
		Skip:  func(path string) bool { return strings.HasPrefix(path, "registry/") },
		Extra: map[string][]byte{"marker.json": []byte("{\"version\":\"2\"}\n")},
	}

	dry := *upgrade
	dry.DryRun = true
	planned, err := dry.Run()
	require.NoError(t, err)
	assert.Equal(t, "v1\n", read(t, root, "untouched.txt"), "a dry run writes nothing")

	result, err := upgrade.Run()
	require.NoError(t, err)
	assert.Equal(t, planned, result)
	assert.Equal(t, []string{"new.txt"}, result.Added)
	assert.Equal(t, []string{"go.mod", "untouched.txt"}, result.Replaced)
	assert.Equal(t, []string{"edited.txt"}, result.Merged)
	assert.Equal(t, []string{"clash.txt"}, result.Conflicts)
	assert.Equal(t, []string{"gone.txt"}, result.Deleted)
	assert.Equal(t, []string{"deleted-here.txt", "kept-edit.txt"}, result.Kept)

	assert.Equal(t, "module acme/api\n\ngo 1.27\n", read(t, root, "go.mod"), "release content is adapted to the project")
	assert.Equal(t, "ONE\ntwo\nthree\nfour\nFIVE\n", read(t, root, "edited.txt"))
	assert.Contains(t, read(t, root, "clash.txt"), ConflictMarker)
	assert.NoFileExists(t, filepath.Join(root, "gone.txt"))
	assert.NoFileExists(t, filepath.Join(root, "deleted-here.txt"))
	assert.NoDirExists(t, filepath.Join(root, "registry"))
	assert.Equal(t, "{\"version\":\"2\"}\n", read(t, root, "marker.json"))
}

func TestUpgradeSavesUnmergeableFilesBesideTheProjectCopy(t *testing.T) {
	base := tree(t, map[string]string{"logo.png": "\x00v1"})
	target := tree(t, map[string]string{"logo.png": "\x00v2", "notes.txt": "upstream\n"})
	root := tree(t, map[string]string{"logo.png": "\x00mine", "notes.txt": "project\n"})

	result, err := (&Upgrade{Root: root, Base: base, Target: target}).Run()
	require.NoError(t, err)
	assert.Equal(t, []string{"logo.png", "notes.txt"}, result.Incoming)
	assert.Equal(t, "\x00mine", read(t, root, "logo.png"))
	assert.Equal(t, "\x00v2", read(t, root, "logo.png"+IncomingSuffix))
	assert.Equal(t, "upstream\n", read(t, root, "notes.txt"+IncomingSuffix))
}

func TestChangeSetRollsBackEverythingItWrote(t *testing.T) {
	root := tree(t, map[string]string{"keep.txt": "original\n"})
	changes := NewChangeSet(root)
	changes.Write("keep.txt", []byte("changed\n"))
	changes.Write("new/dir/file.txt", []byte("new\n"))
	require.NoError(t, changes.Commit())
	assert.Equal(t, "changed\n", read(t, root, "keep.txt"))

	require.NoError(t, changes.Rollback())
	assert.Equal(t, "original\n", read(t, root, "keep.txt"))
	assert.NoDirExists(t, filepath.Join(root, "new"), "created directories are removed")
}

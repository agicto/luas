package commands

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
)

func projectFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"api/go.mod":                   "module " + scaffoldModulePath + "\n\ngo 1.25\n",
		"api/main.go":                  "package main\n\nimport _ \"" + scaffoldModulePath + "/internal/x\"\n",
		"api/.env.example":             "APP_NAME=FastBuild\nOPTIONAL_STARTERS=\n",
		"admin/.env.example":           "VITE_APP_NAME=FastBuild\nVITE_OPTIONAL_FEATURES=\n",
		"web/.env.example":             "NEXT_PUBLIC_OPTIONAL_FEATURES=\n",
		"web/package.json":             "{\n  \"name\": \"" + scaffoldSlug + "-web\"\n}\n",
		"admin/package.json":           "{\n  \"name\": \"" + scaffoldSlug + "-admin\"\n}\n",
		"contracts/package.json":       "{\n  \"name\": \"@" + scaffoldSlug + "/contracts\"\n}\n",
		"contracts/openapi.yaml":       "openapi: 3.1.0\n",
		"compose.dev.yaml":             "name: " + scaffoldSlug + "-dev\n",
		"web/Dockerfile":               "ARG OCI_SOURCE=" + scaffoldRepository + "\n",
		"scripts/dev.sh":               "S=\"${FASTBUILD_DEV_STARTERS:-a,b}\"\nW=\"${FASTBUILD_DEV_WEB_FEATURES:-a}\"\nA=\"${FASTBUILD_DEV_ADMIN_FEATURES:-b}\"\n",
		"docs/untracked-is-ignored.md": scaffoldModulePath,
		"api/assets/binary.bin":        "\x00" + scaffoldModulePath,
		"README.md":                    "FastBuild scaffold docs stay as written.\n",
		"registry/index.json":          "{\"version\": 1, \"plugins\": []}\n",
	}
	for path, content := range files {
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "--", ".", ":!docs/untracked-is-ignored.md"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		output, err := command.CombinedOutput()
		require.NoError(t, err, string(output))
	}
	return root
}

func read(t *testing.T, root, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, path))
	require.NoError(t, err)
	return string(content)
}

func TestInitProjectRewritesIdentityAndSelectsStarters(t *testing.T) {
	root := projectFixture(t)
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	marker, changed, err := InitProject(root, ProjectOptions{
		Name:     "Acme Platform",
		Slug:     "acme",
		Module:   "github.com/example-demo/platform/api",
		Starters: []string{"organization", "webhook", "operator"},
	}, "v0.21.1", now)
	require.NoError(t, err)
	assert.Equal(t, 11, changed)

	assert.Contains(t, read(t, root, "api/go.mod"), "module github.com/example-demo/platform/api")
	assert.Contains(t, read(t, root, "api/main.go"), `"github.com/example-demo/platform/api/internal/x"`)
	assert.Equal(t, "APP_NAME=Acme Platform\nOPTIONAL_STARTERS=organization,webhook,operator\n", read(t, root, "api/.env.example"))
	assert.Equal(t, "VITE_APP_NAME=Acme Platform\nVITE_OPTIONAL_FEATURES=operator,organization,webhook\n", read(t, root, "admin/.env.example"))
	assert.Equal(t, "NEXT_PUBLIC_OPTIONAL_FEATURES=organization,webhook\n", read(t, root, "web/.env.example"))
	assert.Contains(t, read(t, root, "web/package.json"), `"name": "acme-web"`)
	assert.Contains(t, read(t, root, "admin/package.json"), `"name": "acme-admin"`)
	assert.Contains(t, read(t, root, "contracts/package.json"), `"name": "@acme/contracts"`)
	assert.Equal(t, "name: acme-dev\n", read(t, root, "compose.dev.yaml"))
	assert.Equal(t, "ARG OCI_SOURCE=https://github.com/example-demo/platform\n", read(t, root, "web/Dockerfile"))
	dev := read(t, root, "scripts/dev.sh")
	assert.Contains(t, dev, "${FASTBUILD_DEV_STARTERS:-organization,webhook,operator}")
	assert.Contains(t, dev, "${FASTBUILD_DEV_WEB_FEATURES:-organization,webhook}")
	assert.Contains(t, dev, "${FASTBUILD_DEV_ADMIN_FEATURES:-operator,organization,webhook}")

	assert.Equal(t, scaffoldModulePath, read(t, root, "docs/untracked-is-ignored.md"), "untracked files are left alone")
	assert.Contains(t, read(t, root, "api/assets/binary.bin"), scaffoldModulePath, "binary files are left alone")
	assert.Equal(t, "FastBuild scaffold docs stay as written.\n", read(t, root, "README.md"))

	var recorded ProjectMarker
	require.NoError(t, json.Unmarshal([]byte(read(t, root, projectMarkerFile)), &recorded))
	assert.Equal(t, *marker, recorded)
	assert.Equal(t, "https://github.com/example-demo/platform", recorded.Repository)
	assert.Equal(t, "v0.21.1", recorded.ScaffoldVersion)

	_, _, err = InitProject(root, ProjectOptions{Name: "Again", Slug: "again", Module: "github.com/example-again/x/api"}, "v0.21.1", now)
	require.ErrorContains(t, err, "already initialized")
	_, err = os.Stat(filepath.Join(root, registryDir))
	assert.True(t, os.IsNotExist(err), "new projects fetch plugins from their release instead of carrying the registry")
}

func TestInitProjectKeepsStarterDefaultsWhenNoneAreGiven(t *testing.T) {
	root := projectFixture(t)
	_, _, err := InitProject(root, ProjectOptions{
		Name: "Acme", Slug: "acme", Module: "gitlab.example.com/acme/api", Repository: "https://gitlab.example.com/acme",
	}, "v0.21.1", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "APP_NAME=Acme\nOPTIONAL_STARTERS=\n", read(t, root, "api/.env.example"))
	assert.Contains(t, read(t, root, "scripts/dev.sh"), "${FASTBUILD_DEV_STARTERS:-a,b}")
}

func TestInitProjectRejectsInvalidInputWithoutWriting(t *testing.T) {
	root := projectFixture(t)
	cases := []ProjectOptions{
		{Name: "", Slug: "acme", Module: "github.com/example-demo/x/api"},
		{Name: "Acme", Slug: "Acme", Module: "github.com/example-demo/x/api"},
		{Name: "Acme", Slug: "acme", Module: scaffoldModulePath},
		{Name: "Acme", Slug: "acme", Module: "gitlab.example.com/acme/api"},
		{Name: "Acme", Slug: "acme", Module: "github.com/example-demo/x/api", Starters: []string{"webhook"}},
		{Name: "Acme", Slug: "acme", Module: "github.com/example-demo/x/api", Starters: []string{"billing"}},
	}
	for _, options := range cases {
		_, changed, err := InitProject(root, options, "v0.21.1", time.Now())
		assert.Error(t, err, "%+v", options)
		assert.Zero(t, changed)
	}
	assert.Contains(t, read(t, root, "api/go.mod"), scaffoldModulePath)
	_, err := os.Stat(filepath.Join(root, projectMarkerFile))
	assert.True(t, os.IsNotExist(err))
}

func TestInitProjectInstallsATemplateFromTheReleaseItStartsFrom(t *testing.T) {
	root := projectFixture(t)
	write := func(path, content string) {
		full := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	write("registry/plugins/block-hero/plugin.json", `{"name":"block-hero","kind":"block","version":"0.1.0",`+
		`"summary":"Hero","description":"Hero block.","use_when":["landing page"],"fastbuild":"0.25.0","requires":{}}`)
	write("registry/plugins/block-hero/files/web/src/components/blocks/hero/index.tsx", "export {};\n")
	write("registry/plugins/starter-site/plugin.json", `{"name":"starter-site","kind":"template","version":"0.1.0",`+
		`"summary":"Site","description":"A site template.","use_when":["a marketing site"],"fastbuild":"0.25.0",`+
		`"requires":{"starters":["organization"]},"includes":["block-hero"]}`)
	index, err := pluginregistry.BuildIndex(filepath.Join(root, registryDir))
	require.NoError(t, err)
	raw, err := index.Marshal()
	require.NoError(t, err)
	write("registry/index.json", string(raw))

	marker, _, err := InitProject(root, ProjectOptions{
		Name: "Acme Site", Slug: "acme", Module: "github.com/example-demo/site/api", Template: "starter-site",
	}, "v0.25.0", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "starter-site", marker.Template)
	assert.Equal(t, []string{"organization"}, marker.Starters, "the template's starters are selected")
	assert.Contains(t, read(t, root, "api/.env.example"), "OPTIONAL_STARTERS=organization\n")
	assert.FileExists(t, filepath.Join(root, "web/src/components/blocks/hero/index.tsx"))
	lock, err := pluginregistry.ReadLock(root)
	require.NoError(t, err)
	assert.Equal(t, "github:"+scaffoldGitHubRepository+"@v0.25.0", lock.Plugins["block-hero"].Source)
	assert.NoDirExists(t, filepath.Join(root, registryDir))

	_, _, err = InitProject(projectFixture(t), ProjectOptions{Name: "X", Slug: "xyz", Module: "github.com/example-demo/x/api", Template: "missing"}, "v0.25.0", time.Now())
	assert.ErrorContains(t, err, "is not a template")
}

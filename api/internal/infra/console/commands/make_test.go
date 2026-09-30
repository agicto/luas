package commands

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeModuleCommandCreatesDDDScaffold(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	defer func() {
		_ = os.Chdir(wd)
	}()

	cmd := NewMakeModuleCommand()
	require.NoError(t, cmd.Run([]string{"BlogPost"}))

	domainPath := filepath.Join(tmp, "internal", "domain", "blog_post.go")
	moduleDir := filepath.Join(tmp, "internal", "modules", "blog_post")

	requiredFiles := []string{
		domainPath,
		filepath.Join(moduleDir, "model.go"),
		filepath.Join(moduleDir, "service.go"),
		filepath.Join(moduleDir, "handler.go"),
		filepath.Join(moduleDir, "repository.go"),
		filepath.Join(moduleDir, "dto.go"),
		filepath.Join(moduleDir, "routes.go"),
		filepath.Join(moduleDir, "service_test.go"),
		filepath.Join(moduleDir, "provider.go"),
		filepath.Join(moduleDir, "error_mappings.go"),
	}

	for _, path := range requiredFiles {
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, path)
	}

	handlerContent, err := os.ReadFile(filepath.Join(moduleDir, "handler.go"))
	require.NoError(t, err)
	assert.Contains(t, string(handlerContent), "assembly.RouteModule")
	assert.Contains(t, string(handlerContent), "Failed to list blog_posts")

	routesContent, err := os.ReadFile(filepath.Join(moduleDir, "routes.go"))
	require.NoError(t, err)
	assert.Contains(t, string(routesContent), "func (h *Handler) RegisterRoutes")
	assert.Contains(t, string(routesContent), "/blog_posts")
}

func TestMakeServiceCommandUsesExistingModuleScaffold(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	defer func() {
		_ = os.Chdir(wd)
	}()

	require.NoError(t, os.MkdirAll(filepath.Join("internal", "modules", "order_item"), 0755))

	cmd := NewMakeServiceCommand()
	require.NoError(t, cmd.Run([]string{"OrderItem"}))

	servicePath := filepath.Join(tmp, "internal", "modules", "order_item", "service.go")
	domainPath := filepath.Join(tmp, "internal", "domain", "order_item.go")

	_, err = os.Stat(servicePath)
	require.NoError(t, err)
	_, err = os.Stat(domainPath)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmp, "app", "order_item", "service.go"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	serviceContent, err := os.ReadFile(servicePath)
	require.NoError(t, err)
	assert.Contains(t, string(serviceContent), "domain.OrderItemRepository")
	assert.Contains(t, string(serviceContent), "CreateOrderItemRequest")
}

func TestMakeServiceCommandRequiresExistingModule(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	defer func() {
		_ = os.Chdir(wd)
	}()

	cmd := NewMakeServiceCommand()
	err = cmd.Run([]string{"Invoice"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "make:module Invoice"))
}

// TestMakeModuleGeneratesAWiredStarterThatBuilds generates a starter against copies of the wiring
// files, then compiles and tests it inside the real module through a Go build overlay, so the
// generator's output is proven to build, pass its own tests, and satisfy the starter catalog.
func TestMakeModuleGeneratesAWiredStarterThatBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles generated code against the whole API module")
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	apiRoot, err := filepath.Abs(filepath.Join(wd, "..", "..", "..", ".."))
	require.NoError(t, err)

	tmp := t.TempDir()
	wiringFiles := []string{
		filepath.Join("internal", "infra", "config", "starters.go"),
		filepath.Join("internal", "starter", "defaults.go"),
	}
	for _, rel := range wiringFiles {
		raw, readErr := os.ReadFile(filepath.Join(apiRoot, rel))
		require.NoError(t, readErr)
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(tmp, rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, rel), raw, 0o644))
	}

	require.NoError(t, os.Chdir(tmp))
	runErr := NewMakeModuleCommand().Run([]string{"BlogPost"})
	require.NoError(t, os.Chdir(wd))
	require.NoError(t, runErr)

	starters, err := os.ReadFile(filepath.Join(tmp, wiringFiles[0]))
	require.NoError(t, err)
	assert.Contains(t, string(starters), `StarterBlogPost     = "blogpost"`)
	defaults, err := os.ReadFile(filepath.Join(tmp, wiringFiles[1]))
	require.NoError(t, err)
	assert.Contains(t, string(defaults), "blogpost.NewStarterManifest(h.BlogPost)")

	replace := map[string]string{}
	require.NoError(t, filepath.WalkDir(tmp, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, relErr := filepath.Rel(tmp, path)
		if relErr != nil {
			return relErr
		}
		replace[filepath.Join(apiRoot, rel)] = path
		return nil
	}))
	overlay, err := json.Marshal(map[string]any{"Replace": replace})
	require.NoError(t, err)
	overlayPath := filepath.Join(t.TempDir(), "overlay.json")
	require.NoError(t, os.WriteFile(overlayPath, overlay, 0o644))

	for _, args := range [][]string{
		// go vet cannot enter a package directory that exists only in the overlay, so compile with
		// go build and run tests with vet disabled; golangci-lint covers generated code in CI.
		{"build", "-overlay=" + overlayPath, "./internal/modules/blog_post", "./internal/starter", "./internal/infra/config", "./database/migrations", "./internal/wiring"},
		{"test", "-count=1", "-vet=off", "-overlay=" + overlayPath, "./internal/starter"},
		{"test", "-c", "-vet=off", "-overlay=" + overlayPath, "-o", filepath.Join(tmp, "blogpost.test"), "./internal/modules/blog_post"},
	} {
		command := exec.Command("go", args...)
		command.Dir = apiRoot
		output, cmdErr := command.CombinedOutput()
		require.NoError(t, cmdErr, "go %s\n%s", strings.Join(args[:1], " "), output)
	}

	// The generated package exists only in the overlay, so run its compiled tests from the copy.
	generatedTests := exec.Command(filepath.Join(tmp, "blogpost.test"), "-test.count=1")
	generatedTests.Dir = filepath.Join(tmp, "internal", "modules", "blog_post")
	output, err := generatedTests.CombinedOutput()
	require.NoError(t, err, "generated starter tests\n%s", output)
}

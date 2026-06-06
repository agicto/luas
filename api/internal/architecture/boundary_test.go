package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const modulePath = "github.com/zgiai/luas/api"

func TestPackageBoundariesDoNotDrift(t *testing.T) {
	root := findModuleRoot(t)
	files := goFiles(t, root)

	for _, file := range files {
		relPath, err := filepath.Rel(root, file)
		require.NoError(t, err)
		rel := filepath.ToSlash(relPath)
		imports := importsForFile(t, file)

		for _, imported := range imports {
			switch {
			case strings.HasPrefix(rel, "pkg/"):
				assertPkgImportAllowed(t, rel, imported)
			case strings.HasPrefix(rel, "internal/capabilities/"):
				assertCapabilityImportAllowed(t, rel, imported)
			case strings.HasPrefix(rel, "internal/modules/"):
				assertModuleImportAllowed(t, rel, imported)
			}
		}
	}
}

func assertPkgImportAllowed(t *testing.T, rel string, imported string) {
	t.Helper()
	require.Falsef(t, strings.HasPrefix(imported, modulePath+"/internal/"), "%s imports internal package %s", rel, imported)
}

func assertCapabilityImportAllowed(t *testing.T, rel string, imported string) {
	t.Helper()
	if !strings.HasPrefix(imported, modulePath+"/internal/infra/") && !strings.HasPrefix(imported, modulePath+"/internal/modules/") {
		return
	}

	internalPath := strings.TrimPrefix(imported, modulePath+"/")
	require.Failf(t, "capability boundary violation", "%s imports %s", rel, internalPath)
}

func assertModuleImportAllowed(t *testing.T, rel string, imported string) {
	t.Helper()
	const modulesPrefix = modulePath + "/internal/modules/"
	if !strings.HasPrefix(imported, modulesPrefix) {
		return
	}

	sourceModule := strings.Split(strings.TrimPrefix(rel, "internal/modules/"), "/")[0]
	targetModule := strings.Split(strings.TrimPrefix(imported, modulesPrefix), "/")[0]
	require.Equalf(t, sourceModule, targetModule, "%s imports sibling starter module %s", rel, imported)
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not find go.mod from %s", dir)
		dir = parent
	}
}

func goFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	return files
}

func importsForFile(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	require.NoError(t, err)

	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		imports = append(imports, strings.Trim(spec.Path.Value, `"`))
	}
	return imports
}

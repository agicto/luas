package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func writeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho 1.0.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverSearchesPathOnlyNotTheWorkingDirectory(t *testing.T) {
	workdir := t.TempDir()
	writeExecutable(t, workdir, "luas-planted")
	pathDir := t.TempDir()
	writeExecutable(t, pathDir, "luas-installed")

	t.Chdir(workdir)
	t.Setenv("PATH", pathDir)

	plugins := Discover()
	if len(plugins) != 1 || plugins[0].Name != "installed" {
		t.Fatalf("Discover() = %+v, want only the plugin on PATH", plugins)
	}
}

func TestPluginNamesAreValidated(t *testing.T) {
	for _, name := range []string{"ai", "db-tools", "x1"} {
		if !ValidName(name) {
			t.Errorf("ValidName(%q) = false", name)
		}
	}
	for _, name := range []string{"", "../evil", "a/b", "Upper", "-flag", "name with space", "ai;rm"} {
		if ValidName(name) {
			t.Errorf("ValidName(%q) = true", name)
		}
		if IsInstalled(name) {
			t.Errorf("IsInstalled(%q) = true", name)
		}
		if err := Execute(name, nil); err == nil {
			t.Errorf("Execute(%q) succeeded", name)
		}
	}
}

func TestDiscoverSkipsInvalidNames(t *testing.T) {
	pathDir := t.TempDir()
	writeExecutable(t, pathDir, "luas-Bad_Name")
	writeExecutable(t, pathDir, "luas-good")
	t.Setenv("PATH", pathDir)

	plugins := Discover()
	if len(plugins) != 1 || plugins[0].Name != "good" {
		t.Fatalf("Discover() = %+v", plugins)
	}
}

package projectfiles

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ConflictMarker starts the local side of an unresolved merge written by Merge3.
const ConflictMarker = "<<<<<<< local"

// Merge3 merges local edits and an incoming version against their common base with git
// merge-file, and reports whether conflict markers remain.
func Merge3(local, base, incoming []byte) ([]byte, bool, error) {
	dir, err := os.MkdirTemp("", "fastbuild-merge-*")
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	paths := map[string][]byte{"local": local, "base": base, "incoming": incoming}
	for name, content := range paths {
		if writeErr := os.WriteFile(filepath.Join(dir, name), content, 0o600); writeErr != nil {
			return nil, false, writeErr
		}
	}
	command := exec.Command("git", "merge-file", "-p", "-L", "local", "-L", "installed", "-L", "update",
		filepath.Join(dir, "local"), filepath.Join(dir, "base"), filepath.Join(dir, "incoming"))
	merged, err := command.Output()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return merged, false, nil
	case errors.As(err, &exitErr) && exitErr.ExitCode() > 0 && exitErr.ExitCode() < 128:
		return merged, true, nil // the exit code is the number of conflicts
	default:
		return nil, false, fmt.Errorf("git merge-file: %w", err)
	}
}

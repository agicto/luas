// Package projectfiles changes files in a project checkout safely: changes are staged, written in
// one step, and undone completely on failure. Plugin installs and scaffold upgrades share it.
package projectfiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ChangeSet stages file changes in memory, writes them in one step, and can undo everything it
// wrote. A dry run stages and never commits; a failure at any step leaves the project exactly as
// it was.
type ChangeSet struct {
	root string
	// staged holds pending content; nil content means the file is to be deleted.
	staged map[string][]byte
	// original holds each written file's content before the change; nil means it did not exist.
	original map[string][]byte
	order    []string
}

// NewChangeSet returns an empty change set for the checkout at root.
func NewChangeSet(root string) *ChangeSet {
	return &ChangeSet{root: root, staged: map[string][]byte{}, original: map[string][]byte{}}
}

func (c *ChangeSet) path(file string) string {
	return filepath.Join(c.root, filepath.FromSlash(file))
}

// Read returns a file's content including staged changes, or nil when it does not exist.
func (c *ChangeSet) Read(file string) ([]byte, error) {
	if content, ok := c.staged[file]; ok {
		return content, nil
	}
	content, err := os.ReadFile(c.path(file))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return content, err
}

// Write stages new content for a file.
func (c *ChangeSet) Write(file string, content []byte) {
	if content == nil {
		content = []byte{}
	}
	c.staged[file] = content
}

// Remove stages a file's deletion.
func (c *ChangeSet) Remove(file string) {
	c.staged[file] = nil
}

// Commit writes every staged change to disk. On failure it restores what it already wrote.
func (c *ChangeSet) Commit() error {
	files := make([]string, 0, len(c.staged))
	for file := range c.staged {
		files = append(files, file)
	}
	sort.Strings(files)
	for _, file := range files {
		if err := c.remember(file); err != nil {
			return c.Fail(err)
		}
		content := c.staged[file]
		target := c.path(file)
		if content == nil {
			if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return c.Fail(err)
			}
			removeEmptyParents(c.root, filepath.Dir(target))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return c.Fail(err)
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return c.Fail(err)
		}
	}
	c.staged = map[string][]byte{}
	return nil
}

// Track records a file that a tool outside the change set is about to edit, such as the contract
// tooling, so rollback restores it too.
func (c *ChangeSet) Track(file string) error {
	return c.remember(file)
}

func (c *ChangeSet) remember(file string) error {
	if _, seen := c.original[file]; seen {
		return nil
	}
	content, err := os.ReadFile(c.path(file))
	if errors.Is(err, fs.ErrNotExist) {
		content, err = nil, nil
	}
	if err != nil {
		return err
	}
	c.original[file] = content
	c.order = append(c.order, file)
	return nil
}

// Rollback discards staged changes and restores every file written so far, newest first.
func (c *ChangeSet) Rollback() error {
	c.staged = map[string][]byte{}
	var errs []error
	for index := len(c.order) - 1; index >= 0; index-- {
		file := c.order[index]
		target := c.path(file)
		if content := c.original[file]; content != nil {
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				errs = append(errs, err)
			} else if err := os.WriteFile(target, content, 0o644); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
		removeEmptyParents(c.root, filepath.Dir(target))
	}
	c.order, c.original = nil, map[string][]byte{}
	if len(errs) > 0 {
		return fmt.Errorf("rollback incomplete: %w", errors.Join(errs...))
	}
	return nil
}

// Fail undoes the change set and returns the cause together with any rollback problem.
func (c *ChangeSet) Fail(cause error) error {
	return errors.Join(cause, c.Rollback())
}

// removeEmptyParents deletes directories a plugin emptied, stopping at the project root.
func removeEmptyParents(root, dir string) {
	root = filepath.Clean(root)
	for dir = filepath.Clean(dir); dir != root && len(dir) > len(root); dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil {
			return
		}
	}
}

// Staged lists the files with pending changes.
func (c *ChangeSet) Staged() []string {
	files := make([]string, 0, len(c.staged))
	for file := range c.staged {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

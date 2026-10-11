package pluginregistry

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// changeSet stages file changes in memory, writes them in one step, and can undo everything it
// wrote. Every lifecycle command runs through one: a dry run stages and never commits, and a
// failure at any step leaves the project exactly as it was.
type changeSet struct {
	root string
	// staged holds pending content; nil content means the file is to be deleted.
	staged map[string][]byte
	// original holds each written file's content before the change; nil means it did not exist.
	original map[string][]byte
	order    []string
}

func newChangeSet(root string) *changeSet {
	return &changeSet{root: root, staged: map[string][]byte{}, original: map[string][]byte{}}
}

func (c *changeSet) path(file string) string {
	return filepath.Join(c.root, filepath.FromSlash(file))
}

// read returns a file's content including staged changes, or nil when it does not exist.
func (c *changeSet) read(file string) ([]byte, error) {
	if content, ok := c.staged[file]; ok {
		return content, nil
	}
	content, err := os.ReadFile(c.path(file))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return content, err
}

func (c *changeSet) write(file string, content []byte) {
	if content == nil {
		content = []byte{}
	}
	c.staged[file] = content
}

func (c *changeSet) remove(file string) {
	c.staged[file] = nil
}

// applyInsert stages lines before the slot marker and reports whether it added them.
func (c *changeSet) applyInsert(slot Slot, lines []string) (bool, error) {
	content, err := c.read(slot.File)
	if err != nil {
		return false, err
	}
	if content == nil {
		return false, fmt.Errorf("%w: %s needs %s", errSlotMissing, slot.Name, slot.File)
	}
	updated, added, err := applyInsert(slot.File, content, slot.Name, lines)
	if err != nil || !added {
		return false, err
	}
	c.write(slot.File, updated)
	return true, nil
}

// removeInsert stages the reversal of an applied insert and reports whether the lines were still
// there.
func (c *changeSet) removeInsert(applied AppliedInsert) (bool, error) {
	content, err := c.read(applied.File)
	if err != nil || content == nil {
		return false, err
	}
	updated, removed, err := removeInsert(applied.File, content, applied.Lines)
	if err != nil || !removed {
		return false, err
	}
	c.write(applied.File, updated)
	return true, nil
}

// commit writes every staged change to disk. On failure it restores what it already wrote.
func (c *changeSet) commit() error {
	files := make([]string, 0, len(c.staged))
	for file := range c.staged {
		files = append(files, file)
	}
	sort.Strings(files)
	for _, file := range files {
		if err := c.remember(file); err != nil {
			return c.fail(err)
		}
		content := c.staged[file]
		target := c.path(file)
		if content == nil {
			if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return c.fail(err)
			}
			removeEmptyParents(c.root, filepath.Dir(target))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return c.fail(err)
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return c.fail(err)
		}
	}
	c.staged = map[string][]byte{}
	return nil
}

// track records a file that a tool outside the change set is about to edit, such as the contract
// tooling, so rollback restores it too.
func (c *changeSet) track(file string) error {
	return c.remember(file)
}

func (c *changeSet) remember(file string) error {
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

// rollback discards staged changes and restores every file written so far, newest first.
func (c *changeSet) rollback() error {
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

// fail undoes the change set and returns the cause together with any rollback problem.
func (c *changeSet) fail(cause error) error {
	return errors.Join(cause, c.rollback())
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

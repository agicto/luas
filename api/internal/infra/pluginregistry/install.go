package pluginregistry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// LockFile records installed plugins at the repository root.
const LockFile = "fastbuild.plugins.lock"

// Lock is the committed record of installed plugins.
type Lock struct {
	Version int                    `json:"version"`
	Plugins map[string]LockedEntry `json:"plugins"`
}

// LockedEntry records one installed plugin and the checksum of each file at install time, so
// later changes are visible and a future update can merge instead of overwrite.
type LockedEntry struct {
	Version string `json:"version"`
	Source  string `json:"source"`
	Starter string `json:"starter,omitempty"`
	// Fragment is the contract fragment merged into contracts/openapi.yaml at install time.
	Fragment string            `json:"fragment,omitempty"`
	Files    map[string]string `json:"files"`
}

// ReadLock reads the lock file, returning an empty lock when none exists.
func ReadLock(root string) (*Lock, error) {
	raw, err := os.ReadFile(filepath.Join(root, LockFile))
	if errors.Is(err, fs.ErrNotExist) {
		return &Lock{Version: 1, Plugins: map[string]LockedEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var lock Lock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, fmt.Errorf("%s: %w", LockFile, err)
	}
	if lock.Version != 1 {
		return nil, fmt.Errorf("%s version %d is not supported", LockFile, lock.Version)
	}
	if lock.Plugins == nil {
		lock.Plugins = map[string]LockedEntry{}
	}
	return &lock, nil
}

// Write saves the lock with stable formatting.
func (l *Lock) Write(root string) error {
	raw, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, LockFile), append(raw, '\n'), 0o644)
}

// Installer adds one plugin to a project checkout. The hooks keep this package independent of how
// starters are wired and how contracts are merged.
type Installer struct {
	Root   string
	Source Source
	// ScaffoldVersion is the project's FastBuild version, or "" when unknown (the scaffold itself).
	ScaffoldVersion string
	// KnownStarters lists the starters the project can select.
	KnownStarters []string
	// WireStarter edits the project's starter registration and returns the files it changed.
	WireStarter func(spec StarterSpec) ([]string, error)
	// MergeFragment merges a contract fragment into contracts/openapi.yaml and removes it. When nil,
	// the fragment stays in contracts/fragments for an integration step.
	MergeFragment func(fragment string) error
	// EditedFiles lists repository paths the hooks may change, so a failure can restore them.
	EditedFiles []string
	// Transform adapts a verified plugin file to the project before it is written, for example by
	// rewriting the scaffold's Go module path to the project's.
	Transform func(file string, content []byte) []byte
	// DryRun stops after verification and conflict checks and reports what would be written.
	DryRun bool
}

// Result reports what an install changed.
type Result struct {
	Entry          *IndexEntry
	Written        []string
	Edited         []string
	FragmentMerged bool
}

// Install verifies, writes, and wires a plugin, or changes nothing and returns an error.
func (installer *Installer) Install(name string) (*Result, error) {
	lock, err := ReadLock(installer.Root)
	if err != nil {
		return nil, err
	}
	if _, installed := lock.Plugins[name]; installed {
		return nil, fmt.Errorf("plugin %q is already installed; see %s", name, LockFile)
	}
	index, err := installer.Source.Index()
	if err != nil {
		return nil, err
	}
	entry, ok := index.Find(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q is not in %s; run plugin:search", name, installer.Source.Describe())
	}
	if requirementErr := installer.checkRequirements(entry, lock); requirementErr != nil {
		return nil, requirementErr
	}

	contents := map[string][]byte{}
	for file := range entry.Files {
		content, readErr := installer.Source.ReadFile(entry, file)
		if readErr != nil {
			return nil, readErr
		}
		if installer.Transform != nil {
			content = installer.Transform(file, content)
		}
		contents[file] = content
	}
	var conflicts, toWrite []string
	for _, file := range sortedFiles(contents) {
		existing, readErr := os.ReadFile(filepath.Join(installer.Root, filepath.FromSlash(file)))
		switch {
		case errors.Is(readErr, fs.ErrNotExist):
			toWrite = append(toWrite, file)
		case readErr != nil:
			return nil, readErr
		case !bytes.Equal(existing, contents[file]):
			conflicts = append(conflicts, file)
		}
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("plugin %q would overwrite existing files; nothing was changed:\n  %s",
			name, strings.Join(conflicts, "\n  "))
	}

	if installer.DryRun {
		return &Result{Entry: entry, Written: toWrite}, nil
	}
	snapshot, err := installer.snapshot()
	if err != nil {
		return nil, err
	}
	result := &Result{Entry: entry}
	fail := func(cause error) (*Result, error) {
		return nil, errors.Join(cause, installer.rollback(result.Written, snapshot))
	}
	for _, file := range toWrite {
		target := filepath.Join(installer.Root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(target, contents[file], 0o644); err != nil {
			return fail(err)
		}
		result.Written = append(result.Written, file)
	}
	if entry.Starter != nil && installer.WireStarter != nil {
		edited, wireErr := installer.WireStarter(*entry.Starter)
		if wireErr != nil {
			return fail(fmt.Errorf("wire starter %q: %w", entry.Starter.Name, wireErr))
		}
		result.Edited = append(result.Edited, edited...)
	}
	if entry.Fragment != "" && installer.MergeFragment != nil {
		if mergeErr := installer.MergeFragment(entry.Fragment); mergeErr != nil {
			return fail(fmt.Errorf("merge contract fragment: %w", mergeErr))
		}
		result.FragmentMerged = true
		result.Edited = append(result.Edited, "contracts/openapi.yaml")
	}

	locked := LockedEntry{Version: entry.Version, Source: installer.Source.Describe(), Files: map[string]string{}}
	if entry.Starter != nil {
		locked.Starter = entry.Starter.Name
	}
	for file, content := range contents {
		if result.FragmentMerged && file == entry.Fragment {
			locked.Fragment = file
			continue // merged into openapi.yaml and removed
		}
		locked.Files[file] = Checksum(content)
	}
	lock.Plugins[name] = locked
	if err := lock.Write(installer.Root); err != nil {
		return fail(err)
	}
	return result, nil
}

func (installer *Installer) checkRequirements(entry *IndexEntry, lock *Lock) error {
	var problems []string
	if installer.ScaffoldVersion != "" {
		current, err := ParseVersion(installer.ScaffoldVersion)
		minimum, minErr := ParseVersion(entry.FastBuild)
		if err == nil && minErr == nil && !current.AtLeast(minimum) {
			problems = append(problems, fmt.Sprintf("needs FastBuild %s or newer; this project is on %s", minimum, current))
		}
	}
	for _, starter := range entry.Requires.Starters {
		if !slices.Contains(installer.KnownStarters, starter) {
			problems = append(problems, fmt.Sprintf("needs the %q starter, which this project does not have", starter))
		}
	}
	if entry.Starter != nil && slices.Contains(installer.KnownStarters, entry.Starter.Name) {
		problems = append(problems, fmt.Sprintf("the project already has a %q starter", entry.Starter.Name))
	}
	for _, plugin := range entry.Requires.Plugins {
		if _, ok := lock.Plugins[plugin]; !ok {
			problems = append(problems, fmt.Sprintf("needs the %q plugin; install it first", plugin))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("plugin %q cannot be installed: %s", entry.Name, strings.Join(problems, "; "))
	}
	return nil
}

func (installer *Installer) snapshot() (map[string][]byte, error) {
	saved := map[string][]byte{}
	for _, file := range installer.EditedFiles {
		content, err := os.ReadFile(filepath.Join(installer.Root, filepath.FromSlash(file)))
		if errors.Is(err, fs.ErrNotExist) {
			saved[file] = nil
			continue
		}
		if err != nil {
			return nil, err
		}
		saved[file] = content
	}
	return saved, nil
}

func (installer *Installer) rollback(written []string, snapshot map[string][]byte) error {
	var errs []error
	for _, file := range written {
		if err := os.Remove(filepath.Join(installer.Root, filepath.FromSlash(file))); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	for file, content := range snapshot {
		target := filepath.Join(installer.Root, filepath.FromSlash(file))
		if content == nil {
			if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("rollback incomplete: %w", errors.Join(errs...))
	}
	return nil
}

// FileStatus describes one installed file now.
type FileStatus struct {
	Path  string `json:"path"`
	State string `json:"state"` // unchanged, modified, or missing
}

// Status compares installed files with the checksums recorded at install time.
func Status(root string, entry LockedEntry) []FileStatus {
	statuses := make([]FileStatus, 0, len(entry.Files))
	for _, file := range sortedFiles(entry.Files) {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		state := "unchanged"
		switch {
		case err != nil:
			state = "missing"
		case Checksum(content) != entry.Files[file]:
			state = "modified"
		}
		statuses = append(statuses, FileStatus{Path: file, State: state})
	}
	return statuses
}

func sortedFiles[V any](files map[string]V) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

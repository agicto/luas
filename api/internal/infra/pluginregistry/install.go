package pluginregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/fastagi/fastbuild/api/internal/infra/projectfiles"
)

// LockFile records installed plugins at the repository root.
const LockFile = "fastbuild.plugins.lock"

// ContractFile is the project's OpenAPI contract, which feature plugins extend.
const ContractFile = "contracts/openapi.yaml"

// Lock is the committed record of installed plugins.
type Lock struct {
	Version int                    `json:"version"`
	Plugins map[string]LockedEntry `json:"plugins"`
}

// LockedEntry records one installed plugin: the checksum of each file at install time, every
// slot edit, and every contract addition, so status, update, and remove know exactly what the
// plugin owns.
type LockedEntry struct {
	Version  string   `json:"version"`
	Kind     string   `json:"kind,omitempty"`
	Source   string   `json:"source"`
	Starter  string   `json:"starter,omitempty"`
	Requires []string `json:"requires,omitempty"`
	// Fragment is the contract fragment merged into contracts/openapi.yaml, with its checksum so an
	// update knows whether the contract changed.
	Fragment         string             `json:"fragment,omitempty"`
	FragmentChecksum string             `json:"fragment_checksum,omitempty"`
	Contract         *ContractAdditions `json:"contract,omitempty"`
	Wiring           []AppliedInsert    `json:"wiring,omitempty"`
	Files            map[string]string  `json:"files"`
}

// ContractAdditions names everything a merged fragment added to the contract.
type ContractAdditions struct {
	Tags       []string `json:"tags,omitempty"`
	Paths      []string `json:"paths,omitempty"`
	Parameters []string `json:"parameters,omitempty"`
	Schemas    []string `json:"schemas,omitempty"`
	ErrorCodes []string `json:"error_codes,omitempty"`
}

// ContractEditor merges and removes contract fragments. The contract tooling owns the YAML
// layout, so the installer only orchestrates it.
type ContractEditor interface {
	// Merge merges the fragment at the repository path into contracts/openapi.yaml, deletes the
	// fragment, and reports what it added.
	Merge(fragment string) (*ContractAdditions, error)
	// Remove deletes previously merged additions from contracts/openapi.yaml.
	Remove(additions ContractAdditions) error
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
	raw, err := l.encode()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, LockFile), raw, 0o644)
}

func (l *Lock) encode() ([]byte, error) {
	raw, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// save stages the lock, deleting it when no plugin remains.
func (l *Lock) save(changes *projectfiles.ChangeSet) error {
	if len(l.Plugins) == 0 {
		changes.Remove(LockFile)
		return nil
	}
	raw, err := l.encode()
	if err != nil {
		return err
	}
	changes.Write(LockFile, raw)
	return nil
}

// Installer adds, updates, and removes plugins in one project checkout.
type Installer struct {
	Root   string
	Source Source
	// ScaffoldVersion is the project's FastBuild version, or "" when unknown (the scaffold itself).
	ScaffoldVersion string
	// ScaffoldModule is the scaffold's Go module path, used in generated starter imports before
	// Transform adapts them to the project.
	ScaffoldModule string
	// KnownStarters lists the starters the project can select.
	KnownStarters []string
	// Contract merges and removes contract fragments. When nil, fragments stay in
	// contracts/fragments for an integration step.
	Contract ContractEditor
	// Transform adapts verified plugin content to the project before it is written, for example by
	// rewriting the scaffold's Go module path to the project's.
	Transform func(file string, content []byte) []byte
	// Merge3 merges local edits with an incoming version; nil uses git merge-file.
	Merge3 func(local, base, incoming []byte) (merged []byte, conflict bool, err error)
	// DryRun stops after verification and conflict checks and reports what would change.
	DryRun bool

	// planned names plugins a dry-run template counts as installed for requirement checks.
	planned []string
}

// Result reports what an install changed.
type Result struct {
	Entry          *IndexEntry
	Written        []string
	Wiring         []AppliedInsert
	FragmentMerged bool
}

// Edited lists the core files the install changed through slots and the contract.
func (r *Result) Edited() []string {
	var files []string
	for _, applied := range r.Wiring {
		if !slices.Contains(files, applied.File) {
			files = append(files, applied.File)
		}
	}
	if r.FragmentMerged {
		files = append(files, ContractFile)
	}
	return files
}

// Install verifies, writes, and wires a plugin, or changes nothing and returns an error.
func (installer *Installer) Install(name string) (*Result, error) {
	lock, err := ReadLock(installer.Root)
	if err != nil {
		return nil, err
	}
	if _, installed := lock.Plugins[name]; installed {
		return nil, fmt.Errorf("plugin %q is already installed; see %s or run plugin:update", name, LockFile)
	}
	entry, err := installer.find(name)
	if err != nil {
		return nil, err
	}
	if requirementErr := installer.checkRequirements(entry, lock, true); requirementErr != nil {
		return nil, requirementErr
	}
	contents, err := installer.read(entry)
	if err != nil {
		return nil, err
	}
	var conflicts, toWrite []string
	for _, file := range sortedFiles(contents) {
		existing, readErr := os.ReadFile(filepath.Join(installer.Root, filepath.FromSlash(file)))
		switch {
		case errors.Is(readErr, fs.ErrNotExist):
			toWrite = append(toWrite, file)
		case readErr != nil:
			return nil, readErr
		case string(existing) != string(contents[file]):
			conflicts = append(conflicts, file)
		}
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("plugin %q would overwrite existing files; nothing was changed:\n  %s",
			name, strings.Join(conflicts, "\n  "))
	}
	inserts, err := installer.inserts(entry)
	if err != nil {
		return nil, err
	}

	result := &Result{Entry: entry}
	changes := projectfiles.NewChangeSet(installer.Root)
	for _, file := range toWrite {
		changes.Write(file, contents[file])
		result.Written = append(result.Written, file)
	}
	for _, insert := range inserts {
		applied, err := stageInsert(changes, insert.slot, insert.lines)
		if err != nil {
			return nil, fmt.Errorf("plugin %q: %w", name, err)
		}
		if applied {
			result.Wiring = append(result.Wiring, AppliedInsert{Slot: insert.slot.Name, File: insert.slot.File, Lines: insert.lines})
		}
	}
	if installer.DryRun {
		return result, nil
	}
	if err := changes.Commit(); err != nil {
		return nil, err
	}

	locked := LockedEntry{
		Version: entry.Version, Kind: entry.Kind, Source: installer.Source.Describe(),
		Requires: entry.Requires.Plugins, Wiring: result.Wiring, Files: map[string]string{},
	}
	if entry.Starter != nil {
		locked.Starter = entry.Starter.Name
	}
	if entry.Fragment != "" && installer.Contract != nil {
		additions, mergeErr := installer.mergeFragment(changes, entry.Fragment)
		if mergeErr != nil {
			return nil, changes.Fail(mergeErr)
		}
		result.FragmentMerged = true
		locked.Fragment, locked.FragmentChecksum, locked.Contract = entry.Fragment, entry.Files[entry.Fragment], additions
	}
	for file, content := range contents {
		if result.FragmentMerged && file == entry.Fragment {
			continue // merged into openapi.yaml and removed
		}
		locked.Files[file] = Checksum(content)
	}
	lock.Plugins[name] = locked
	if err := lock.save(changes); err != nil {
		return nil, changes.Fail(err)
	}
	if err := changes.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (installer *Installer) find(name string) (*IndexEntry, error) {
	index, err := installer.Source.Index()
	if err != nil {
		return nil, err
	}
	entry, ok := index.Find(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q is not in %s; run plugin:search", name, installer.Source.Describe())
	}
	return entry, nil
}

// read returns every plugin file, verified against the index and adapted to the project.
func (installer *Installer) read(entry *IndexEntry) (map[string][]byte, error) {
	contents := map[string][]byte{}
	for file := range entry.Files {
		content, err := installer.Source.ReadFile(entry, file)
		if err != nil {
			return nil, err
		}
		contents[file] = installer.transform(file, content)
	}
	return contents, nil
}

func (installer *Installer) transform(file string, content []byte) []byte {
	if installer.Transform == nil {
		return content
	}
	return installer.Transform(file, content)
}

type resolvedInsert struct {
	slot  Slot
	lines []string
}

// inserts resolves the plugin's slot edits against the catalog and adapts their text.
func (installer *Installer) inserts(entry *IndexEntry) ([]resolvedInsert, error) {
	var resolved []resolvedInsert
	for _, insert := range entry.AllInserts(installer.ScaffoldModule) {
		slot, ok := FindSlot(insert.Slot)
		if !ok {
			return nil, fmt.Errorf("plugin %q wires unknown slot %q", entry.Name, insert.Slot)
		}
		adapted := installer.transform(slot.File, []byte(strings.Join(insert.Lines, "\n")))
		resolved = append(resolved, resolvedInsert{slot: slot, lines: strings.Split(string(adapted), "\n")})
	}
	return resolved, nil
}

func (installer *Installer) mergeFragment(changes *projectfiles.ChangeSet, fragment string) (*ContractAdditions, error) {
	if err := changes.Track(ContractFile); err != nil {
		return nil, err
	}
	if err := changes.Track(fragment); err != nil {
		return nil, err
	}
	additions, err := installer.Contract.Merge(fragment)
	if err != nil {
		return nil, fmt.Errorf("merge contract fragment: %w", err)
	}
	return additions, nil
}

func (installer *Installer) checkRequirements(entry *IndexEntry, lock *Lock, fresh bool) error {
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
	if fresh && entry.Starter != nil && slices.Contains(installer.KnownStarters, entry.Starter.Name) {
		problems = append(problems, fmt.Sprintf("the project already has a %q starter", entry.Starter.Name))
	}
	for _, plugin := range entry.Requires.Plugins {
		if _, ok := lock.Plugins[plugin]; !ok && !slices.Contains(installer.planned, plugin) {
			problems = append(problems, fmt.Sprintf("needs the %q plugin; install it first", plugin))
		}
	}
	if fresh && entry.PluginKind() == KindTheme {
		for name, locked := range lock.Plugins {
			if locked.Kind == KindTheme {
				problems = append(problems, fmt.Sprintf("theme %q is installed; run plugin:remove %s first", name, name))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("plugin %q cannot be installed: %s", entry.Name, strings.Join(problems, "; "))
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
		statuses = append(statuses, FileStatus{Path: file, State: fileState(root, file, entry.Files[file])})
	}
	return statuses
}

func fileState(root, file, checksum string) string {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	switch {
	case err != nil:
		return "missing"
	case Checksum(content) != checksum:
		return "modified"
	default:
		return "unchanged"
	}
}

func sortedFiles[V any](files map[string]V) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

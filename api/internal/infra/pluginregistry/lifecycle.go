package pluginregistry

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// IncomingSuffix names the incoming version of a locally edited file when update cannot merge it
// automatically because the version it was based on is unavailable.
const IncomingSuffix = ".fastbuild-new"

// RemoveOptions controls plugin removal.
type RemoveOptions struct {
	// Force deletes files the project changed since installation.
	Force bool
}

// RemoveResult reports what a removal changed and what it left for the builder.
type RemoveResult struct {
	Entry   LockedEntry
	Deleted []string
	Unwired []AppliedInsert
	// Manual lists follow-ups the CLI could not do safely, such as edited wiring lines.
	Manual []string
}

// Remove deletes a plugin's files, slot edits, and contract additions, and drops it from the lock.
// It never touches data: tables created by the plugin's migrations stay in existing databases.
func (installer *Installer) Remove(name string, options RemoveOptions) (*RemoveResult, error) {
	lock, err := ReadLock(installer.Root)
	if err != nil {
		return nil, err
	}
	entry, ok := lock.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q is not installed; see plugin:status", name)
	}
	var dependents []string
	for other, locked := range lock.Plugins {
		if slices.Contains(locked.Requires, name) {
			dependents = append(dependents, other)
		}
	}
	if len(dependents) > 0 {
		sort.Strings(dependents)
		return nil, fmt.Errorf("plugin %q is required by %s; remove those first", name, strings.Join(dependents, ", "))
	}
	var modified []string
	for _, status := range Status(installer.Root, entry) {
		if status.State == "modified" {
			modified = append(modified, status.Path)
		}
	}
	if len(modified) > 0 && !options.Force {
		return nil, fmt.Errorf("plugin %q has local changes; review them, then pass --force to delete them too:\n  %s",
			name, strings.Join(modified, "\n  "))
	}

	result := &RemoveResult{Entry: entry}
	changes := newChangeSet(installer.Root)
	for _, file := range sortedFiles(entry.Files) {
		if content, readErr := changes.read(file); readErr != nil {
			return nil, readErr
		} else if content != nil {
			changes.remove(file)
			result.Deleted = append(result.Deleted, file)
		}
	}
	for index := len(entry.Wiring) - 1; index >= 0; index-- {
		applied := entry.Wiring[index]
		removed, removeErr := changes.removeInsert(applied)
		if removeErr != nil {
			return nil, removeErr
		}
		if removed {
			result.Unwired = append(result.Unwired, applied)
		} else {
			result.Manual = append(result.Manual, fmt.Sprintf("remove from %s (slot %s): %s",
				applied.File, applied.Slot, strings.Join(applied.Lines, " ")))
		}
	}
	delete(lock.Plugins, name)
	if err := lock.save(changes); err != nil {
		return nil, err
	}
	if installer.DryRun {
		return result, nil
	}
	if err := changes.commit(); err != nil {
		return nil, err
	}
	if entry.Contract != nil {
		if installer.Contract == nil {
			result.Manual = append(result.Manual, "remove the plugin's paths and schemas from "+ContractFile)
		} else {
			if err := changes.track(ContractFile); err != nil {
				return nil, changes.fail(err)
			}
			if err := installer.Contract.Remove(*entry.Contract); err != nil {
				return nil, changes.fail(fmt.Errorf("remove contract additions: %w", err))
			}
		}
	}
	return result, nil
}

// UpdateResult reports how each file of an update was handled.
type UpdateResult struct {
	Entry       *IndexEntry
	FromVersion string
	// Written files were added or replaced because the project had not changed them.
	Written []string
	// Merged files combined local edits with the update without conflicts.
	Merged []string
	// Conflicts hold conflict markers that the builder or agent must resolve.
	Conflicts []string
	// Incoming files were saved next to a locally edited file with IncomingSuffix, because the
	// version the edit was based on was unavailable for a merge.
	Incoming []string
	Deleted  []string
	// Kept files were edited or deleted locally and left as they are.
	Kept           []string
	Wired          []AppliedInsert
	Unwired        []AppliedInsert
	ContractMerged bool
	Manual         []string
}

// Update moves an installed plugin to the source's version. Files the project never changed are
// replaced; edited files are merged three ways with the installed version when Base can supply it.
// Nothing is changed when an error is returned.
func (installer *Installer) Update(name string, base Source) (*UpdateResult, error) {
	lock, err := ReadLock(installer.Root)
	if err != nil {
		return nil, err
	}
	locked, ok := lock.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q is not installed; run plugin:add %s", name, name)
	}
	entry, err := installer.find(name)
	if err != nil {
		return nil, err
	}
	installed, curErr := ParseVersion(locked.Version)
	offered, nextErr := ParseVersion(entry.Version)
	if curErr == nil && nextErr == nil && installed.AtLeast(offered) {
		return nil, fmt.Errorf("plugin %q is already at %s (%s offers %s)", name, locked.Version, installer.Source.Describe(), entry.Version)
	}
	if requirementErr := installer.checkRequirements(entry, lock, false); requirementErr != nil {
		return nil, requirementErr
	}
	incoming, err := installer.read(entry)
	if err != nil {
		return nil, err
	}
	baseFiles := installer.baseFiles(base, name, locked.Version)

	result := &UpdateResult{Entry: entry, FromVersion: locked.Version}
	changes := newChangeSet(installer.Root)
	next := LockedEntry{
		Version: entry.Version, Kind: entry.Kind, Source: installer.Source.Describe(),
		Requires: entry.Requires.Plugins, Files: map[string]string{},
	}
	if entry.Starter != nil {
		next.Starter = entry.Starter.Name
	}
	// A merged contract is updated as a whole: unchanged fragments carry over, changed ones replace
	// the previous additions. An unmerged fragment is an ordinary file.
	merged := locked.Contract != nil
	replaceContract := merged && (entry.Fragment == "" || entry.Files[entry.Fragment] != locked.FragmentChecksum)
	if replaceContract && installer.Contract == nil {
		return nil, errors.New("the plugin's contract changed and the contract tooling is unavailable; install Node.js and the contracts dependencies")
	}
	if merged && !replaceContract {
		next.Fragment, next.FragmentChecksum, next.Contract = locked.Fragment, locked.FragmentChecksum, locked.Contract
	}
	for _, file := range unionFiles(locked.Files, incoming) {
		if merged && (file == locked.Fragment || file == entry.Fragment) {
			continue
		}
		if err := installer.updateFile(changes, result, next.Files, file, locked.Files[file], baseFiles[file], incoming[file]); err != nil {
			return nil, err
		}
	}
	if err := installer.updateWiring(changes, result, entry, locked, &next); err != nil {
		return nil, err
	}
	if replaceContract && entry.Fragment != "" {
		changes.write(entry.Fragment, incoming[entry.Fragment])
	}
	if installer.DryRun {
		result.ContractMerged = replaceContract
		return result, nil
	}
	if err := changes.commit(); err != nil {
		return nil, err
	}
	if replaceContract {
		if err := changes.track(ContractFile); err != nil {
			return nil, changes.fail(err)
		}
		if err := installer.Contract.Remove(*locked.Contract); err != nil {
			return nil, changes.fail(fmt.Errorf("remove previous contract additions: %w", err))
		}
		if entry.Fragment != "" {
			additions, mergeErr := installer.mergeFragment(changes, entry.Fragment)
			if mergeErr != nil {
				return nil, changes.fail(mergeErr)
			}
			next.Fragment, next.FragmentChecksum, next.Contract = entry.Fragment, entry.Files[entry.Fragment], additions
		}
		result.ContractMerged = true
	}
	lock.Plugins[name] = next
	if err := lock.save(changes); err != nil {
		return nil, changes.fail(err)
	}
	if err := changes.commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// updateFile decides one file: replace untouched files, merge edited ones, and keep local
// deletions. lockedSum is "" for a file new in this version; incoming is nil for a file the
// version no longer ships.
func (installer *Installer) updateFile(changes *changeSet, result *UpdateResult, nextFiles map[string]string,
	file, lockedSum string, base, incoming []byte,
) error {
	local, err := changes.read(file)
	if err != nil {
		return err
	}
	untouched := local != nil && lockedSum != "" && Checksum(local) == lockedSum
	switch {
	case incoming == nil && (untouched || local == nil):
		if local != nil {
			changes.remove(file)
			result.Deleted = append(result.Deleted, file)
		}
		return nil
	case incoming == nil:
		result.Kept = append(result.Kept, file)
		return nil
	case lockedSum == "" && local != nil && !bytes.Equal(local, incoming):
		return fmt.Errorf("the update adds %s, which already exists in the project; move it aside first", file)
	case lockedSum != "" && local == nil:
		result.Kept = append(result.Kept, file+" (deleted locally)")
		return nil
	}
	nextFiles[file] = Checksum(incoming)
	switch {
	case local == nil || untouched:
		changes.write(file, incoming)
		result.Written = append(result.Written, file)
	case bytes.Equal(local, incoming):
		// The local edit already matches the new version.
	case base == nil:
		changes.write(file+IncomingSuffix, incoming)
		result.Incoming = append(result.Incoming, file)
	default:
		merged, conflict, mergeErr := installer.merge3(local, base, incoming)
		if mergeErr != nil {
			return fmt.Errorf("merge %s: %w", file, mergeErr)
		}
		changes.write(file, merged)
		if conflict {
			result.Conflicts = append(result.Conflicts, file)
		} else {
			result.Merged = append(result.Merged, file)
		}
	}
	return nil
}

// updateWiring removes slot edits the new version dropped and applies the ones it added.
func (installer *Installer) updateWiring(changes *changeSet, result *UpdateResult, entry *IndexEntry, locked LockedEntry, next *LockedEntry) error {
	wanted, err := installer.inserts(entry)
	if err != nil {
		return err
	}
	key := func(slot string, lines []string) string {
		normalized := make([]string, len(lines))
		for index, line := range lines {
			normalized[index] = normalizeLine(line)
		}
		return slot + "\x00" + strings.Join(normalized, "\n")
	}
	wantedKeys := map[string]bool{}
	for _, insert := range wanted {
		wantedKeys[key(insert.slot.Name, insert.lines)] = true
	}
	existing := map[string]bool{}
	for _, applied := range locked.Wiring {
		existing[key(applied.Slot, applied.Lines)] = true
		if wantedKeys[key(applied.Slot, applied.Lines)] {
			next.Wiring = append(next.Wiring, applied)
			continue
		}
		removed, removeErr := changes.removeInsert(applied)
		if removeErr != nil {
			return removeErr
		}
		if removed {
			result.Unwired = append(result.Unwired, applied)
		} else {
			result.Manual = append(result.Manual, fmt.Sprintf("remove from %s (slot %s): %s",
				applied.File, applied.Slot, strings.Join(applied.Lines, " ")))
		}
	}
	for _, insert := range wanted {
		if existing[key(insert.slot.Name, insert.lines)] {
			continue
		}
		applied, applyErr := changes.applyInsert(insert.slot, insert.lines)
		if applyErr != nil {
			return fmt.Errorf("plugin %q: %w", entry.Name, applyErr)
		}
		if applied {
			record := AppliedInsert{Slot: insert.slot.Name, File: insert.slot.File, Lines: insert.lines}
			next.Wiring = append(next.Wiring, record)
			result.Wired = append(result.Wired, record)
		}
	}
	return nil
}

// baseFiles returns the installed version's files from base, adapted like an install, or nil
// when base does not offer exactly that version.
func (installer *Installer) baseFiles(base Source, name, version string) map[string][]byte {
	if base == nil {
		return nil
	}
	index, err := base.Index()
	if err != nil {
		return nil
	}
	entry, ok := index.Find(name)
	if !ok || entry.Version != version {
		return nil
	}
	files := map[string][]byte{}
	for file := range entry.Files {
		content, readErr := base.ReadFile(entry, file)
		if readErr != nil {
			return nil
		}
		files[file] = installer.transform(file, content)
	}
	return files
}

func (installer *Installer) merge3(local, base, incoming []byte) ([]byte, bool, error) {
	if installer.Merge3 != nil {
		return installer.Merge3(local, base, incoming)
	}
	return GitMerge3(local, base, incoming)
}

// GitMerge3 merges with git merge-file and reports whether conflict markers remain.
func GitMerge3(local, base, incoming []byte) ([]byte, bool, error) {
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

func unionFiles(locked map[string]string, incoming map[string][]byte) []string {
	seen := map[string]bool{}
	var files []string
	for file := range locked {
		seen[file] = true
		files = append(files, file)
	}
	for file := range incoming {
		if !seen[file] {
			files = append(files, file)
		}
	}
	sort.Strings(files)
	return files
}

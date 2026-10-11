package projectfiles

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// IncomingSuffix names the incoming version of a file that could not be merged automatically.
const IncomingSuffix = ".fastbuild-new"

// Upgrade moves a project from one release of its scaffold to another. For every file that changed
// between the two releases it keeps the project's work: files the project never changed take the
// new release, edited files are merged three ways, and files the project deleted stay deleted.
type Upgrade struct {
	Root string
	// Base and Target are directories holding the release the project is on and the release to
	// move to, for example extracted archives.
	Base, Target string
	// Adapt turns scaffold content into what the project would have had, such as its module path.
	Adapt func(path string, content []byte) []byte
	// Skip names paths that are never upgraded, such as the scaffold's own registry.
	Skip func(path string) bool
	// Merge3 merges local edits with the target against the base; nil uses Merge3.
	Merge3 func(local, base, target []byte) ([]byte, bool, error)
	// Write extra changes, such as the project marker, in the same commit.
	Extra  map[string][]byte
	DryRun bool
}

// UpgradeResult lists what happened to each changed file.
type UpgradeResult struct {
	Added     []string `json:"added"`
	Replaced  []string `json:"replaced"`
	Merged    []string `json:"merged"`
	Conflicts []string `json:"conflicts"`
	// Incoming files are binary or otherwise unmergeable: the target is saved with IncomingSuffix.
	Incoming []string `json:"incoming"`
	Deleted  []string `json:"deleted"`
	// Kept files were deleted or edited locally where the release deleted or changed them.
	Kept []string `json:"kept"`
}

// Run applies the upgrade, or reports it when DryRun is set. Nothing changes when it fails.
func (u *Upgrade) Run() (*UpgradeResult, error) {
	base, err := listFiles(u.Base)
	if err != nil {
		return nil, err
	}
	target, err := listFiles(u.Target)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for path := range base {
		paths[path] = true
	}
	for path := range target {
		paths[path] = true
	}
	sorted := make([]string, 0, len(paths))
	for path := range paths {
		if u.Skip == nil || !u.Skip(path) {
			sorted = append(sorted, path)
		}
	}
	sort.Strings(sorted)

	result := &UpgradeResult{}
	changes := NewChangeSet(u.Root)
	for _, path := range sorted {
		baseContent, readErr := u.read(u.Base, path, base[path])
		if readErr != nil {
			return nil, readErr
		}
		targetContent, readErr := u.read(u.Target, path, target[path])
		if readErr != nil {
			return nil, readErr
		}
		if bytes.Equal(baseContent, targetContent) && base[path] == target[path] {
			continue // unchanged between the releases
		}
		local, readErr := changes.Read(path)
		if readErr != nil {
			return nil, readErr
		}
		if err := u.decide(changes, result, path, local, baseContent, targetContent, base[path], target[path]); err != nil {
			return nil, err
		}
	}
	for path, content := range u.Extra {
		changes.Write(path, content)
	}
	if u.DryRun {
		return result, nil
	}
	return result, changes.Commit()
}

// decide applies the upgrade rule for one file. inBase and inTarget tell an empty file from a
// missing one.
func (u *Upgrade) decide(changes *ChangeSet, result *UpgradeResult, path string, local, base, target []byte, inBase, inTarget bool) error {
	switch {
	case !inTarget && local == nil:
		return nil
	case !inTarget && bytes.Equal(local, base):
		changes.Remove(path)
		result.Deleted = append(result.Deleted, path)
	case !inTarget:
		result.Kept = append(result.Kept, path)
	case local == nil && inBase:
		result.Kept = append(result.Kept, path) // the project deleted it on purpose
	case local == nil:
		changes.Write(path, target)
		result.Added = append(result.Added, path)
	case bytes.Equal(local, target):
	case inBase && bytes.Equal(local, base):
		changes.Write(path, target)
		result.Replaced = append(result.Replaced, path)
	case !inBase || isBinary(local) || isBinary(target):
		changes.Write(path+IncomingSuffix, target)
		result.Incoming = append(result.Incoming, path)
	default:
		merge := u.Merge3
		if merge == nil {
			merge = Merge3
		}
		merged, conflict, err := merge(local, base, target)
		if err != nil {
			return err
		}
		changes.Write(path, merged)
		if conflict {
			result.Conflicts = append(result.Conflicts, path)
		} else {
			result.Merged = append(result.Merged, path)
		}
	}
	return nil
}

func (u *Upgrade) read(dir, path string, present bool) ([]byte, error) {
	if !present {
		return nil, nil
	}
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		return nil, err
	}
	if u.Adapt != nil {
		content = u.Adapt(path, content)
	}
	return content, nil
}

func listFiles(dir string) (map[string]bool, error) {
	files := map[string]bool{}
	err := filepath.WalkDir(dir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return errors.New(current + " is not a regular file")
		}
		relative, err := filepath.Rel(dir, current)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = true
		return nil
	})
	return files, err
}

func isBinary(content []byte) bool {
	return bytes.IndexByte(content, 0) >= 0
}

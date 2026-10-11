package pluginregistry

import (
	"fmt"
	"os"
	"path/filepath"
)

// IntegrationFile documents the adaptive steps an agent performs after the deterministic install.
const IntegrationFile = "INTEGRATION.md"

// Source is a registry the CLI reads plugins from.
type Source interface {
	// Describe names the source for messages and the lock file, for example
	// "github:fastagi/fastbuild@v0.25.0".
	Describe() string
	Index() (*Index, error)
	// ReadFile returns a plugin file and verifies it against the index checksum.
	ReadFile(entry *IndexEntry, file string) ([]byte, error)
	// ReadDocument returns a plugin document such as INTEGRATION.md; it is not installed.
	ReadDocument(plugin, name string) ([]byte, error)
}

// LocalSource reads a registry directory, such as registry/ in the scaffold repository or a
// release archive extracted by RemoteSource.
type LocalSource struct {
	Dir   string
	Label string
}

// NewLocalSource returns a source for a registry directory.
func NewLocalSource(dir, label string) *LocalSource {
	return &LocalSource{Dir: dir, Label: label}
}

// Describe implements Source.
func (s *LocalSource) Describe() string {
	if s.Label != "" {
		return s.Label
	}
	return "path:" + s.Dir
}

// Index implements Source.
func (s *LocalSource) Index() (*Index, error) {
	raw, err := os.ReadFile(filepath.Join(s.Dir, IndexFile))
	if err != nil {
		return nil, fmt.Errorf("read registry index from %s: %w", s.Describe(), err)
	}
	return ParseIndex(raw)
}

// ReadFile implements Source.
func (s *LocalSource) ReadFile(entry *IndexEntry, file string) ([]byte, error) {
	want, ok := entry.Files[file]
	if !ok {
		return nil, fmt.Errorf("plugin %q does not list %s", entry.Name, file)
	}
	content, err := os.ReadFile(PluginFilePath(s.Dir, entry.Name, file))
	if err != nil {
		return nil, err
	}
	if got := Checksum(content); got != want {
		return nil, fmt.Errorf("plugin %q file %s does not match the registry checksum", entry.Name, file)
	}
	return content, nil
}

// ReadDocument implements Source.
func (s *LocalSource) ReadDocument(plugin, name string) ([]byte, error) {
	if !pluginNamePattern.MatchString(plugin) || filepath.Base(name) != name {
		return nil, fmt.Errorf("invalid plugin document %s/%s", plugin, name)
	}
	return os.ReadFile(filepath.Join(s.Dir, PluginsDir, plugin, name))
}

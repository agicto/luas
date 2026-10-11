package pluginregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IndexFile is the generated catalog at the root of a registry.
const IndexFile = "index.json"

// PluginsDir holds one directory per plugin inside a registry.
const PluginsDir = "plugins"

const (
	maxPluginFiles    = 500
	maxPluginFileSize = 2 << 20
)

// Index is the registry catalog: every plugin's manifest plus the checksum of each file it
// installs. The CLI trusts a file only when its checksum matches the index of the same source.
type Index struct {
	Version int          `json:"version"`
	Plugins []IndexEntry `json:"plugins"`
}

// IndexEntry is one plugin in the index.
type IndexEntry struct {
	Manifest
	// Files maps each repository path the plugin installs to its SHA-256.
	Files map[string]string `json:"files"`
}

// Find returns the entry with the given name.
func (i *Index) Find(name string) (*IndexEntry, bool) {
	for index := range i.Plugins {
		if i.Plugins[index].Name == name {
			return &i.Plugins[index], true
		}
	}
	return nil, false
}

// BuildIndex reads every plugin under registryDir/plugins and returns the catalog, failing on an
// invalid manifest, an unsafe path, or a fragment the plugin does not ship.
func BuildIndex(registryDir string) (*Index, error) {
	entries, err := os.ReadDir(filepath.Join(registryDir, PluginsDir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	index := &Index{Version: 1, Plugins: []IndexEntry{}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(registryDir, PluginsDir, entry.Name())
		manifest, err := ReadManifest(pluginDir)
		if err != nil {
			return nil, err
		}
		if manifest.Name != entry.Name() {
			return nil, fmt.Errorf("plugin directory %q declares name %q", entry.Name(), manifest.Name)
		}
		files := map[string]string{}
		if manifest.PluginKind() == KindTemplate {
			if _, statErr := os.Stat(filepath.Join(pluginDir, FilesDir)); statErr == nil {
				return nil, fmt.Errorf("template %q ships files; templates only include plugins", manifest.Name)
			}
		} else if files, err = hashPluginFiles(filepath.Join(pluginDir, FilesDir)); err != nil {
			return nil, fmt.Errorf("plugin %q: %w", manifest.Name, err)
		}
		if manifest.Fragment != "" {
			if _, ok := files[manifest.Fragment]; !ok {
				return nil, fmt.Errorf("plugin %q declares fragment %s but does not ship it", manifest.Name, manifest.Fragment)
			}
		}
		index.Plugins = append(index.Plugins, IndexEntry{Manifest: *manifest, Files: files})
	}
	sort.Slice(index.Plugins, func(a, b int) bool { return index.Plugins[a].Name < index.Plugins[b].Name })
	for _, entry := range index.Plugins {
		for _, include := range entry.Includes {
			member, ok := index.Find(include)
			if !ok || member.PluginKind() == KindTemplate {
				return nil, fmt.Errorf("template %q includes %q, which is not a plugin in this registry", entry.Name, include)
			}
		}
	}
	return index, nil
}

// ReadManifest reads and validates plugin.json in pluginDir.
func ReadManifest(pluginDir string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(pluginDir, ManifestFile))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(pluginDir, ManifestFile), err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func hashPluginFiles(filesDir string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(filesDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; plugins ship regular files only", current)
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(filesDir, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !ValidFilePath(relative) {
			return fmt.Errorf("%s is not a safe repository path", relative)
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		if len(content) > maxPluginFileSize {
			return fmt.Errorf("%s exceeds %d bytes", relative, maxPluginFileSize)
		}
		files[relative] = Checksum(content)
		if len(files) > maxPluginFiles {
			return fmt.Errorf("more than %d files", maxPluginFiles)
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("plugin has no %s directory", FilesDir)
	}
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("plugin ships no files")
	}
	return files, nil
}

// Checksum returns the hex SHA-256 of content.
func Checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// Marshal renders the index exactly as it is committed.
func (i *Index) Marshal() ([]byte, error) {
	raw, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// ParseIndex decodes a committed index and validates every entry.
func ParseIndex(raw []byte) (*Index, error) {
	var index Index
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("registry index: %w", err)
	}
	if index.Version != 1 {
		return nil, fmt.Errorf("registry index version %d is not supported", index.Version)
	}
	for _, entry := range index.Plugins {
		if err := entry.Validate(); err != nil {
			return nil, err
		}
		for file, sum := range entry.Files {
			if !ValidFilePath(file) || len(sum) != sha256.Size*2 {
				return nil, fmt.Errorf("plugin %q lists an invalid file entry %q", entry.Name, file)
			}
		}
	}
	return &index, nil
}

// SearchResult is one ranked match.
type SearchResult struct {
	Entry IndexEntry
	Score int
}

// Search ranks plugins against a free-text query: name matches weigh most, then keywords and
// use cases, then the summary and description. Queries can be sentences in any language: a
// keyword or name that appears inside the query counts too, which matters for languages written
// without spaces. An empty query returns every plugin.
func (i *Index) Search(query string) []SearchResult {
	query = strings.ToLower(query)
	terms := searchTerms(query)
	var results []SearchResult
	for _, entry := range i.Plugins {
		score := 0
		for _, term := range terms {
			score += termScore(entry, term)
		}
		score += embeddedScore(entry, query)
		if len(terms) == 0 || score > 0 {
			results = append(results, SearchResult{Entry: entry, Score: score})
		}
	}
	sort.SliceStable(results, func(a, b int) bool {
		if results[a].Score != results[b].Score {
			return results[a].Score > results[b].Score
		}
		return results[a].Entry.Name < results[b].Entry.Name
	})
	return results
}

// searchStopWords carry no meaning about which plugin fits.
var searchStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "add": true, "for": true, "in": true, "into": true, "it": true,
	"me": true, "my": true, "need": true, "of": true, "on": true, "our": true, "the": true, "this": true,
	"to": true, "want": true, "we": true, "with": true, "project": true, "app": true, "fastbuild": true,
}

// searchTerms splits a query on anything that is not a letter or digit and drops stop words.
func searchTerms(query string) []string {
	var terms []string
	for _, term := range strings.FieldsFunc(query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if !searchStopWords[term] {
			terms = append(terms, term)
		}
	}
	return terms
}

func termScore(entry IndexEntry, term string) int {
	score := 0
	if strings.Contains(entry.Name, term) {
		score += 10
	}
	for _, keyword := range entry.Keywords {
		if strings.Contains(strings.ToLower(keyword), term) {
			score += 6
		}
	}
	for _, situation := range entry.UseWhen {
		if strings.Contains(strings.ToLower(situation), term) {
			score += 4
		}
	}
	if strings.Contains(strings.ToLower(entry.Summary), term) {
		score += 3
	}
	if strings.Contains(strings.ToLower(entry.Description), term) {
		score++
	}
	return score
}

// embeddedScore credits keywords and name words that appear inside the query itself, such as
// "知识库" in "我们需要一个知识库".
func embeddedScore(entry IndexEntry, query string) int {
	score := 0
	for _, keyword := range entry.Keywords {
		keyword = strings.ToLower(keyword)
		if utf8.RuneCountInString(keyword) >= 2 && !isASCII(keyword) && strings.Contains(query, keyword) {
			score += 6
		}
	}
	return score
}

func isASCII(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// PluginFilePath returns the path of a plugin file inside a registry directory.
func PluginFilePath(registryDir, plugin, file string) string {
	return filepath.Join(registryDir, PluginsDir, plugin, FilesDir, filepath.FromSlash(path.Clean(file)))
}

package pluginregistry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	maxArchiveBytes   = 50 << 20
	maxArchiveEntries = 20_000
	maxExtractedBytes = 100 << 20
	downloadTimeout   = 2 * time.Minute
)

var (
	remotePattern = regexp.MustCompile(`^github:([A-Za-z0-9-]+)/([A-Za-z0-9._-]+)@([A-Za-z0-9._/-]+)$`)
	// ArchiveBaseURL is the GitHub archive endpoint; tests point it at a local server.
	ArchiveBaseURL = "https://codeload.github.com"
)

// RemoteSource reads the registry/ directory of a tagged GitHub release. The archive is fetched
// over HTTPS once, bounded in size and entry count, and only regular files under registry/ are
// extracted; every installed file is then verified against the archive's own index.
type RemoteSource struct {
	Owner, Repo, Ref string
	Client           *http.Client

	once  sync.Once
	local *LocalSource
	err   error
}

// ParseRemoteSource parses "github:owner/repo@ref".
func ParseRemoteSource(raw string) (*RemoteSource, error) {
	match := remotePattern.FindStringSubmatch(raw)
	if match == nil || strings.Contains(match[3], "..") {
		return nil, fmt.Errorf("source %q must look like github:owner/repo@tag", raw)
	}
	return &RemoteSource{Owner: match[1], Repo: match[2], Ref: match[3]}, nil
}

// Describe implements Source.
func (s *RemoteSource) Describe() string {
	return fmt.Sprintf("github:%s/%s@%s", s.Owner, s.Repo, s.Ref)
}

// Index implements Source.
func (s *RemoteSource) Index() (*Index, error) {
	local, err := s.fetch()
	if err != nil {
		return nil, err
	}
	return local.Index()
}

// ReadFile implements Source.
func (s *RemoteSource) ReadFile(entry *IndexEntry, file string) ([]byte, error) {
	local, err := s.fetch()
	if err != nil {
		return nil, err
	}
	return local.ReadFile(entry, file)
}

// ReadDocument implements Source.
func (s *RemoteSource) ReadDocument(plugin, name string) ([]byte, error) {
	local, err := s.fetch()
	if err != nil {
		return nil, err
	}
	return local.ReadDocument(plugin, name)
}

// Close removes the extracted archive.
func (s *RemoteSource) Close() error {
	if s.local == nil {
		return nil
	}
	return os.RemoveAll(s.local.Dir)
}

func (s *RemoteSource) fetch() (*LocalSource, error) {
	s.once.Do(func() {
		dir, err := os.MkdirTemp("", "fastbuild-registry-*")
		if err != nil {
			s.err = err
			return
		}
		if err := s.download(dir); err != nil {
			_ = os.RemoveAll(dir)
			s.err = fmt.Errorf("fetch %s: %w", s.Describe(), err)
			return
		}
		s.local = NewLocalSource(dir, s.Describe())
	})
	return s.local, s.err
}

func (s *RemoteSource) download(dir string) error {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: downloadTimeout}
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	url := fmt.Sprintf("%s/%s/%s/tar.gz/%s", strings.TrimSuffix(ArchiveBaseURL, "/"), s.Owner, s.Repo, s.Ref)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}
	return ExtractRegistry(io.LimitReader(response.Body, maxArchiveBytes+1), dir)
}

// ExtractRegistry extracts the registry/ directory of a GitHub tar.gz archive into dir. It rejects
// archives that are too large, contain too many entries, or contain links or unsafe paths.
func ExtractRegistry(archive io.Reader, dir string) error {
	counted := &countingReader{reader: archive, limit: maxArchiveBytes}
	gz, err := gzip.NewReader(counted)
	if err != nil {
		return fmt.Errorf("archive is not gzip: %w", err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	var entries, extracted int64
	found := false
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		if entries++; entries > maxArchiveEntries {
			return fmt.Errorf("archive has more than %d entries", maxArchiveEntries)
		}
		// GitHub archives wrap everything in one "<repo>-<ref>/" directory.
		_, rest, ok := strings.Cut(header.Name, "/")
		if !ok || !strings.HasPrefix(rest, "registry/") {
			continue
		}
		relative := strings.TrimPrefix(rest, "registry/")
		if relative == "" {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
		default:
			return fmt.Errorf("archive entry %s is not a regular file", rest)
		}
		clean := strings.TrimSuffix(relative, "/")
		if !ValidFilePath(clean) || path.Clean(clean) != clean {
			return fmt.Errorf("archive entry %s has an unsafe path", rest)
		}
		if extracted += header.Size; extracted > maxExtractedBytes || header.Size > maxPluginFileSize {
			return fmt.Errorf("archive registry exceeds the size limit")
		}
		target := filepath.Join(dir, filepath.FromSlash(clean))
		if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o755); mkdirErr != nil {
			return mkdirErr
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, io.LimitReader(reader, header.Size))
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return errors.Join(copyErr, closeErr)
		}
		found = true
	}
	if counted.exceeded {
		return fmt.Errorf("archive exceeds %d bytes", maxArchiveBytes)
	}
	if !found {
		return errors.New("archive has no registry/ directory; the release predates the plugin registry")
	}
	return nil
}

type countingReader struct {
	reader   io.Reader
	limit    int64
	read     int64
	exceeded bool
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.read += int64(n)
	if r.read > r.limit {
		r.exceeded = true
		return n, fmt.Errorf("archive exceeds %d bytes", r.limit)
	}
	return n, err
}

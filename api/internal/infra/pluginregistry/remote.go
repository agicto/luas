package pluginregistry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/fastagi/fastbuild/api/internal/infra/projectfiles"
)

var remotePattern = regexp.MustCompile(`^github:([A-Za-z0-9-]+)/([A-Za-z0-9._-]+)@([A-Za-z0-9._/-]+)$`)

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
	return projectfiles.FetchGitHubArchive(context.Background(), s.Client, s.Owner, s.Repo, s.Ref, "registry/", dir)
}

// ExtractRegistry extracts the registry/ directory of a GitHub tar.gz archive into dir, with the
// bounds of projectfiles.ExtractArchive.
func ExtractRegistry(archive io.Reader, dir string) error {
	err := projectfiles.ExtractArchive(archive, "registry/", dir)
	if errors.Is(err, projectfiles.ErrPrefixMissing) {
		return errors.New("archive has no registry/ directory; the release predates the plugin registry")
	}
	return err
}

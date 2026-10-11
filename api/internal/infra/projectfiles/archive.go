package projectfiles

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
	"strings"
	"time"
)

// Archive limits keep a downloaded release from exhausting disk or memory.
const (
	MaxArchiveBytes   = 50 << 20
	MaxArchiveEntries = 20_000
	MaxExtractedBytes = 100 << 20
	MaxFileBytes      = 2 << 20
	downloadTimeout   = 2 * time.Minute
)

// GitHubArchiveBaseURL is the GitHub archive endpoint; tests point it at a local server.
var GitHubArchiveBaseURL = "https://codeload.github.com"

// ErrPrefixMissing reports an archive without the requested directory.
var ErrPrefixMissing = errors.New("archive has no files under the requested directory")

// FetchGitHubArchive downloads the tar.gz archive of owner/repo at ref over HTTPS and extracts the
// files under prefix (for example "registry/", or "" for the whole repository) into dir.
func FetchGitHubArchive(ctx context.Context, client *http.Client, owner, repo, ref, prefix, dir string) error {
	if client == nil {
		client = &http.Client{Timeout: downloadTimeout}
	}
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	url := fmt.Sprintf("%s/%s/%s/tar.gz/%s", strings.TrimSuffix(GitHubArchiveBaseURL, "/"), owner, repo, ref)
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
	return ExtractArchive(io.LimitReader(response.Body, MaxArchiveBytes+1), prefix, dir)
}

// ExtractArchive extracts the regular files under prefix of a GitHub tar.gz archive into dir,
// without the archive's top-level "<repo>-<ref>/" directory or the prefix. It rejects archives that
// are too large or have too many entries, and entries that are links or have unsafe paths.
func ExtractArchive(archive io.Reader, prefix, dir string) error {
	counted := &countingReader{reader: archive, limit: MaxArchiveBytes}
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
		if entries++; entries > MaxArchiveEntries {
			return fmt.Errorf("archive has more than %d entries", MaxArchiveEntries)
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue // GitHub records the commit in a global pax header
		}
		_, rest, ok := strings.Cut(header.Name, "/")
		if !ok || !strings.HasPrefix(rest, prefix) {
			continue
		}
		relative := strings.TrimSuffix(strings.TrimPrefix(rest, prefix), "/")
		if relative == "" || header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("archive entry %s is not a regular file", rest)
		}
		if !SafeRelativePath(relative) {
			return fmt.Errorf("archive entry %s has an unsafe path", rest)
		}
		if extracted += header.Size; extracted > MaxExtractedBytes || header.Size > MaxFileBytes {
			return errors.New("archive exceeds the size limit")
		}
		target := filepath.Join(dir, filepath.FromSlash(relative))
		if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o755); mkdirErr != nil {
			return mkdirErr
		}
		mode := os.FileMode(0o644)
		if header.Mode&0o111 != 0 {
			mode = 0o755 // keep scripts executable
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
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
		return fmt.Errorf("archive exceeds %d bytes", MaxArchiveBytes)
	}
	if !found {
		return ErrPrefixMissing
	}
	return nil
}

// SafeRelativePath reports whether name is a clean, slash-separated path inside a checkout and
// outside .git.
func SafeRelativePath(name string) bool {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return false
	}
	clean := path.Clean(name)
	return clean == name && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") &&
		clean != ".git" && !strings.HasPrefix(clean, ".git/")
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

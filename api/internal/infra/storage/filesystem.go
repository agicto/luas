package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Filesystem defines the interface for file operations
type Filesystem interface {
	// File operations
	Exists(path string) bool
	Get(path string) ([]byte, error)
	Put(path string, contents []byte) error
	Append(path string, contents []byte) error
	Delete(paths ...string) error
	Copy(from, to string) error
	Move(from, to string) error

	// File info
	Size(path string) (int64, error)
	LastModified(path string) (time.Time, error)
	MimeType(path string) string

	// Directory operations
	Files(directory string) ([]string, error)
	AllFiles(directory string) ([]string, error)
	Directories(directory string) ([]string, error)
	AllDirectories(directory string) ([]string, error)
	MakeDirectory(path string) error
	DeleteDirectory(path string) error

	// URL (for cloud storage)
	URL(path string) string
	TemporaryURL(path string, expiration time.Duration) (string, error)

	// Stream operations
	ReadStream(path string) (io.ReadCloser, error)
	WriteStream(path string, stream io.Reader) error
}

// FileInfo holds file metadata
type FileInfo struct {
	Path         string
	Name         string
	Extension    string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

// Manager manages multiple filesystem disks
type Manager struct {
	disks       map[string]Filesystem
	defaultDisk string
}

// config holds filesystem configuration
var manager = &Manager{
	disks:       make(map[string]Filesystem),
	defaultDisk: "local",
}

// RegisterDisk registers a filesystem disk
func RegisterDisk(name string, fs Filesystem) {
	manager.disks[name] = fs
}

// SetDefaultDisk sets the default disk
func SetDefaultDisk(name string) {
	manager.defaultDisk = name
}

// Disk returns a filesystem by name
func Disk(name ...string) Filesystem {
	diskName := manager.defaultDisk
	if len(name) > 0 {
		diskName = name[0]
	}

	if fs, ok := manager.disks[diskName]; ok {
		return fs
	}

	// Return local disk if not found
	return manager.disks["local"]
}

// --- Local Filesystem Implementation ---

// LocalFilesystem implements Filesystem for local disk
type LocalFilesystem struct {
	root string
}

// NewLocalFilesystem creates a new local filesystem
func NewLocalFilesystem(root string) *LocalFilesystem {
	// Ensure root exists
	os.MkdirAll(root, 0755)
	return &LocalFilesystem{root: root}
}

// path returns the full path
func (fs *LocalFilesystem) path(p string) string {
	return filepath.Join(fs.root, p)
}

// Exists checks if a file exists
func (fs *LocalFilesystem) Exists(path string) bool {
	_, err := os.Stat(fs.path(path))
	return err == nil
}

// Get reads a file's contents
func (fs *LocalFilesystem) Get(path string) ([]byte, error) {
	return os.ReadFile(fs.path(path))
}

// Put writes contents to a file
func (fs *LocalFilesystem) Put(path string, contents []byte) error {
	fullPath := fs.path(path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, contents, 0644)
}

// Append appends contents to a file
func (fs *LocalFilesystem) Append(path string, contents []byte) error {
	fullPath := fs.path(path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(contents)
	return err
}

// Delete removes files
func (fs *LocalFilesystem) Delete(paths ...string) error {
	for _, path := range paths {
		if err := os.Remove(fs.path(path)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Copy copies a file
func (fs *LocalFilesystem) Copy(from, to string) error {
	src, err := os.Open(fs.path(from))
	if err != nil {
		return err
	}
	defer src.Close()

	// Ensure destination directory exists
	destPath := fs.path(to)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// Move moves a file
func (fs *LocalFilesystem) Move(from, to string) error {
	// Ensure destination directory exists
	destPath := fs.path(to)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.Rename(fs.path(from), destPath)
}

// Size returns the file size
func (fs *LocalFilesystem) Size(path string) (int64, error) {
	info, err := os.Stat(fs.path(path))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// LastModified returns the last modification time
func (fs *LocalFilesystem) LastModified(path string) (time.Time, error) {
	info, err := os.Stat(fs.path(path))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// MimeType returns the MIME type based on extension
func (fs *LocalFilesystem) MimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mimeTypes := map[string]string{
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".txt":  "text/plain",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// Files returns files in a directory (non-recursive)
func (fs *LocalFilesystem) Files(directory string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(fs.path(directory))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(directory, entry.Name()))
		}
	}
	return files, nil
}

// AllFiles returns all files recursively
func (fs *LocalFilesystem) AllFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(fs.path(directory), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(fs.root, path)
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}

// Directories returns directories in a directory (non-recursive)
func (fs *LocalFilesystem) Directories(directory string) ([]string, error) {
	var dirs []string
	entries, err := os.ReadDir(fs.path(directory))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(directory, entry.Name()))
		}
	}
	return dirs, nil
}

// AllDirectories returns all directories recursively
func (fs *LocalFilesystem) AllDirectories(directory string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(fs.path(directory), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != fs.path(directory) {
			relPath, _ := filepath.Rel(fs.root, path)
			dirs = append(dirs, relPath)
		}
		return nil
	})
	return dirs, err
}

// MakeDirectory creates a directory
func (fs *LocalFilesystem) MakeDirectory(path string) error {
	return os.MkdirAll(fs.path(path), 0755)
}

// DeleteDirectory removes a directory
func (fs *LocalFilesystem) DeleteDirectory(path string) error {
	return os.RemoveAll(fs.path(path))
}

// URL returns the URL (for local, just the path)
func (fs *LocalFilesystem) URL(path string) string {
	return "/" + path
}

// TemporaryURL returns a temporary URL (not supported for local)
func (fs *LocalFilesystem) TemporaryURL(path string, expiration time.Duration) (string, error) {
	return fs.URL(path), nil
}

// ReadStream opens a file for reading
func (fs *LocalFilesystem) ReadStream(path string) (io.ReadCloser, error) {
	return os.Open(fs.path(path))
}

// WriteStream writes from a stream
func (fs *LocalFilesystem) WriteStream(path string, stream io.Reader) error {
	fullPath := fs.path(path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, stream)
	return err
}

// --- Package-level convenience functions ---

// Exists checks if a file exists (uses default disk)
func Exists(path string) bool {
	return Disk().Exists(path)
}

// Get reads a file (uses default disk)
func Get(path string) ([]byte, error) {
	return Disk().Get(path)
}

// Put writes a file (uses default disk)
func Put(path string, contents []byte) error {
	return Disk().Put(path, contents)
}

// Delete removes files (uses default disk)
func Delete(paths ...string) error {
	return Disk().Delete(paths...)
}

// Copy copies a file (uses default disk)
func Copy(from, to string) error {
	return Disk().Copy(from, to)
}

// Move moves a file (uses default disk)
func Move(from, to string) error {
	return Disk().Move(from, to)
}

// init registers the default local disk
func init() {
	RegisterDisk("local", NewLocalFilesystem("storage"))
}

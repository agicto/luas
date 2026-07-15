package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/google/uuid"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
)

// LocalStore keeps development objects under one traversal-resistant filesystem root.
type LocalStore struct {
	root string
}

var _ storagecap.ObjectStore = (*LocalStore)(nil)

// NewLocalStore creates a private local object root. Local storage is not a production replica store.
func NewLocalStore(root string) (*LocalStore, error) {
	if root == "" {
		return nil, fmt.Errorf("local object root is required")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local object root: %w", err)
	}
	if mkdirErr := os.MkdirAll(absoluteRoot, 0o700); mkdirErr != nil {
		return nil, fmt.Errorf("create local object root: %w", mkdirErr)
	}
	rootHandle, err := os.OpenRoot(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("open local object root: %w", err)
	}
	if err := rootHandle.Close(); err != nil {
		return nil, fmt.Errorf("close local object root: %w", err)
	}
	return &LocalStore{root: absoluteRoot}, nil
}

func (s *LocalStore) Driver() string { return storagecap.DriverLocal }

func (s *LocalStore) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	_ string,
) error {
	if err := readyForOperation(ctx, s, key); err != nil {
		return err
	}
	if body == nil || size < 0 {
		return storagecap.ErrObjectSizeMismatch
	}

	root, err := os.OpenRoot(s.root)
	if err != nil {
		return fmt.Errorf("open local object root: %w", err)
	}
	defer root.Close()

	directory := path.Dir(key)
	if directory != "." {
		if mkdirErr := root.MkdirAll(directory, 0o700); mkdirErr != nil {
			return fmt.Errorf("create local object directory: %w", mkdirErr)
		}
	}
	temporaryKey := key + ".tmp-" + uuid.NewString()
	file, err := root.OpenFile(temporaryKey, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary local object: %w", err)
	}
	removeTemporary := true
	defer func() {
		_ = file.Close() //nolint:errcheck // best-effort cleanup after the primary operation
		if removeTemporary {
			_ = root.Remove(temporaryKey) //nolint:errcheck // best-effort cleanup of an uncommitted temp file
		}
	}()

	written, copyErr := io.Copy(file, io.LimitReader(body, size+1))
	if copyErr != nil {
		return fmt.Errorf("write local object: %w", copyErr)
	}
	if written != size {
		return storagecap.ErrObjectSizeMismatch
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync local object: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close local object: %w", err)
	}
	if err := root.Rename(temporaryKey, key); err != nil {
		return fmt.Errorf("commit local object: %w", err)
	}
	removeTemporary = false
	return nil
}

func (s *LocalStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := readyForOperation(ctx, s, key); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, fmt.Errorf("open local object root: %w", err)
	}
	file, err := root.Open(key)
	closeErr := root.Close()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, storagecap.ErrObjectNotFound
		}
		return nil, fmt.Errorf("open local object: %w", err)
	}
	if closeErr != nil {
		_ = file.Close()
		return nil, fmt.Errorf("close local object root: %w", closeErr)
	}
	return file, nil
}

func (s *LocalStore) Stat(ctx context.Context, key string) (storagecap.ObjectInfo, error) {
	if err := readyForOperation(ctx, s, key); err != nil {
		return storagecap.ObjectInfo{}, err
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return storagecap.ObjectInfo{}, fmt.Errorf("open local object root: %w", err)
	}
	defer root.Close()
	info, err := root.Stat(key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return storagecap.ObjectInfo{}, storagecap.ErrObjectNotFound
		}
		return storagecap.ObjectInfo{}, fmt.Errorf("stat local object: %w", err)
	}
	if !info.Mode().IsRegular() {
		return storagecap.ObjectInfo{}, storagecap.ErrObjectNotFound
	}
	return storagecap.ObjectInfo{Size: info.Size(), LastModified: info.ModTime()}, nil
}

func (s *LocalStore) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	if err := storagecap.ValidateObjectKey(destinationKey); err != nil {
		return err
	}
	info, err := s.Stat(ctx, sourceKey)
	if err != nil {
		return err
	}
	reader, err := s.Open(ctx, sourceKey)
	if err != nil {
		return err
	}
	defer reader.Close()
	return s.Put(ctx, destinationKey, reader, info.Size, info.MediaType)
}

func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := readyForOperation(ctx, s, key); err != nil {
		return err
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return fmt.Errorf("open local object root: %w", err)
	}
	defer root.Close()
	if err := root.Remove(key); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete local object: %w", err)
	}
	return nil
}

func readyForOperation(ctx context.Context, store *LocalStore, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil || store.root == "" {
		return fmt.Errorf("local object store is unavailable")
	}
	return storagecap.ValidateObjectKey(key)
}

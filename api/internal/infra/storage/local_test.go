package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
)

func TestLocalStoreRoundTripCopyAndDelete(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	payload := []byte("hello object storage")

	if putErr := store.Put(ctx, "asset-uploads/source/object", bytes.NewReader(payload), int64(len(payload)), "text/plain"); putErr != nil {
		t.Fatalf("Put() error = %v", putErr)
	}
	if copyErr := store.Copy(ctx, "asset-uploads/source/object", "assets/final/object"); copyErr != nil {
		t.Fatalf("Copy() error = %v", copyErr)
	}
	info, err := store.Stat(ctx, "assets/final/object")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size != int64(len(payload)) {
		t.Fatalf("Stat().Size = %d, want %d", info.Size, len(payload))
	}
	reader, err := store.Open(ctx, "assets/final/object")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()
	got := make([]byte, len(payload))
	if _, err := reader.Read(got); err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("stored payload = %q, want %q", got, payload)
	}
	if err := store.Delete(ctx, "assets/final/object"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := store.Delete(ctx, "assets/final/object"); err != nil {
		t.Fatalf("idempotent Delete() error = %v", err)
	}
	if _, err := store.Stat(ctx, "assets/final/object"); !errors.Is(err, storagecap.ErrObjectNotFound) {
		t.Fatalf("Stat() after delete error = %v, want ErrObjectNotFound", err)
	}
}

func TestLocalStoreRejectsWrongSizeWithoutPartialObject(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(context.Background(), "asset-uploads/short/object", bytes.NewReader([]byte("too long")), 3, "text/plain")
	if !errors.Is(err, storagecap.ErrObjectSizeMismatch) {
		t.Fatalf("Put() error = %v, want ErrObjectSizeMismatch", err)
	}
	if _, statErr := store.Stat(context.Background(), "asset-uploads/short/object"); !errors.Is(statErr, storagecap.ErrObjectNotFound) {
		t.Fatalf("partial object Stat() error = %v, want ErrObjectNotFound", statErr)
	}
}

func TestLocalStoreRejectsParentTraversal(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "objects")
	store, err := NewLocalStore(root)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Put(context.Background(), "../escaped.txt", bytes.NewReader([]byte("escape")), 6, "text/plain")
	if !errors.Is(err, storagecap.ErrInvalidObjectKey) {
		t.Fatalf("Put() error = %v, want ErrInvalidObjectKey", err)
	}
	if _, statErr := os.Stat(filepath.Join(parent, "escaped.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("parent traversal created an object outside root: %v", statErr)
	}
}

func TestLocalStoreRejectsSymlinkEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "objects")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	store, err := NewLocalStore(root)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Put(context.Background(), "link/escaped.txt", bytes.NewReader([]byte("escape")), 6, "text/plain")
	if err == nil {
		t.Fatal("Put() error = nil, want symlink containment rejection")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "escaped.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("symlink escape created an object outside root: %v", statErr)
	}
}

func TestLocalStoreHonorsCanceledContext(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = store.Put(ctx, "assets/canceled/object", bytes.NewReader(nil), 0, "application/octet-stream")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Put() error = %v, want context.Canceled", err)
	}
}

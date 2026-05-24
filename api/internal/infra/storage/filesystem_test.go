package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFilesystem_PutAndGet(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	content := []byte("Hello, World!")
	err := fs.Put("test.txt", content)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := fs.Get("test.txt")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("Expected %s, got %s", content, got)
	}
}

func TestLocalFilesystem_Exists(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	if fs.Exists("nonexistent.txt") {
		t.Error("Exists should return false for non-existent file")
	}

	fs.Put("exists.txt", []byte("test"))
	if !fs.Exists("exists.txt") {
		t.Error("Exists should return true for existing file")
	}
}

func TestLocalFilesystem_Delete(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	fs.Put("delete.txt", []byte("test"))
	if !fs.Exists("delete.txt") {
		t.Fatal("File should exist before delete")
	}

	err := fs.Delete("delete.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if fs.Exists("delete.txt") {
		t.Error("File should not exist after delete")
	}
}

func TestLocalFilesystem_Copy(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	content := []byte("copy me")
	fs.Put("original.txt", content)

	err := fs.Copy("original.txt", "copied.txt")
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	got, _ := fs.Get("copied.txt")
	if string(got) != string(content) {
		t.Errorf("Copied content mismatch")
	}

	// Original should still exist
	if !fs.Exists("original.txt") {
		t.Error("Original file should still exist after copy")
	}
}

func TestLocalFilesystem_Move(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	content := []byte("move me")
	fs.Put("source.txt", content)

	err := fs.Move("source.txt", "destination.txt")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if fs.Exists("source.txt") {
		t.Error("Source should not exist after move")
	}

	if !fs.Exists("destination.txt") {
		t.Error("Destination should exist after move")
	}
}

func TestLocalFilesystem_Append(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	fs.Put("append.txt", []byte("Hello"))
	fs.Append("append.txt", []byte(" World"))

	got, _ := fs.Get("append.txt")
	if string(got) != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", got)
	}
}

func TestLocalFilesystem_Size(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	content := []byte("12345")
	fs.Put("size.txt", content)

	size, err := fs.Size("size.txt")
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}

	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}

func TestLocalFilesystem_MimeType(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	tests := []struct {
		path     string
		expected string
	}{
		{"file.html", "text/html"},
		{"file.json", "application/json"},
		{"file.png", "image/png"},
		{"file.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		got := fs.MimeType(tt.path)
		if got != tt.expected {
			t.Errorf("MimeType(%s) = %s, want %s", tt.path, got, tt.expected)
		}
	}
}

func TestLocalFilesystem_Directories(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalFilesystem(root)

	// Create directories
	os.MkdirAll(filepath.Join(root, "dir1"), 0755)
	os.MkdirAll(filepath.Join(root, "dir2"), 0755)

	// Create a file
	fs.Put("file.txt", []byte("test"))

	dirs, err := fs.Directories("")
	if err != nil {
		t.Fatalf("Directories failed: %v", err)
	}

	if len(dirs) != 2 {
		t.Errorf("Expected 2 directories, got %d", len(dirs))
	}
}

func TestLocalFilesystem_Files(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalFilesystem(root)

	// Create files
	fs.Put("file1.txt", []byte("test"))
	fs.Put("file2.txt", []byte("test"))

	// Create a directory
	os.MkdirAll(filepath.Join(root, "dir"), 0755)

	files, err := fs.Files("")
	if err != nil {
		t.Fatalf("Files failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestLocalFilesystem_MakeAndDeleteDirectory(t *testing.T) {
	fs := NewLocalFilesystem(t.TempDir())

	err := fs.MakeDirectory("new/nested/dir")
	if err != nil {
		t.Fatalf("MakeDirectory failed: %v", err)
	}

	if !fs.Exists("new/nested/dir") {
		t.Error("Directory should exist after creation")
	}

	err = fs.DeleteDirectory("new")
	if err != nil {
		t.Fatalf("DeleteDirectory failed: %v", err)
	}

	if fs.Exists("new") {
		t.Error("Directory should not exist after deletion")
	}
}

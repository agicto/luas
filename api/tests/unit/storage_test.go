package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgiai/zgo/internal/infra/storage"
)

func TestLocalFilesystemSmoke(t *testing.T) {
	fs := storage.NewLocalFilesystem(t.TempDir())

	err := fs.Put("docs/readme.txt", []byte("hello world"))
	assert.NoError(t, err)
	assert.True(t, fs.Exists("docs/readme.txt"))

	content, err := fs.Get("docs/readme.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestLocalFilesystemCopyMoveAndDelete(t *testing.T) {
	fs := storage.NewLocalFilesystem(t.TempDir())
	assert.NoError(t, fs.Put("source.txt", []byte("payload")))

	assert.NoError(t, fs.Copy("source.txt", "copy.txt"))
	assert.True(t, fs.Exists("source.txt"))
	assert.True(t, fs.Exists("copy.txt"))

	assert.NoError(t, fs.Move("copy.txt", "nested/moved.txt"))
	assert.False(t, fs.Exists("copy.txt"))
	assert.True(t, fs.Exists("nested/moved.txt"))

	size, err := fs.Size("nested/moved.txt")
	assert.NoError(t, err)
	assert.EqualValues(t, 7, size)

	assert.NoError(t, fs.Delete("source.txt", "nested/moved.txt"))
	assert.False(t, fs.Exists("source.txt"))
	assert.False(t, fs.Exists("nested/moved.txt"))
}

func TestLocalFilesystemURL(t *testing.T) {
	fs := storage.NewLocalFilesystem(t.TempDir())

	assert.Equal(t, "/images/photo.jpg", fs.URL("images/photo.jpg"))
}

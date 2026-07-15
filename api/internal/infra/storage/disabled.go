package storage

import (
	"context"
	"io"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
)

type disabledStore struct{}

var _ storagecap.ObjectStore = (*disabledStore)(nil)

func newDisabledStore() storagecap.ObjectStore { return &disabledStore{} }

func (*disabledStore) Driver() string { return "disabled" }

func (*disabledStore) Put(context.Context, string, io.Reader, int64, string) error {
	return storagecap.ErrObjectStoreUnavailable
}

func (*disabledStore) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, storagecap.ErrObjectStoreUnavailable
}

func (*disabledStore) Stat(context.Context, string) (storagecap.ObjectInfo, error) {
	return storagecap.ObjectInfo{}, storagecap.ErrObjectStoreUnavailable
}

func (*disabledStore) Copy(context.Context, string, string) error {
	return storagecap.ErrObjectStoreUnavailable
}

func (*disabledStore) Delete(context.Context, string) error {
	return storagecap.ErrObjectStoreUnavailable
}

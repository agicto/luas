package storage

import (
	"fmt"
	"strings"

	"github.com/google/wire"

	storagecap "github.com/zgiai/luas/api/internal/capabilities/storage"
	"github.com/zgiai/luas/api/internal/infra/config"
)

// ProviderSet exposes one provider-neutral store selected from the typed startup snapshot.
var ProviderSet = wire.NewSet(NewObjectStore)

// NewObjectStore constructs only the explicitly selected object-storage adapter.
func NewObjectStore(cfg *config.Config) (storagecap.ObjectStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for object storage")
	}
	switch strings.TrimSpace(cfg.ObjectStorage.Driver) {
	case "", "disabled":
		return newDisabledStore(), nil
	case storagecap.DriverLocal:
		return NewLocalStore(cfg.ObjectStorage.LocalRoot)
	case storagecap.DriverR2:
		return NewR2Store(R2Options{
			AccessKeyID:     cfg.R2.AccessKeyID,
			SecretAccessKey: cfg.R2.SecretAccessKey,
			Bucket:          cfg.R2.Bucket,
			Region:          cfg.R2.Region,
			Endpoint:        cfg.R2.Endpoint,
			RequestTimeout:  cfg.ObjectStorage.RequestTimeout,
		})
	default:
		return nil, fmt.Errorf("unsupported object storage driver")
	}
}

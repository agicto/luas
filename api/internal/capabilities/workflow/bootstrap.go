package workflow

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// QueueRuntimeConfig is the workflow-owned queue runtime configuration.
type QueueRuntimeConfig struct {
	Driver       string
	DefaultQueue string
	BufferSize   int
	Database     *gorm.DB
}

// Bootstrap applies process-level workflow runtime configuration.
// It configures the default queue driver and queue name used by the workflow capability.
func Bootstrap(cfg QueueRuntimeConfig) (*Manager, error) {
	manager := Default()

	queueManager := GlobalQueue()
	driverName := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if driverName == "" {
		driverName = "sync"
	}

	switch driverName {
	case "sync":
		queueManager.RegisterDriver("sync", NewSyncDriver())
	case "memory":
		if existing := queueManager.Driver("memory"); existing == nil {
			bufferSize := cfg.BufferSize
			if bufferSize < 1 {
				bufferSize = 256
			}
			queueManager.RegisterDriver("memory", NewMemoryDriver(bufferSize))
		}
	case "postgres":
		driver, err := NewPostgresDriver(cfg.Database)
		if err != nil {
			return nil, err
		}
		queueManager.RegisterDriver("postgres", driver)
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Driver)
	}

	if err := queueManager.SetDefaultDriver(driverName); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.DefaultQueue) != "" {
		queueManager.SetDefaultQueue(cfg.DefaultQueue)
	}

	return manager, nil
}

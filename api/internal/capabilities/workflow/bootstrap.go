package workflow

import (
	"fmt"
	"strings"

	"github.com/zgiai/luas/api/internal/infra/queue"
)

// QueueRuntimeConfig is the workflow-owned queue runtime configuration.
type QueueRuntimeConfig struct {
	Driver       string
	DefaultQueue string
	BufferSize   int
}

// Bootstrap applies process-level workflow runtime configuration.
// It configures the default queue driver and queue name used by the workflow capability.
func Bootstrap(cfg QueueRuntimeConfig) (*Manager, error) {
	manager := Default()

	queueManager := queue.Global()
	driverName := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if driverName == "" {
		driverName = "sync"
	}

	switch driverName {
	case "sync":
		queueManager.RegisterDriver("sync", queue.NewSyncDriver())
	case "memory":
		if existing := queueManager.Driver("memory"); existing == nil {
			bufferSize := cfg.BufferSize
			if bufferSize < 1 {
				bufferSize = 256
			}
			queueManager.RegisterDriver("memory", queue.NewMemoryDriver(bufferSize))
		}
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

package workflow

import (
	"fmt"
	"strings"

	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/queue"
)

// Bootstrap applies process-level workflow runtime configuration.
// It configures the default queue driver and queue name used by the workflow capability.
func Bootstrap(cfg *config.Config) (*Manager, error) {
	manager := Default()
	if cfg == nil {
		return manager, nil
	}

	queueManager := queue.Global()
	driverName := strings.ToLower(strings.TrimSpace(cfg.Queue.Driver))
	if driverName == "" {
		driverName = "sync"
	}

	switch driverName {
	case "sync":
		queueManager.RegisterDriver("sync", queue.NewSyncDriver())
	case "memory":
		if existing := queueManager.Driver("memory"); existing == nil {
			bufferSize := cfg.Queue.BufferSize
			if bufferSize < 1 {
				bufferSize = 256
			}
			queueManager.RegisterDriver("memory", queue.NewMemoryDriver(bufferSize))
		}
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}

	if err := queueManager.SetDefaultDriver(driverName); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.Queue.DefaultQueue) != "" {
		queueManager.SetDefaultQueue(cfg.Queue.DefaultQueue)
	}

	return manager, nil
}

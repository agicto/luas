package bootstrap

import (
	"github.com/zgiai/zgo/pkg/logger"
)

// InitLogger initializes the logger.
// Called before Wire initialization since logger is used during startup.
func InitLogger() {
	logger.Boot()
}

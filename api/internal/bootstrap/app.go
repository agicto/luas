package bootstrap

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/pkg/logger"
)

// InitLogger initializes logging from the same typed configuration snapshot
// passed into the application dependency graph.
func InitLogger(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	runtimeLogger := logger.New(buildLoggerConfig(cfg))
	if dsn := strings.TrimSpace(cfg.Sentry.DSN); dsn != "" {
		handler, err := logger.NewSentryHandler(dsn, cfg.App.Env)
		if err != nil {
			return fmt.Errorf("initialize Sentry logger: %w", err)
		}
		runtimeLogger.AddHandler(handler)
	}
	logger.SetDefault(runtimeLogger)
	return nil
}

func buildLoggerConfig(cfg *config.Config) *logger.Config {
	logConfig := logger.DefaultConfig()
	logConfig.Level = logger.ParseLevel(cfg.Log.Level)
	logConfig.StdoutPrint = cfg.Log.Stdout
	logConfig.FileEnabled = cfg.Log.FileEnabled
	logConfig.JSON = cfg.Log.JSON
	logConfig.ColorEnabled = cfg.App.Debug && !cfg.Log.JSON

	if file := strings.TrimSpace(cfg.Log.File); file != "" {
		logConfig.Path = filepath.Dir(file)
		logConfig.File = filepath.Base(file)
	}
	return logConfig
}

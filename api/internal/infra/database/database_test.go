package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func TestResolveGormLogLevel_DefaultsToInfoForDebugAndTest(t *testing.T) {
	debugCfg := &config.Config{
		App: config.AppConfig{
			Env:   "development",
			Debug: true,
		},
	}

	testCfg := &config.Config{
		App: config.AppConfig{
			Env:   "test",
			Debug: false,
		},
	}

	assert.Equal(t, logger.Info, resolveGormLogLevel(debugCfg))
	assert.Equal(t, logger.Info, resolveGormLogLevel(testCfg))
}

func TestResolveGormLogLevel_DefaultsToWarnForProduction(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Env:   "production",
			Debug: false,
		},
	}

	assert.Equal(t, logger.Warn, resolveGormLogLevel(cfg))
}

func TestResolveGormLogLevel_UsesExplicitOverride(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Env:   "production",
			Debug: false,
		},
		Database: config.DatabaseConfig{
			LogLevel: "error",
		},
	}

	assert.Equal(t, logger.Error, resolveGormLogLevel(cfg))
}

func TestResolveGormLogLevel_FallsBackOnUnknownLevel(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Env:   "production",
			Debug: false,
		},
		Database: config.DatabaseConfig{
			LogLevel: "loud",
		},
	}

	assert.Equal(t, logger.Warn, resolveGormLogLevel(cfg))
}

func TestBuildLoggerConfig_UsesDatabaseSettings(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Env:   "production",
			Debug: false,
		},
		Database: config.DatabaseConfig{
			LogLevel:             "silent",
			SlowThreshold:        2 * time.Second,
			IgnoreRecordNotFound: false,
		},
	}

	loggerCfg := buildLoggerConfig(cfg)

	assert.Equal(t, logger.Silent, loggerCfg.LogLevel)
	assert.Equal(t, 2*time.Second, loggerCfg.SlowThreshold)
	assert.False(t, loggerCfg.IgnoreRecordNotFoundError)
	assert.True(t, loggerCfg.Colorful)
}

func TestNewDB_ReturnsNilWhenDisabled(t *testing.T) {
	db, err := NewDB(&config.Config{
		Database: config.DatabaseConfig{
			Enabled: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, db)
}

func TestNewDB_ReturnsErrorWhenEnabledDatabaseUnavailable(t *testing.T) {
	db, err := NewDB(&config.Config{
		App: config.AppConfig{
			Env: "test",
		},
		Database: config.DatabaseConfig{
			Enabled:              true,
			Driver:               "sqlite",
			Name:                 filepath.Join(t.TempDir(), "missing-parent", "luas.sqlite"),
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database is enabled but unavailable")
	assert.Nil(t, db)
}

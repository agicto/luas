package database

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/exception"
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
	assert.True(t, loggerCfg.ParameterizedQueries)
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

func TestNewDB_RejectsNilConfigAndUnknownDriver(t *testing.T) {
	db, err := NewDB(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration is required")
	assert.Nil(t, db)

	db, err = NewDB(&config.Config{
		Database: config.DatabaseConfig{
			Enabled: true,
			Driver:  "mysql",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_DRIVER")
	assert.Nil(t, db)
}

func TestNewDB_RejectsNonTLSPostgresInProduction(t *testing.T) {
	db, err := NewDB(&config.Config{
		App: config.AppConfig{Env: "production"},
		Database: config.DatabaseConfig{
			Enabled:              true,
			Driver:               "postgres",
			Host:                 "db.internal",
			Port:                 5432,
			Name:                 "luas",
			Username:             "luas",
			Password:             "secret",
			SSLMode:              "disable",
			Timezone:             "UTC",
			MaxIdleConns:         2,
			MaxOpenConns:         4,
			ConnMaxIdleTime:      15 * time.Minute,
			ConnMaxLifetime:      time.Hour,
			ConnectTimeout:       5 * time.Second,
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_SSLMODE")
	assert.Nil(t, db)
}

func TestPostgresDSN_EncodesCredentialsAndConnectionPolicy(t *testing.T) {
	dsn, err := postgresDSN(&config.Config{
		Database: config.DatabaseConfig{
			Enabled:              true,
			Driver:               "postgres",
			Host:                 "2001:db8::1",
			Port:                 5433,
			Name:                 "tenant/db",
			Username:             "user@example.com",
			Password:             "p@ss word:/?#[]",
			SSLMode:              "verify-full",
			Timezone:             "Europe/Dublin",
			MaxIdleConns:         2,
			MaxOpenConns:         4,
			ConnMaxIdleTime:      15 * time.Minute,
			ConnMaxLifetime:      time.Hour,
			ConnectTimeout:       7 * time.Second,
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})
	require.NoError(t, err)

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	password, hasPassword := parsed.User.Password()
	assert.True(t, hasPassword)
	assert.Equal(t, "user@example.com", parsed.User.Username())
	assert.Equal(t, "p@ss word:/?#[]", password)
	assert.Equal(t, "2001:db8::1", parsed.Hostname())
	assert.Equal(t, "5433", parsed.Port())
	assert.Equal(t, "/tenant/db", parsed.Path)
	assert.Equal(t, "verify-full", parsed.Query().Get("sslmode"))
	assert.Equal(t, "Europe/Dublin", parsed.Query().Get("timezone"))
	assert.Equal(t, "7", parsed.Query().Get("connect_timeout"))
	assert.Equal(t, "luas", parsed.Query().Get("application_name"))
	assert.NotContains(t, dsn, "p@ss word:/?#[]")
}

func TestNewDB_PostgresAppliesRuntimeConnectionPolicy(t *testing.T) {
	rawDSN := os.Getenv("LUAS_TEST_POSTGRES_DSN")
	if rawDSN == "" {
		t.Skip("LUAS_TEST_POSTGRES_DSN is not set")
	}
	parsed, err := url.Parse(rawDSN)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)
	password, _ := parsed.User.Password()
	sslMode := parsed.Query().Get("sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}

	db, err := NewDB(&config.Config{
		App: config.AppConfig{Name: "luas-database-test", Env: "test"},
		Database: config.DatabaseConfig{
			Enabled:              true,
			Driver:               "postgres",
			Host:                 parsed.Hostname(),
			Port:                 port,
			Name:                 strings.TrimPrefix(parsed.Path, "/"),
			Username:             parsed.User.Username(),
			Password:             password,
			SSLMode:              sslMode,
			Timezone:             "UTC",
			MaxIdleConns:         2,
			MaxOpenConns:         3,
			ConnMaxIdleTime:      50 * time.Millisecond,
			ConnMaxLifetime:      5 * time.Minute,
			ConnectTimeout:       2 * time.Second,
			LogLevel:             "silent",
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	assert.Equal(t, 3, sqlDB.Stats().MaxOpenConnections)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var applicationName string
	require.NoError(t, db.WithContext(ctx).Raw("SHOW application_name").Scan(&applicationName).Error)
	assert.Equal(t, "luas-database-test", applicationName)
	var timezone string
	require.NoError(t, db.WithContext(ctx).Raw("SHOW timezone").Scan(&timezone).Error)
	assert.Equal(t, "UTC", timezone)

	deadline := time.Now().Add(3 * time.Second)
	for sqlDB.Stats().MaxIdleTimeClosed == 0 && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	assert.Positive(t, sqlDB.Stats().MaxIdleTimeClosed, "idle connection policy was not observed")
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
			MaxIdleConns:         1,
			MaxOpenConns:         1,
			ConnMaxIdleTime:      15 * time.Minute,
			ConnMaxLifetime:      time.Hour,
			ConnectTimeout:       time.Second,
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database is enabled but unavailable")
	assert.Nil(t, db)
}

func TestDatabaseDiagnosticsDoNotCaptureBoundValues(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	collector := exception.NewCollector(nil)
	ctx := exception.WithCollector(context.Background(), collector)

	var result string
	require.NoError(t, db.WithContext(ctx).
		Raw("SELECT ? AS secret_value", "database-secret").
		Scan(&result).Error)

	queries := collector.SQL()
	require.NotEmpty(t, queries)
	statement := queries[len(queries)-1].Statement
	assert.NotContains(t, statement, "database-secret")
	assert.True(t, strings.Contains(statement, "?") || strings.Contains(statement, "$1"))
}

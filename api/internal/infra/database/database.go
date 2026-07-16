package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/infra/config"
)

// NewDB creates a new database connection via Wire DI.
// Returns nil only when database support is explicitly disabled in config.
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	if err := cfg.ValidateDatabase(); err != nil {
		return nil, err
	}
	if !cfg.Database.Enabled {
		log.Println("Database initialization skipped (DB_ENABLED=false)")
		return nil, nil
	}

	db, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("database is enabled but unavailable: %w", err)
	}
	return db, nil
}

// initDB initializes database connection with the given config
func initDB(cfg *config.Config) (*gorm.DB, error) {
	dbCfg := cfg.Database

	// Configure custom logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		buildLoggerConfig(cfg),
	)
	newLogger = wrapObservedLogger(newLogger)

	var dialector gorm.Dialector
	switch dbCfg.Driver {
	case "sqlite":
		dsn := dbCfg.Name
		if dbCfg.Memory {
			dsn = ":memory:"
		}
		dialector = sqlite.Open(dsn)
	case "postgres":
		dsn, dsnErr := postgresDSN(cfg)
		if dsnErr != nil {
			return nil, dsnErr
		}
		dialector = postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		})
	default:
		return nil, fmt.Errorf("DB_DRIVER must be one of postgres or sqlite")
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:               newLogger,
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Bound the pool before retaining idle connections. database/sql otherwise
	// defaults to unlimited open connections.
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(dbCfg.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(dbCfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(context.Background(), dbCfg.ConnectTimeout)
	defer cancel()
	if pingErr := sqlDB.PingContext(pingCtx); pingErr != nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			pingErr = errors.Join(pingErr, fmt.Errorf("close failed database pool: %w", closeErr))
		}
		return nil, fmt.Errorf("failed to ping database: %w", pingErr)
	}

	return db, nil
}

// NewTestDB creates an in-memory SQLite database for testing.
// This is a convenience function for tests that need a real database.
func NewTestDB() (*gorm.DB, error) {
	return initDB(&config.Config{
		App: config.AppConfig{
			Env:   "test",
			Debug: true,
		},
		Database: config.DatabaseConfig{
			Driver:               "sqlite",
			Memory:               true,
			MaxIdleConns:         1,
			MaxOpenConns:         1,
			ConnMaxIdleTime:      config.DefaultDatabaseConnMaxIdleTime,
			ConnMaxLifetime:      config.DefaultDatabaseConnMaxLifetime,
			ConnectTimeout:       config.DefaultDatabaseConnectTimeout,
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})
}

func postgresDSN(cfg *config.Config) (string, error) {
	if err := cfg.ValidateDatabase(); err != nil {
		return "", err
	}
	dbCfg := cfg.Database
	host := strings.TrimSuffix(strings.TrimPrefix(dbCfg.Host, "["), "]")
	applicationName := strings.TrimSpace(cfg.App.Name)
	if applicationName == "" {
		applicationName = "luas"
	}

	dsn := &url.URL{
		Scheme:  "postgres",
		User:    url.UserPassword(dbCfg.Username, dbCfg.Password),
		Host:    net.JoinHostPort(host, strconv.Itoa(dbCfg.Port)),
		Path:    "/" + dbCfg.Name,
		RawPath: "/" + url.PathEscape(dbCfg.Name),
	}
	parameters := url.Values{}
	parameters.Set("application_name", applicationName)
	parameters.Set("connect_timeout", strconv.FormatInt(timeoutSeconds(dbCfg.ConnectTimeout), 10))
	parameters.Set("sslmode", dbCfg.SSLMode)
	parameters.Set("timezone", dbCfg.Timezone)
	dsn.RawQuery = parameters.Encode()
	return dsn.String(), nil
}

func timeoutSeconds(timeout time.Duration) int64 {
	seconds := int64(timeout / time.Second)
	if timeout%time.Second != 0 {
		seconds++
	}
	return seconds
}

func buildLoggerConfig(cfg *config.Config) logger.Config {
	return logger.Config{
		SlowThreshold:             cfg.Database.SlowThreshold,
		LogLevel:                  resolveGormLogLevel(cfg),
		IgnoreRecordNotFoundError: cfg.Database.IgnoreRecordNotFound,
		Colorful:                  true,
		ParameterizedQueries:      true,
	}
}

func resolveGormLogLevel(cfg *config.Config) logger.LogLevel {
	fallback := logger.Warn
	if cfg.App.Debug || strings.EqualFold(cfg.App.Env, "test") {
		fallback = logger.Info
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Database.LogLevel)) {
	case "":
		return fallback
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return fallback
	}
}

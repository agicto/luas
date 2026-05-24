package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/zgiai/zgo/internal/infra/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB creates a new database connection via Wire DI.
// Returns nil if database is disabled in config.
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	if !cfg.Database.Enabled {
		log.Println("Database initialization skipped (DB_ENABLED=false)")
		return nil, nil
	}

	db, err := initDB(cfg)
	if err != nil {
		log.Printf("⚠️  Database unavailable: %v", err)
		log.Printf("⚠️  The application will start without database. Please configure DB_HOST/DB_PORT/DB_USERNAME/DB_PASSWORD/DB_NAME, or set DB_ENABLED=false to silence this warning.")
		return nil, nil
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

	if dbCfg.Driver == "sqlite" {
		dsn := dbCfg.Name
		if dbCfg.Memory {
			dsn = ":memory:"
		}
		dialector = sqlite.Open(dsn)
	} else {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s timezone=%s",
			dbCfg.Host,
			dbCfg.Username,
			dbCfg.Password,
			dbCfg.Name,
			dbCfg.Port,
			dbCfg.SSLMode,
			dbCfg.Timezone,
		)
		dialector = postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		})
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool
	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(dbCfg.ConnMaxLifetime)

	// Check if we can connect to the database
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
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
			SlowThreshold:        time.Second,
			IgnoreRecordNotFound: true,
		},
	})
}

func buildLoggerConfig(cfg *config.Config) logger.Config {
	return logger.Config{
		SlowThreshold:             cfg.Database.SlowThreshold,
		LogLevel:                  resolveGormLogLevel(cfg),
		IgnoreRecordNotFoundError: cfg.Database.IgnoreRecordNotFound,
		Colorful:                  true,
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

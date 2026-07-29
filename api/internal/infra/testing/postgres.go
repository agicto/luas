package testing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/infra/config"
)

const TestPostgresDSNEnv = "LUAS_TEST_POSTGRES_DSN"

var postgresResourceSequence atomic.Uint64

// OpenPostgres opens an isolated PostgreSQL schema and migrates the requested models.
func OpenPostgres(tb testing.TB, gormConfig *gorm.Config, models ...any) *gorm.DB {
	tb.Helper()
	db, cleanup := OpenPostgresWithCleanup(tb, gormConfig, models...)
	tb.Cleanup(cleanup)
	return db
}

// OpenPostgresWithCleanup opens an isolated schema that callers can release
// inside repeated property checks instead of retaining pools until test exit.
func OpenPostgresWithCleanup(
	tb testing.TB,
	gormConfig *gorm.Config,
	models ...any,
) (*gorm.DB, func()) {
	tb.Helper()
	parsed := testPostgresURL(tb)
	admin := openPostgres(tb, parsed.String(), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		DisableAutomaticPing: true,
	})
	schema := postgresResourceName("schema")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		closePostgres(tb, admin)
		tb.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	closePostgres(tb, admin)

	scoped := *parsed
	parameters := scoped.Query()
	parameters.Set("search_path", schema)
	parameters.Set("timezone", "UTC")
	scoped.RawQuery = parameters.Encode()
	if gormConfig == nil {
		gormConfig = &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	}
	db := openPostgres(tb, scoped.String(), gormConfig)
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			closePostgres(tb, db)
			cleanupAdmin := openPostgres(tb, parsed.String(), &gorm.Config{
				Logger:               logger.Default.LogMode(logger.Silent),
				DisableAutomaticPing: true,
			})
			if err := cleanupAdmin.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error; err != nil {
				tb.Errorf("drop isolated PostgreSQL schema: %v", err)
			}
			closePostgres(tb, cleanupAdmin)
		})
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			tb.Fatalf("migrate isolated PostgreSQL schema: %v", err)
		}
	}
	return db, cleanup
}

// CreatePostgresDatabase creates an isolated database for full application tests.
func CreatePostgresDatabase(tb testing.TB) config.DatabaseConfig {
	tb.Helper()
	parsed := testPostgresURL(tb)
	admin := openPostgres(tb, parsed.String(), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		DisableAutomaticPing: true,
	})
	databaseName := postgresResourceName("database")
	if err := admin.Exec(`CREATE DATABASE "` + databaseName + `"`).Error; err != nil {
		closePostgres(tb, admin)
		tb.Fatalf("create isolated PostgreSQL database: %v", err)
	}
	closePostgres(tb, admin)
	tb.Cleanup(func() {
		cleanupAdmin := openPostgres(tb, parsed.String(), &gorm.Config{
			Logger:               logger.Default.LogMode(logger.Silent),
			DisableAutomaticPing: true,
		})
		if err := cleanupAdmin.Exec(`DROP DATABASE IF EXISTS "` + databaseName + `" WITH (FORCE)`).Error; err != nil {
			tb.Errorf("drop isolated PostgreSQL database: %v", err)
		}
		closePostgres(tb, cleanupAdmin)
	})

	host := parsed.Hostname()
	port := 5432
	if rawPort := parsed.Port(); rawPort != "" {
		value, err := strconv.Atoi(rawPort)
		if err != nil {
			tb.Fatalf("parse PostgreSQL test port: %v", err)
		}
		port = value
	}
	username := parsed.User.Username()
	password, _ := parsed.User.Password()
	sslMode := parsed.Query().Get("sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}
	return config.DatabaseConfig{
		Enabled:              true,
		Driver:               "postgres",
		Host:                 host,
		Port:                 port,
		Name:                 databaseName,
		Username:             username,
		Password:             password,
		SSLMode:              sslMode,
		Timezone:             "UTC",
		MaxIdleConns:         2,
		MaxOpenConns:         4,
		ConnMaxIdleTime:      config.DefaultDatabaseConnMaxIdleTime,
		ConnMaxLifetime:      config.DefaultDatabaseConnMaxLifetime,
		ConnectTimeout:       config.DefaultDatabaseConnectTimeout,
		LogLevel:             "silent",
		SlowThreshold:        time.Second,
		IgnoreRecordNotFound: true,
	}
}

func testPostgresURL(tb testing.TB) *url.URL {
	tb.Helper()
	raw := strings.TrimSpace(os.Getenv(TestPostgresDSNEnv))
	if raw == "" {
		tb.Skip(TestPostgresDSNEnv + " is not set")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		tb.Fatalf("%s must be a PostgreSQL connection URI", TestPostgresDSNEnv)
	}
	if parsed.Hostname() == "" || strings.TrimPrefix(parsed.Path, "/") == "" {
		tb.Fatalf("%s must include a host and database name", TestPostgresDSNEnv)
	}
	if parsed.User == nil || parsed.User.Username() == "" {
		tb.Fatalf("%s must include a database username", TestPostgresDSNEnv)
	}
	return parsed
}

func openPostgres(tb testing.TB, dsn string, gormConfig *gorm.Config) *gorm.DB {
	tb.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), gormConfig)
	if err != nil {
		tb.Fatalf("open PostgreSQL test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Fatalf("resolve PostgreSQL test pool: %v", err)
	}
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxIdleTime(time.Minute)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		tb.Fatalf("ping PostgreSQL test database: %v", err)
	}
	return db
}

func closePostgres(tb testing.TB, db *gorm.DB) {
	tb.Helper()
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Errorf("resolve PostgreSQL test pool during cleanup: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		tb.Errorf("close PostgreSQL test pool: %v", err)
	}
}

func postgresResourceName(kind string) string {
	sequence := postgresResourceSequence.Add(1)
	return fmt.Sprintf("luas_%s_%d_%d_%d", kind, os.Getpid(), time.Now().UnixNano(), sequence)
}

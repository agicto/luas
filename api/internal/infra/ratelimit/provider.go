package ratelimit

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/infra/config"
)

// ProvideStores resolves the configured rate-limit backend for every HTTP limiter rule.
func ProvideStores(cfg *config.Config, db *gorm.DB) (*Stores, error) {
	if cfg == nil || cfg.Middleware.RateLimitStore.Driver != config.RateLimitStorePostgres {
		return NewMemoryStores(), nil
	}
	if db == nil {
		return nil, fmt.Errorf("RATE_LIMIT_STORE=postgres requires DB_ENABLED=true")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("shared rate-limit store: %w", err)
	}
	return NewPostgresStores(sqlDB, PostgresOptions{
		Timeout: cfg.Middleware.RateLimitStore.Timeout,
		Failure: FailurePolicy(cfg.Middleware.RateLimitStore.Failure),
	})
}

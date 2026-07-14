//go:build wireinject
// +build wireinject

package wiring

import (
	"github.com/google/wire"
	"github.com/zgiai/luas/api/internal/app"
	"github.com/zgiai/luas/api/internal/infra"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter"
)

// InitApplication initializes the entire application with all dependencies.
// This is the single entry point for Wire DI.
func InitApplication() (*app.Application, error) {
	wire.Build(
		// Infrastructure providers
		infra.ProviderSet,

		// Default starter providers
		starter.ProviderSet,

		// Build final application
		wire.Struct(new(app.Application), "*"),
	)
	return nil, nil
}

// InitApplicationWithConfig initializes the application from an already validated startup snapshot.
// Production bootstraps use it to share that snapshot with pre-Wire services; tests use it to reuse
// the same dependency graph without reading process configuration.
func InitApplicationWithConfig(cfg *config.Config) (*app.Application, error) {
	wire.Build(
		infra.ConfiguredProviderSet,
		starter.ProviderSet,
		wire.Struct(new(app.Application), "*"),
	)
	return nil, nil
}

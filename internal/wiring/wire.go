//go:build wireinject
// +build wireinject

package wiring

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/app"
	"github.com/zgiai/zgo/internal/infra"
	"github.com/zgiai/zgo/internal/modules/event"
	"github.com/zgiai/zgo/internal/modules/ingest"
	"github.com/zgiai/zgo/internal/modules/issue"
	"github.com/zgiai/zgo/internal/modules/permission"
	"github.com/zgiai/zgo/internal/modules/project"
	"github.com/zgiai/zgo/internal/modules/user"
)

// InitApplication initializes the entire application with all dependencies.
// This is the single entry point for Wire DI.
func InitApplication() (*app.Application, error) {
	wire.Build(
		// Infrastructure providers
		infra.ProviderSet,

		// Module providers
		user.ProviderSet,
		permission.ProviderSet,
		project.ProviderSet,
		event.ProviderSet,
		issue.ProviderSet,
		ingest.ProviderSet,

		// Bind event.Service to ingest.EventProcessor
		wire.Bind(new(ingest.EventProcessor), new(event.Service)),

		// Aggregate handlers
		wire.Struct(new(app.Handlers), "*"),

		// Build final application
		wire.Struct(new(app.Application), "*"),
	)
	return nil, nil
}

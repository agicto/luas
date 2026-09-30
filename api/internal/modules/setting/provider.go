package setting

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional typed setting starter.
var ProviderSet = wire.NewSet(
	NewDefaultCatalog,
	NewRepository,
	wire.Bind(new(settingStore), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.SettingReader), new(*service)),
	wire.Bind(new(domain.AppSettingWriter), new(*service)),
	NewHandler,
)

// NewStarterManifest describes setting routes, ownership dependencies, and persistence.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterSetting,
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit, config.StarterOrganization),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_07_15_040000_create_settings_table"),
	)
}

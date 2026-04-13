package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/platform"
	"gorm.io/gorm"
)

func init() {
	register("2026_04_10_000000_create_platform_tables", &createPlatformTables{})
}

type createPlatformTables struct {
	migration.BaseMigration
}

func (m *createPlatformTables) Up(db *gorm.DB) error {
	return db.AutoMigrate(
		&platform.GitHubConnectionPO{},
		&platform.ProjectPO{},
		&platform.ServicePO{},
		&platform.ServiceEnvVarPO{},
	)
}

func (m *createPlatformTables) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(
		"platform_service_env_vars",
		"platform_services",
		"platform_projects",
		"platform_github_connections",
	)
}

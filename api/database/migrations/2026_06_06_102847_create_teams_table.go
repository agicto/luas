package migrations

import (
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/team"
	"gorm.io/gorm"
)

func init() {
	register("2026_06_06_102847_create_teams_table", &createTeamsTable{})
}

// createTeamsTable creates the teams table.
type createTeamsTable struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *createTeamsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&team.TeamPO{}, &team.TeamMemberPO{})
}

// Down reverts the migration.
func (m *createTeamsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("team_members", "teams")
}

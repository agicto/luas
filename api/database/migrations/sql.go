package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// execStatements runs frozen DDL one statement at a time so every migration stays independent of
// module persistence structs and of GORM schema generation. A released migration is history: edit
// its SQL only to fix a defect that TestMigrationsProduceGoldenSchema proves is schema-neutral.
func execStatements(db *gorm.DB, statements ...string) error {
	for i, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("migration statement %d: %w", i+1, err)
		}
	}
	return nil
}

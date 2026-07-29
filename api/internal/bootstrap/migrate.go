// Package bootstrap provides application startup and initialization.
package bootstrap

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/starter"
)

// MigrationStatus represents the status of a migration.
type MigrationStatus struct {
	ID    string
	Ran   bool
	Batch int
	RanAt *time.Time
}

// getMigrator creates a new Migrator instance with the default starter migrations.
// If eventBus is nil, events will not be fired.
func getMigrator(db *gorm.DB, eventBus *events.EventBus) (*migration.Migrator, error) {
	// Create repository with default table name
	repo := migration.NewDatabaseRepository(db, "migrations")

	// Create migrator with optional event bus
	migrator := migration.NewMigrator(repo, db, eventBus)

	defaultMigrations, err := starter.DefaultMigrations()
	if err != nil {
		return nil, err
	}
	migrator.RegisterMany(defaultMigrations)

	return migrator, nil
}

// RunMigrations runs all pending migrations for the application.
func RunMigrations(db *gorm.DB) error {
	return RunMigrationsWithEvents(db, nil)
}

// RunMigrationsWithEvents runs all pending migrations with event bus integration.
func RunMigrationsWithEvents(db *gorm.DB, eventBus *events.EventBus) error {
	migrator, err := getMigrator(db, eventBus)
	if err != nil {
		log.Printf("Failed to load starter migrations: %v", err)
		return err
	}
	return RunRegisteredMigrations(migrator)
}

// RunRegisteredMigrations runs migrations already resolved by application starter assembly.
func RunRegisteredMigrations(migrator *migration.Migrator) error {
	if migrator == nil {
		return fmt.Errorf("configured migrator is required")
	}

	log.Println("Starting database migrations")
	startTime := time.Now()

	// Execute migrations
	executed, err := migrator.Run(migration.NewMigratorOptions())
	if err != nil {
		log.Printf("Migration failed: %v", err)
		return err
	}

	if len(executed) == 0 {
		log.Println("No pending migrations")
	} else {
		for _, name := range executed {
			log.Printf("Migrated: %s", name)
		}
	}

	log.Printf("Migration completed successfully in %v", time.Since(startTime))
	return nil
}

// RollbackLastMigration rolls back the last batch of migrations.
func RollbackLastMigration(db *gorm.DB) error {
	return RollbackLastMigrationWithEvents(db, nil)
}

// RollbackLastMigrationWithEvents rolls back the last batch with event bus integration.
func RollbackLastMigrationWithEvents(db *gorm.DB, eventBus *events.EventBus) error {
	log.Println("Rolling back last migration batch")

	migrator, err := getMigrator(db, eventBus)
	if err != nil {
		log.Printf("Failed to load starter migrations: %v", err)
		return err
	}

	opts := migration.NewRollbackOptions()
	rolledBack, err := migrator.Rollback(opts)
	if err != nil {
		log.Printf("Rollback failed: %v", err)
		return err
	}

	if len(rolledBack) == 0 {
		log.Println("No migrations to rollback")
	} else {
		for _, name := range rolledBack {
			log.Printf("Rolled back: %s", name)
		}
	}

	log.Println("Rollback completed successfully")
	return nil
}

// RollbackMigrations rolls back N migrations.
func RollbackMigrations(db *gorm.DB, steps int) error {
	return RollbackMigrationsWithEvents(db, steps, nil)
}

// RollbackMigrationsWithEvents rolls back N migrations with event bus integration.
func RollbackMigrationsWithEvents(db *gorm.DB, steps int, eventBus *events.EventBus) error {
	log.Printf("Rolling back %d migration(s)", steps)

	migrator, err := getMigrator(db, eventBus)
	if err != nil {
		log.Printf("Failed to load starter migrations: %v", err)
		return err
	}

	opts := migration.NewRollbackOptions().WithSteps(steps)
	rolledBack, err := migrator.Rollback(opts)
	if err != nil {
		log.Printf("Rollback failed: %v", err)
		return err
	}

	if len(rolledBack) == 0 {
		log.Println("No migrations to rollback")
	} else {
		for _, name := range rolledBack {
			log.Printf("Rolled back: %s", name)
		}
	}

	log.Printf("Rolled back %d migration(s) successfully", len(rolledBack))
	return nil
}

// ResetMigrations rolls back all migrations.
func ResetMigrations(db *gorm.DB) error {
	return ResetMigrationsWithEvents(db, nil)
}

// ResetMigrationsWithEvents rolls back all migrations with event bus integration.
func ResetMigrationsWithEvents(db *gorm.DB, eventBus *events.EventBus) error {
	log.Println("Resetting all migrations")

	migrator, err := getMigrator(db, eventBus)
	if err != nil {
		log.Printf("Failed to load starter migrations: %v", err)
		return err
	}

	rolledBack, err := migrator.Reset(false)
	if err != nil {
		log.Printf("Reset failed: %v", err)
		return err
	}

	if len(rolledBack) == 0 {
		log.Println("No migrations to reset")
	} else {
		for _, name := range rolledBack {
			log.Printf("Rolled back: %s", name)
		}
	}

	log.Printf("Reset completed (%d migrations)", len(rolledBack))
	return nil
}

// GetMigrationStatus returns the status of all migrations.
func GetMigrationStatus(db *gorm.DB) ([]MigrationStatus, error) {
	migrator, err := getMigrator(db, nil)
	if err != nil {
		return nil, err
	}
	repo := migrator.Repository()

	// Ensure repository exists
	if !repo.RepositoryExists() {
		// Return all migrations as pending
		registered := migrator.GetRegisteredMigrations()
		statuses := make([]MigrationStatus, 0, len(registered))
		for i, name := range registered {
			statuses = append(statuses, MigrationStatus{
				ID:    name,
				Ran:   false,
				Batch: i + 1,
			})
		}
		return statuses, nil
	}

	// Get migration batches
	batches, err := repo.GetMigrationBatches()
	if err != nil {
		return nil, err
	}

	// Get all registered migrations
	registered := migrator.GetRegisteredMigrations()

	// Build status list
	statuses := make([]MigrationStatus, 0, len(registered))
	for _, name := range registered {
		batch, ran := batches[name]
		statuses = append(statuses, MigrationStatus{
			ID:    name,
			Ran:   ran,
			Batch: batch,
		})
	}

	return statuses, nil
}

// DropAllTables drops all tables in the database.
func DropAllTables(db *gorm.DB) error {
	log.Println("Dropping all tables")

	var tables []string
	if err := db.Raw(`
		SELECT tablename FROM pg_tables
		WHERE schemaname = current_schema()
	`).Scan(&tables).Error; err != nil {
		return err
	}

	for _, table := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS \"" + table + "\" CASCADE").Error; err != nil {
			log.Printf("Failed to drop table %s: %v", table, err)
		}
	}

	log.Println("All tables dropped")
	return nil
}

// FreshMigrations drops all tables and re-runs all migrations.
func FreshMigrations(db *gorm.DB) error {
	return FreshMigrationsWithEvents(db, nil)
}

// FreshMigrationsWithEvents drops all tables and re-runs all migrations with event bus.
func FreshMigrationsWithEvents(db *gorm.DB, eventBus *events.EventBus) error {
	migrator, err := getMigrator(db, eventBus)
	if err != nil {
		log.Printf("Failed to load starter migrations: %v", err)
		return err
	}

	opts := migration.NewMigratorOptions()
	executed, err := migrator.Fresh(opts)
	if err != nil {
		return err
	}

	if len(executed) > 0 {
		for _, name := range executed {
			log.Printf("Migrated: %s", name)
		}
	}

	return nil
}

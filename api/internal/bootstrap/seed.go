package bootstrap

import (
	"log"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/database/seeders"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter"
)

// RunSeeders runs all registered database seeders with the given database connection
func RunSeeders(db *gorm.DB) error {
	defaultSeeders, err := starter.DefaultSeeders()
	if err != nil {
		log.Printf("Failed to load starter seeders: %v", err)
		return err
	}
	return runSeeders(db, defaultSeeders)
}

// RunConfiguredSeeders runs seeders from the same starter selection as HTTP and migrations.
func RunConfiguredSeeders(db *gorm.DB, cfg *config.Config) error {
	configuredSeeders, err := starter.ConfiguredSeeders(cfg)
	if err != nil {
		log.Printf("Failed to load configured starter seeders: %v", err)
		return err
	}
	return runSeeders(db, configuredSeeders)
}

func runSeeders(db *gorm.DB, selected []seeders.Seeder) error {
	log.Println("Running active starter seeders")

	for _, seeder := range selected {
		if err := seeder.Run(db); err != nil {
			log.Printf("Seeder failed: %v", err)
			return err
		}
	}

	log.Printf("Successfully ran %d seeders", len(selected))
	return nil
}

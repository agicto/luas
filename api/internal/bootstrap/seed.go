package bootstrap

import (
	"log"

	_ "github.com/zgiai/zgo/database/seeders" // Import to trigger init()
	"github.com/zgiai/zgo/internal/starter"
	"gorm.io/gorm"
)

// RunSeeders runs all registered database seeders with the given database connection
func RunSeeders(db *gorm.DB) error {
	log.Println("Running default starter seeders")

	defaultSeeders, err := starter.DefaultSeeders()
	if err != nil {
		log.Printf("Failed to load starter seeders: %v", err)
		return err
	}

	for _, seeder := range defaultSeeders {
		if err := seeder.Run(db); err != nil {
			log.Printf("Seeder failed: %v", err)
			return err
		}
	}

	log.Printf("Successfully ran %d seeders", len(defaultSeeders))
	return nil
}

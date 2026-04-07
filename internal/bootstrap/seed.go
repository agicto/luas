package bootstrap

import (
	"log"

	"github.com/zgiai/zgo/database/seeders"
	_ "github.com/zgiai/zgo/database/seeders" // Import to trigger init()
	"gorm.io/gorm"
)

// RunSeeders runs all registered database seeders with the given database connection
func RunSeeders(db *gorm.DB) error {
	log.Println("Running default database seeders")

	if err := seeders.RunDefault(db); err != nil {
		log.Printf("Seeder failed: %v", err)
		return err
	}

	log.Printf("Successfully ran %d seeders", len(seeders.Default()))
	return nil
}

package seeders

import "gorm.io/gorm"

// Seeder interface defines the contract for database seeders
type Seeder interface {
	Name() string
	Run(db *gorm.DB) error
}

var registry []Seeder

var defaultExcluded = map[string]struct{}{
	"roles": {},
}

// register adds a seeder to the registry
func register(s Seeder) {
	registry = append(registry, s)
}

// All returns all registered seeders
func All() []Seeder {
	return registry
}

// Default returns the legacy default scaffold seeder set.
// Deprecated: starter.DefaultSeeders() is the canonical source for default starter assembly.
func Default() []Seeder {
	filtered := make([]Seeder, 0, len(registry))
	for _, seeder := range registry {
		if _, excluded := defaultExcluded[seeder.Name()]; excluded {
			continue
		}
		filtered = append(filtered, seeder)
	}
	return filtered
}

// RunAll executes all registered seeders with the given database connection
func RunAll(db *gorm.DB) error {
	for _, seeder := range registry {
		if err := seeder.Run(db); err != nil {
			return err
		}
	}
	return nil
}

// RunDefault executes the legacy default scaffold seeder set.
// Deprecated: starter.DefaultSeeders() is the canonical source for default starter assembly.
func RunDefault(db *gorm.DB) error {
	for _, seeder := range Default() {
		if err := seeder.Run(db); err != nil {
			return err
		}
	}
	return nil
}

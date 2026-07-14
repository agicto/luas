package main

import (
	"log"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter"
	"github.com/zgiai/luas/api/internal/wiring"
)

func main() {
	// 1. Load and validate the process configuration snapshot.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if validationErr := starter.ValidateConfig(cfg); validationErr != nil {
		log.Fatalf("Failed to resolve starter configuration: %v", validationErr)
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		log.Fatalf("Failed to initialize logger: %v", loggerErr)
	}

	// 2. Initialize the application with that same snapshot.
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// 3. Create HTTP Kernel and Start Server
	kernel := bootstrap.NewHttpKernel(application)
	kernel.Handle()
}

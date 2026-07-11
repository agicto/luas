package main

import (
	"log"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/wiring"
)

func main() {
	// 1. Initialize Logger
	bootstrap.InitLogger()

	// 2. Initialize Application via Wire DI
	application, err := wiring.InitApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// 3. Create HTTP Kernel and Start Server
	kernel := bootstrap.NewHttpKernel(application)
	kernel.Handle()
}

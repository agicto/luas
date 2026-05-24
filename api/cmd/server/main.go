package main

import (
	"log"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/wiring"
)

// @title Luas API
// @version 1.0
// @description A modular Go API scaffold for production-ready services
// @host localhost:8025
// @BasePath /v1

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

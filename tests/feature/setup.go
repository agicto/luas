package feature

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/bootstrap"
	"github.com/zgiai/zgo/internal/infra/config"
	test_platform "github.com/zgiai/zgo/internal/infra/testing"
	"github.com/zgiai/zgo/internal/wiring"
)

// SetupApp initializes the feature-test application by reusing the production DI graph
// and HTTP startup chain with a test-specific config.
func SetupApp() *gin.Engine {
	cfg := &config.Config{}
	cfg.App.Name = "ZGO Test"
	cfg.App.Env = "test"
	cfg.App.Debug = false
	cfg.App.URL = "http://localhost:0"
	cfg.Server.Mode = "test"
	cfg.Database.Enabled = true
	cfg.Database.Driver = "sqlite"
	cfg.Database.Memory = true
	cfg.Database.MaxIdleConns = 1
	cfg.Database.MaxOpenConns = 1
	cfg.JWT.Secret = "testing-secret"
	cfg.JWT.Expire = time.Hour
	cfg.JWT.ExpireDays = 1
	cfg.CORS.AllowOrigins = []string{"*"}
	cfg.CORS.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	cfg.CORS.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Request-ID"}
	cfg.CORS.ExposeHeaders = []string{"Content-Length", "X-Request-ID"}
	cfg.CORS.AllowCredentials = true
	cfg.AI.DefaultProvider = "openai"
	cfg.AI.DefaultModel = "gpt-5.4"

	if _, err := config.Use(cfg); err != nil {
		panic("failed to register test config: " + err.Error())
	}

	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		panic("failed to init test application: " + err.Error())
	}

	if err := bootstrap.RunMigrationsWithEvents(application.DB, application.EventBus); err != nil {
		panic("failed to run migrations for test db: " + err.Error())
	}

	kernel := bootstrap.NewHttpKernel(application)
	return kernel.Engine
}

// NewTestCase is a shortcut to create a test case with the setup app
func NewTestCase(t *testing.T) *test_platform.TestCase {
	engine := SetupApp()
	return test_platform.NewTestCase(t, engine)
}
